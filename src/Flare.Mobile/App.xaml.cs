using Flare.Mobile.Pages;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile;

public partial class App : Application
{
    private readonly IServiceProvider _services;
    private readonly AppNavigator _navigator;
    private readonly LiveTelemetryService _telemetry;
    private readonly ILogger<App> _logger;

    public App(IServiceProvider services, AppNavigator navigator, LiveTelemetryService telemetry, ILogger<App> logger)
    {
        InitializeComponent();
        _services = services;
        _navigator = navigator;
        _telemetry = telemetry;
        _logger = logger;
        UserAppTheme = Preferences.Default.Get("flare.theme", "Dark") == "System"
            ? AppTheme.Unspecified
            : AppTheme.Dark;
    }

    protected override Window CreateWindow(IActivationState? activationState)
    {
        // Resolve the first page only after InitializeComponent has loaded application resources.
        // Resolving it as an App constructor dependency makes StaticResource lookups fail at startup.
        var window = new Window(_services.GetRequiredService<StartupPage>());
        _navigator.Attach(window);
        window.Resumed += WindowResumed;
        window.Stopped += WindowStopped;
        return window;
    }

    private async void WindowResumed(object? sender, EventArgs eventArgs)
    {
        try
        {
            await _telemetry.StartAsync();
        }
        catch (Exception exception)
        {
            MobileLog.ResumeTelemetryFailed(_logger, exception);
        }
    }

    private async void WindowStopped(object? sender, EventArgs eventArgs)
    {
        try
        {
            await _telemetry.StopAsync();
        }
        catch (Exception exception)
        {
            MobileLog.StopTelemetryFailed(_logger, exception);
        }
    }
}
