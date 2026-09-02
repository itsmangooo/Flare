using Flare.Mobile.Pages;
using Flare.Mobile.Services;

namespace Flare.Mobile;

public partial class App : Application
{
    private readonly IServiceProvider _services;
    private readonly AppNavigator _navigator;
    private readonly LiveTelemetryService _telemetry;

    public App(IServiceProvider services, AppNavigator navigator, LiveTelemetryService telemetry)
    {
        InitializeComponent();
        _services = services;
        _navigator = navigator;
        _telemetry = telemetry;
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
        window.Resumed += (_, _) => _ = _telemetry.StartAsync();
        window.Stopped += (_, _) => _ = _telemetry.StopAsync();
        return window;
    }
}
