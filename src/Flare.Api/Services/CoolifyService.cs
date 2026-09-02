using System.Globalization;
using System.Net.Http.Headers;
using System.Text.Json;
using System.Text.RegularExpressions;
using Flare.Api.Configuration;
using Flare.Api.Infrastructure;
using Flare.Contracts;
using Microsoft.Extensions.Options;

namespace Flare.Api.Services;

public sealed partial class CoolifyService : ICoolifyService
{
    private readonly HttpClient _client;
    private readonly CoolifyOptions _options;
    private readonly ILogger<CoolifyService> _logger;

    public CoolifyService(HttpClient client, IOptions<CoolifyOptions> options, ILogger<CoolifyService> logger)
    {
        _client = client;
        _options = options.Value;
        _logger = logger;
        if (_options.IsConfigured)
        {
            _client.BaseAddress = new Uri($"{_options.BaseUrl!.TrimEnd('/')}/api/v1/");
            _client.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Bearer", _options.ApiToken);
        }
    }

    public bool IsConfigured => _options.IsConfigured;

    public async Task<IReadOnlyList<CoolifyServerResponse>> GetServersAsync(CancellationToken cancellationToken)
    {
        using var document = await GetAsync("servers", cancellationToken);
        return Elements(document.RootElement).Select(element => new CoolifyServerResponse(
            String(element, "uuid") ?? string.Empty,
            String(element, "name") ?? "Unnamed server",
            NestedBoolean(element, "settings", "is_reachable"),
            NestedBoolean(element, "settings", "is_usable"))).ToArray();
    }

    public async Task<IReadOnlyList<CoolifyResourceResponse>> GetServerResourcesAsync(
        string serverUuid, CancellationToken cancellationToken)
    {
        ValidateUuid(serverUuid);
        using var document = await GetAsync($"servers/{Uri.EscapeDataString(serverUuid)}/resources", cancellationToken);
        return Elements(document.RootElement).Select(element => new CoolifyResourceResponse(
            String(element, "uuid") ?? string.Empty,
            String(element, "name") ?? "Unnamed resource",
            String(element, "type") ?? "unknown",
            String(element, "status"))).ToArray();
    }

    public async Task<IReadOnlyList<CoolifyApplicationResponse>> GetApplicationsAsync(CancellationToken cancellationToken)
    {
        using var document = await GetAsync("applications", cancellationToken);
        return Elements(document.RootElement).Select(element => new CoolifyApplicationResponse(
            String(element, "uuid") ?? string.Empty,
            String(element, "name") ?? "Unnamed application",
            String(element, "status"),
            String(element, "fqdn"),
            String(element, "git_branch") ?? String(element, "branch"))).ToArray();
    }

    public async Task<IReadOnlyList<CoolifyServiceResponse>> GetServicesAsync(CancellationToken cancellationToken)
    {
        using var document = await GetAsync("services", cancellationToken);
        return Elements(document.RootElement).Select(element => new CoolifyServiceResponse(
            String(element, "uuid") ?? string.Empty,
            String(element, "name") ?? "Unnamed service",
            String(element, "status"),
            String(element, "description"))).ToArray();
    }

    public async Task<PagedResponse<DeploymentResponse>> GetDeploymentsAsync(
        int page, int pageSize, CancellationToken cancellationToken)
    {
        page = Math.Max(1, page);
        pageSize = Math.Clamp(pageSize, 1, 50);
        var applications = await GetApplicationsAsync(cancellationToken);
        var takePerApplication = Math.Clamp(page * pageSize + 1, 1, 100);
        var deploymentBatches = new List<DeploymentResponse>();
        foreach (var batch in applications.Chunk(6))
        {
            var tasks = batch.Select(application =>
                GetApplicationDeploymentsAsync(application.Uuid, takePerApplication, cancellationToken));
            deploymentBatches.AddRange((await Task.WhenAll(tasks)).SelectMany(items => items));
        }
        var deployments = deploymentBatches
            .OrderByDescending(item => item.StartedAt ?? DateTimeOffset.MinValue)
            .ToArray();
        var offset = (page - 1) * pageSize;
        var items = deployments.Skip(offset).Take(pageSize).ToArray();
        return new PagedResponse<DeploymentResponse>(items, page, pageSize, deployments.Length > offset + items.Length);
    }

    public async Task<DeploymentResponse?> GetDeploymentAsync(string uuid, CancellationToken cancellationToken)
    {
        ValidateUuid(uuid);
        try
        {
            using var document = await GetAsync($"deployments/{Uri.EscapeDataString(uuid)}", cancellationToken);
            return MapDeployment(document.RootElement);
        }
        catch (CoolifyNotFoundException)
        {
            return null;
        }
    }

    public Task<ActionResponse> StartApplicationAsync(string uuid, CancellationToken cancellationToken) =>
        PostActionAsync($"applications/{Safe(uuid)}/start", cancellationToken);

    public Task<ActionResponse> StopApplicationAsync(string uuid, CancellationToken cancellationToken) =>
        PostActionAsync($"applications/{Safe(uuid)}/stop", cancellationToken);

    public Task<ActionResponse> RestartApplicationAsync(string uuid, CancellationToken cancellationToken) =>
        PostActionAsync($"applications/{Safe(uuid)}/restart", cancellationToken);

    public Task<ActionResponse> RestartServiceAsync(string uuid, CancellationToken cancellationToken) =>
        PostActionAsync($"services/{Safe(uuid)}/restart?latest=false", cancellationToken);

    public Task<ActionResponse> RedeployApplicationAsync(string uuid, CancellationToken cancellationToken) =>
        PostActionAsync($"deploy?uuid={Safe(uuid)}&force=false", cancellationToken);

    private async Task<IReadOnlyList<DeploymentResponse>> GetApplicationDeploymentsAsync(
        string applicationUuid, int take, CancellationToken cancellationToken)
    {
        ValidateUuid(applicationUuid);
        using var document = await GetAsync(
            $"deployments/applications/{Uri.EscapeDataString(applicationUuid)}?skip=0&take={take}", cancellationToken);
        return Elements(document.RootElement).Select(element =>
        {
            var deployment = MapDeployment(element);
            return string.IsNullOrWhiteSpace(deployment.ResourceUuid)
                ? deployment with { ResourceUuid = applicationUuid }
                : deployment;
        }).ToArray();
    }

    private async Task<JsonDocument> GetAsync(string path, CancellationToken cancellationToken)
    {
        EnsureConfigured();
        try
        {
            using var response = await _client.GetAsync(path, HttpCompletionOption.ResponseHeadersRead, cancellationToken);
            if (response.StatusCode == System.Net.HttpStatusCode.NotFound)
            {
                throw new CoolifyNotFoundException();
            }
            if (!response.IsSuccessStatusCode)
            {
                LogRejectedRequest(_logger, path, (int)response.StatusCode);
                throw new InfrastructureUnavailableException("Coolify rejected the request.");
            }
            await using var stream = await response.Content.ReadAsStreamAsync(cancellationToken);
            return await JsonDocument.ParseAsync(stream, cancellationToken: cancellationToken);
        }
        catch (Exception exception) when (exception is CoolifyNotFoundException or InfrastructureUnavailableException)
        {
            throw;
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            LogRequestFailure(_logger, exception);
            throw new InfrastructureUnavailableException("Coolify is unavailable.", exception);
        }
    }

    private async Task<ActionResponse> PostActionAsync(string path, CancellationToken cancellationToken)
    {
        EnsureConfigured();
        try
        {
            using var response = await _client.PostAsync(path, content: null, cancellationToken);
            if (!response.IsSuccessStatusCode)
            {
                LogRejectedAction(_logger, path.Split('?')[0], (int)response.StatusCode);
                throw new InfrastructureUnavailableException("Coolify rejected the operation.");
            }
            var payload = await response.Content.ReadAsStringAsync(cancellationToken);
            if (string.IsNullOrWhiteSpace(payload)) return new ActionResponse("Operation queued.");
            using var document = JsonDocument.Parse(payload);
            var root = document.RootElement;
            var operationId = String(root, "deployment_uuid");
            if (operationId is null && root.TryGetProperty("deployments", out var deployments)
                && deployments.ValueKind == JsonValueKind.Array && deployments.GetArrayLength() > 0)
            {
                operationId = String(deployments[0], "deployment_uuid");
            }
            return new ActionResponse(String(root, "message") ?? "Operation queued.", operationId);
        }
        catch (InfrastructureUnavailableException)
        {
            throw;
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            LogActionFailure(_logger, exception);
            throw new InfrastructureUnavailableException("Coolify is unavailable.", exception);
        }
    }

    private static DeploymentResponse MapDeployment(JsonElement element)
    {
        var started = Date(element, "created_at") ?? Date(element, "started_at");
        var finished = Date(element, "finished_at") ?? Date(element, "updated_at");
        var status = String(element, "status");
        if (status is "in_progress" or "queued") finished = null;
        return new DeploymentResponse(
            String(element, "deployment_uuid") ?? String(element, "uuid") ?? string.Empty,
            String(element, "application_uuid") ?? String(element, "resource_uuid") ?? string.Empty,
            String(element, "application_name") ?? String(element, "resource_name") ?? "Application",
            status,
            String(element, "git_branch") ?? String(element, "branch"),
            String(element, "commit"),
            String(element, "commit_message"),
            started,
            finished,
            started is not null && finished is not null ? finished - started : null,
            Truncate(String(element, "logs"), 1_000_000));
    }

    private void EnsureConfigured()
    {
        if (!IsConfigured)
        {
            throw new InfrastructureUnavailableException("Coolify integration is not configured.");
        }
    }

    private static JsonElement[] Elements(JsonElement root) =>
        root.ValueKind == JsonValueKind.Array ? root.EnumerateArray().ToArray() : [];

    private static string? String(JsonElement element, string property) =>
        element.TryGetProperty(property, out var value) && value.ValueKind is not JsonValueKind.Null
            ? value.ValueKind == JsonValueKind.String ? value.GetString() : value.ToString()
            : null;

    private static DateTimeOffset? Date(JsonElement element, string property) =>
        DateTimeOffset.TryParse(String(element, property), CultureInfo.InvariantCulture,
            DateTimeStyles.AssumeUniversal, out var parsed) ? parsed : null;

    private static string? Truncate(string? value, int maximumLength) => value is { Length: > 0 }
        ? value.Length <= maximumLength ? value : value[..maximumLength]
        : value;

    private static bool? NestedBoolean(JsonElement element, string parent, string property) =>
        element.TryGetProperty(parent, out var nested) && nested.TryGetProperty(property, out var value)
            && value.ValueKind is JsonValueKind.True or JsonValueKind.False ? value.GetBoolean() : null;

    private static string Safe(string uuid)
    {
        ValidateUuid(uuid);
        return Uri.EscapeDataString(uuid);
    }

    private static void ValidateUuid(string uuid)
    {
        if (!UuidRegex().IsMatch(uuid)) throw new ArgumentException("Coolify identifier is invalid.", nameof(uuid));
    }

    [GeneratedRegex("^[A-Za-z0-9_-]{1,128}$", RegexOptions.CultureInvariant)]
    private static partial Regex UuidRegex();

    private sealed class CoolifyNotFoundException : Exception;

    [LoggerMessage(LogLevel.Warning, "Coolify request to {Path} failed with status {StatusCode}.")]
    private static partial void LogRejectedRequest(ILogger logger, string path, int statusCode);

    [LoggerMessage(LogLevel.Warning, "Coolify API request failed.")]
    private static partial void LogRequestFailure(ILogger logger, Exception exception);

    [LoggerMessage(LogLevel.Warning, "Coolify action {Path} failed with status {StatusCode}.")]
    private static partial void LogRejectedAction(ILogger logger, string path, int statusCode);

    [LoggerMessage(LogLevel.Warning, "Coolify API action failed.")]
    private static partial void LogActionFailure(ILogger logger, Exception exception);
}
