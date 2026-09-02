using System.Security.Authentication;

namespace Flare.Mobile.Services;

public sealed class ConnectionService(HttpClient httpClient, SessionStore sessionStore)
{
    public async Task<(bool Success, string Message)> ConnectAsync(string input, CancellationToken cancellationToken)
    {
        const bool allowHttpLoopback = false;
        if (!ServerUrlValidator.TryNormalize(input, allowHttpLoopback, out var url, out var error))
        {
            return (false, error);
        }
        try
        {
            using var response = await httpClient.GetAsync($"{url}/health/ready", cancellationToken);
            if (!response.IsSuccessStatusCode)
            {
                return (false, "Flare server is reachable but not ready.");
            }
            sessionStore.SaveServerUrl(url);
            return (true, "Connected");
        }
        catch (TaskCanceledException) when (!cancellationToken.IsCancellationRequested)
        {
            return (false, "Flare server timed out.");
        }
        catch (HttpRequestException exception) when (exception.InnerException is AuthenticationException)
        {
            return (false, "Flare server TLS validation failed.");
        }
        catch (HttpRequestException)
        {
            return (false, "Flare server unreachable.");
        }
    }
}
