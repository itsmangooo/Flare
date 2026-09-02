using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class StartupPage : ContentPage
{
    private readonly SessionStore _session;
    private readonly AppNavigator _navigator;
    private bool _started;

    public StartupPage(SessionStore session, AppNavigator navigator)
    {
        InitializeComponent();
        _session = session;
        _navigator = navigator;
    }

    protected override async void OnAppearing()
    {
        base.OnAppearing();
        if (_started) return;
        _started = true;
        await Task.Delay(180);
        if (_session.ServerUrl is null) _navigator.ShowConnect();
        else if (await _session.HasSessionAsync()) _navigator.ShowShell();
        else _navigator.ShowLogin();
    }
}
