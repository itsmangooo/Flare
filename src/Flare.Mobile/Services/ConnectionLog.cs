using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Services;

internal static partial class ConnectionLog
{
    [LoggerMessage(1000, LogLevel.Information, "Flare connection rejected because the server URL is invalid.")]
    public static partial void InvalidUrl(ILogger logger);

    [LoggerMessage(1001, LogLevel.Information, "Starting Flare readiness probe for {ServerOrigin}.")]
    public static partial void ProbeStarted(ILogger logger, string serverOrigin);

    [LoggerMessage(1002, LogLevel.Debug, "Flare readiness probe for {ServerOrigin} was canceled.")]
    public static partial void ProbeCanceled(ILogger logger, string serverOrigin);

    [LoggerMessage(1003, LogLevel.Warning, "Flare readiness probe for {ServerOrigin} timed out.")]
    public static partial void ProbeTimedOut(ILogger logger, string serverOrigin, Exception? exception = null);

    [LoggerMessage(1004, LogLevel.Warning, "TLS validation failed while probing {ServerOrigin}.")]
    public static partial void TlsFailed(ILogger logger, string serverOrigin, Exception exception);

    [LoggerMessage(1005, LogLevel.Warning, "Flare readiness probe for {ServerOrigin} received an invalid redirect response.")]
    public static partial void InvalidRedirect(ILogger logger, string serverOrigin, Exception exception);

    [LoggerMessage(1006, LogLevel.Warning, "Flare readiness probe for {ServerOrigin} failed.")]
    public static partial void ProbeFailed(ILogger logger, string serverOrigin, Exception exception);

    [LoggerMessage(1007, LogLevel.Error, "Unexpected failure while probing {ServerOrigin}.")]
    public static partial void UnexpectedProbeFailure(ILogger logger, string serverOrigin, Exception exception);

    [LoggerMessage(1008, LogLevel.Information, "Flare readiness probe for {ServerOrigin} returned HTTP {StatusCode}.")]
    public static partial void HealthResponse(ILogger logger, string serverOrigin, int statusCode);

    [LoggerMessage(1009, LogLevel.Information, "Stored normalized Flare server URL for {ServerOrigin}.")]
    public static partial void ServerUrlStored(ILogger logger, string serverOrigin);

    [LoggerMessage(1010, LogLevel.Error, "Flare connected to {ServerOrigin}, but its URL could not be stored.")]
    public static partial void ServerUrlStorageFailed(ILogger logger, string serverOrigin, Exception exception);

    [LoggerMessage(1011, LogLevel.Warning, "The health response from {ServerOrigin} could not be disposed cleanly.")]
    public static partial void ResponseDisposeFailed(ILogger logger, string serverOrigin, Exception exception);
}
