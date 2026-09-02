using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Services;

internal static partial class MobileLog
{
    [LoggerMessage(1100, LogLevel.Debug, "Resolving {Destination}.")]
    public static partial void ResolvingPage(ILogger logger, string destination);

    [LoggerMessage(1101, LogLevel.Information, "Navigation to {Destination} completed.")]
    public static partial void NavigationCompleted(ILogger logger, string destination);

    [LoggerMessage(1102, LogLevel.Warning, "Telemetry reset failed before navigating to {Destination}.")]
    public static partial void TelemetryResetFailed(ILogger logger, string destination, Exception exception);

    [LoggerMessage(1103, LogLevel.Warning, "Telemetry startup failed after navigating to {Destination}.")]
    public static partial void TelemetryStartFailed(ILogger logger, string destination, Exception exception);

    [LoggerMessage(1104, LogLevel.Debug, "Navigation to {Destination} was canceled.")]
    public static partial void NavigationCanceled(ILogger logger, string destination);

    [LoggerMessage(1105, LogLevel.Error, "Navigation to {Destination} failed.")]
    public static partial void NavigationFailed(ILogger logger, string destination, Exception exception);

    [LoggerMessage(1200, LogLevel.Debug, "Ignored a repeated Connect action while a connection attempt was active.")]
    public static partial void RepeatedConnectIgnored(ILogger logger);

    [LoggerMessage(1201, LogLevel.Information, "Connection succeeded; starting LoginPage navigation.")]
    public static partial void LoginNavigationStarted(ILogger logger);

    [LoggerMessage(1202, LogLevel.Debug, "Connect flow was canceled because ConnectPage disappeared.")]
    public static partial void ConnectFlowCanceled(ILogger logger);

    [LoggerMessage(1203, LogLevel.Error, "Connect flow failed before LoginPage could be shown.")]
    public static partial void ConnectFlowFailed(ILogger logger, Exception exception);

    [LoggerMessage(1300, LogLevel.Information, "LoginPage initialized for configured Flare server {ServerOrigin}.")]
    public static partial void LoginPageInitialized(ILogger logger, string serverOrigin);

    [LoggerMessage(1301, LogLevel.Error, "Login flow failed unexpectedly.")]
    public static partial void LoginFlowFailed(ILogger logger, Exception exception);

    [LoggerMessage(1302, LogLevel.Error, "Changing the configured Flare server failed.")]
    public static partial void ChangeServerFailed(ILogger logger, Exception exception);

    [LoggerMessage(1303, LogLevel.Error, "Navigation to LoginPage after logout failed.")]
    public static partial void LogoutNavigationFailed(ILogger logger, Exception exception);

    [LoggerMessage(1400, LogLevel.Critical, "Initial application navigation failed.")]
    public static partial void InitialNavigationFailed(ILogger logger, Exception exception);

    [LoggerMessage(1500, LogLevel.Warning, "Telemetry failed to start when the Android window resumed.")]
    public static partial void ResumeTelemetryFailed(ILogger logger, Exception exception);

    [LoggerMessage(1501, LogLevel.Warning, "Telemetry failed to stop when the Android window stopped.")]
    public static partial void StopTelemetryFailed(ILogger logger, Exception exception);
}
