using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Security.Authentication;
using System.Text.Json;
using System.Text.Json.Serialization;
using Flare.Contracts;

namespace Flare.Mobile.Services;

public sealed class ApiClient(HttpClient httpClient, SessionStore sessionStore) : IDisposable
{
    private readonly SemaphoreSlim _refreshLock = new(1, 1);
    private readonly JsonSerializerOptions _json = new(JsonSerializerDefaults.Web)
    {
        Converters = { new JsonStringEnumConverter() }
    };

    public Task<T> GetAsync<T>(string path, CancellationToken cancellationToken) =>
        SendAsync<T>(HttpMethod.Get, path, null, true, true, cancellationToken);

    public Task<T> PostAsync<T>(string path, object? body, bool authenticated, CancellationToken cancellationToken) =>
        SendAsync<T>(HttpMethod.Post, path, body, authenticated, authenticated, cancellationToken);

    public async Task PostAsync(string path, object? body, CancellationToken cancellationToken) =>
        _ = await SendAsync<JsonElement>(HttpMethod.Post, path, body, true, true, cancellationToken);

    public async Task<string?> GetValidAccessTokenAsync(CancellationToken cancellationToken)
    {
        var expiry = await sessionStore.GetAccessExpiryAsync();
        if (expiry <= DateTimeOffset.UtcNow.AddMinutes(1))
        {
            await RefreshAsync(force: false, cancellationToken);
        }
        return await sessionStore.GetAccessTokenAsync();
    }

    private async Task<T> SendAsync<T>(HttpMethod method, string path, object? body, bool authenticated,
        bool retryAfterRefresh, CancellationToken cancellationToken)
    {
        using var response = await SendCoreAsync(method, path, body, authenticated, cancellationToken);
        if (response.StatusCode == HttpStatusCode.Unauthorized && retryAfterRefresh)
        {
            await RefreshAsync(force: true, cancellationToken);
            using var retry = await SendCoreAsync(method, path, body, authenticated, cancellationToken);
            return await ReadAsync<T>(retry, cancellationToken);
        }
        return await ReadAsync<T>(response, cancellationToken);
    }

    private async Task<HttpResponseMessage> SendCoreAsync(HttpMethod method, string path, object? body,
        bool authenticated, CancellationToken cancellationToken)
    {
        var serverUrl = sessionStore.ServerUrl ?? throw new FlareApiException("Connect to a Flare server first.");
        using var request = new HttpRequestMessage(method, $"{serverUrl}/{path.TrimStart('/')}");
        if (body is not null) request.Content = JsonContent.Create(body, options: _json);
        if (authenticated)
        {
            var token = await sessionStore.GetAccessTokenAsync();
            if (string.IsNullOrWhiteSpace(token)) throw new FlareApiException("Sign in to continue.", HttpStatusCode.Unauthorized);
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);
        }
        try
        {
            return await httpClient.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, cancellationToken);
        }
        catch (TaskCanceledException) when (!cancellationToken.IsCancellationRequested)
        {
            throw new FlareApiException("Flare server timed out.");
        }
        catch (HttpRequestException exception) when (exception.InnerException is AuthenticationException)
        {
            throw new FlareApiException("Flare server TLS validation failed.");
        }
        catch (HttpRequestException)
        {
            throw new FlareApiException("Flare server unreachable.");
        }
    }

    private async Task RefreshAsync(bool force, CancellationToken cancellationToken)
    {
        await _refreshLock.WaitAsync(cancellationToken);
        try
        {
            var expiry = await sessionStore.GetAccessExpiryAsync();
            if (!force && expiry > DateTimeOffset.UtcNow.AddMinutes(1)) return;
            var refreshToken = await sessionStore.GetRefreshTokenAsync();
            if (string.IsNullOrWhiteSpace(refreshToken)) throw new FlareApiException("Session expired. Sign in again.");
            using var response = await SendCoreAsync(HttpMethod.Post, "api/v1/auth/refresh",
                new RefreshRequest(refreshToken), false, cancellationToken);
            var tokens = await ReadAsync<TokenResponse>(response, cancellationToken);
            await sessionStore.SaveTokensAsync(tokens);
        }
        catch (FlareApiException exception) when (exception.StatusCode == HttpStatusCode.Unauthorized)
        {
            sessionStore.ClearAuthentication();
            throw new FlareApiException("Session expired. Sign in again.", HttpStatusCode.Unauthorized);
        }
        finally
        {
            _refreshLock.Release();
        }
    }

    private async Task<T> ReadAsync<T>(HttpResponseMessage response, CancellationToken cancellationToken)
    {
        if (response.IsSuccessStatusCode)
        {
            if (response.StatusCode == HttpStatusCode.NoContent) return default!;
            return await response.Content.ReadFromJsonAsync<T>(_json, cancellationToken)
                ?? throw new FlareApiException("Flare returned an empty response.");
        }
        ApiProblem? problem = null;
        try { problem = await response.Content.ReadFromJsonAsync<ApiProblem>(_json, cancellationToken); }
        catch (JsonException) { }
        var message = response.StatusCode switch
        {
            HttpStatusCode.Forbidden => "You do not have permission to perform this action.",
            HttpStatusCode.TooManyRequests => "Too many requests. Wait a moment and try again.",
            HttpStatusCode.Unauthorized => "Session expired. Sign in again.",
            >= HttpStatusCode.InternalServerError => "Flare server could not complete the request.",
            _ => problem?.Detail ?? problem?.Title ?? "The request could not be completed."
        };
        throw new FlareApiException(message, response.StatusCode);
    }

    private sealed record ApiProblem(string? Title, string? Detail);
    public void Dispose() => _refreshLock.Dispose();
}

public sealed class FlareApiException(string message, HttpStatusCode? statusCode = null) : Exception(message)
{
    public HttpStatusCode? StatusCode { get; } = statusCode;
}
