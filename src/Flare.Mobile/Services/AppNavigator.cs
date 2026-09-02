using Flare.Mobile.Pages;

namespace Flare.Mobile.Services;

public sealed class AppNavigator(IServiceProvider provider, LiveTelemetryService telemetry, SessionStore sessionStore)
{
    private Window? _window;
    public void Attach(Window window) => _window = window;
    public void ShowConnect() => SetRoot(provider.GetRequiredService<ConnectPage>());
    public void ShowLogin()
    {
        _ = telemetry.ResetAsync();
        SetRoot(provider.GetRequiredService<LoginPage>());
    }
    public void ShowShell()
    {
        SetRoot(provider.GetRequiredService<AppShell>());
        _ = telemetry.StartAsync();
    }
    public async Task ShowSettingsAsync() => await Shell.Current.Navigation.PushAsync(provider.GetRequiredService<SettingsPage>());
    public void ChangeServer()
    {
        sessionStore.ClearServer();
        _ = telemetry.ResetAsync();
        ShowConnect();
    }

    private void SetRoot(Page page)
    {
        if (_window is null) return;
        _window.Page = page;
    }
}
