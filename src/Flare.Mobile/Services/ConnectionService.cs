using System.Net;
using System.Security.Authentication;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Services;

public sealed class ConnectionService(
    HttpClient httpClient,
    IServerUrlStore sessionStore,
    ILogger<ConnectionService> logger)
{
    public async Task<ConnectionResult> ConnectAsync(string input, CancellationToken cancellationToken)
    {
        const bool allowHttpLoopback = false;
        if (!ServerUrlValidator.TryNormalize(input, allowHttpLoopback, out var url, out var error))
        {
            ConnectionLog.InvalidUrl(logger);
            return new(ConnectionOutcome.InvalidUrl, error);
        }

        ConnectionLog.ProbeStarted(logger, url);

        HttpResponseMessage response;
        try
        {
            // Buffer the small health response before resuming on Android's UI context. Disposing a
            // headers-only Android response can synchronously drain its stream and violate StrictMode.
            response = await httpClient.GetAsync($"{url}/health/ready", cancellationToken)
                .ConfigureAwait(false);
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            ConnectionLog.ProbeCanceled(logger, url);
            throw;
        }
        catch (TaskCanceledException) when (!cancellationToken.IsCancellationRequested)
        {
            ConnectionLog.ProbeTimedOut(logger, url);
            return new(ConnectionOutcome.Timeout, "Flare server timed out.");
        }
        catch (WebException exception) when (IsTlsFailure(exception))
        {
            ConnectionLog.TlsFailed(logger, url, exception);
            return new(ConnectionOutcome.TlsError, "Flare server TLS validation failed.");
        }
        catch (WebException exception) when (exception.Status == WebExceptionStatus.Timeout)
        {
            ConnectionLog.ProbeTimedOut(logger, url, exception);
            return new(ConnectionOutcome.Timeout, "Flare server timed out.");
        }
        catch (WebException exception) when (IsRedirectFailure(exception))
        {
            ConnectionLog.InvalidRedirect(logger, url, exception);
            return new(ConnectionOutcome.InvalidResponse,
                "Flare server returned an invalid redirect response. Check its public URL and proxy routing.");
        }
        catch (WebException exception)
        {
            ConnectionLog.ProbeFailed(logger, url, exception);
            return new(ConnectionOutcome.Unreachable, "Flare server unreachable.");
        }
        catch (HttpRequestException exception) when (IsTlsFailure(exception))
        {
            ConnectionLog.TlsFailed(logger, url, exception);
            return new(ConnectionOutcome.TlsError, "Flare server TLS validation failed.");
        }
        catch (HttpRequestException exception)
        {
            ConnectionLog.ProbeFailed(logger, url, exception);
            return new(ConnectionOutcome.Unreachable, "Flare server unreachable.");
        }
        catch (Exception exception)
        {
            ConnectionLog.UnexpectedProbeFailure(logger, url, exception);
            return new(ConnectionOutcome.Unreachable, "Flare server unreachable.");
        }

        try
        {
            ConnectionLog.HealthResponse(logger, url, (int)response.StatusCode);

            if (!response.IsSuccessStatusCode)
            {
                return new(ConnectionOutcome.NotReady,
                    $"Flare server is reachable but not ready (HTTP {(int)response.StatusCode}).");
            }

            try
            {
                sessionStore.SaveServerUrl(url);
                ConnectionLog.ServerUrlStored(logger, url);
                return new(ConnectionOutcome.Connected, "Connected");
            }
            catch (Exception exception)
            {
                ConnectionLog.ServerUrlStorageFailed(logger, url, exception);
                return new(ConnectionOutcome.StorageError,
                    "Flare connected, but the server address could not be saved on this device.");
            }
        }
        finally
        {
            try
            {
                response.Dispose();
            }
            catch (Exception exception)
            {
                // The response body has already been buffered. Cleanup failure must not invalidate a
                // successful readiness probe or terminate Android's main thread.
                ConnectionLog.ResponseDisposeFailed(logger, url, exception);
            }
        }
    }

    private static bool IsRedirectFailure(WebException exception) =>
        exception.Message.Contains("redirect", StringComparison.OrdinalIgnoreCase);

    private static bool IsTlsFailure(Exception exception)
    {
        for (Exception? current = exception; current is not null; current = current.InnerException)
        {
            if (current is AuthenticationException)
            {
                return true;
            }

            if (current is WebException webException
                && webException.Status is WebExceptionStatus.TrustFailure or WebExceptionStatus.SecureChannelFailure)
            {
                return true;
            }
        }

        return false;
    }
}
