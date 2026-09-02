using Flare.Mobile.Pages;
using Flare.Mobile.Services;

namespace Flare.Mobile;

public partial class App : Application
{
    private readonly StartupPage _startup;
    private readonly AppNavigator _navigator;
    private readonly LiveTelemetryService _telemetry;

    public App(StartupPage startup, AppNavigator navigator, LiveTelemetryService telemetry)
    {
        InitializeComponent();
        _startup = startup;
        _navigator = navigator;
        _telemetry = telemetry;
        UserAppTheme = Preferences.Default.Get("flare.theme", "Dark") == "System"
            ? AppTheme.Unspecified
            : AppTheme.Dark;
    }

    protected override Window CreateWindow(IActivationState? activationState)
    {
        var window = new Window(_startup);
        _navigator.Attach(window);
        window.Resumed += (_, _) => _ = _telemetry.StartAsync();
        window.Stopped += (_, _) => _ = _telemetry.StopAsync();
        return window;
    }
}
