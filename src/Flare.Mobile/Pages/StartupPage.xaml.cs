using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class StartupPage : ContentPage
{
    private readonly SessionStore _session;
    private readonly AppNavigator _navigator;
    private readonly ILogger<StartupPage> _logger;
    private bool _started;

    public StartupPage(SessionStore session, AppNavigator navigator, ILogger<StartupPage> logger)
    {
        InitializeComponent();
        _session = session;
        _navigator = navigator;
        _logger = logger;
    }

    protected override async void OnAppearing()
    {
        base.OnAppearing();
        if (_started) return;
        _started = true;
        try
        {
            await Task.Delay(180);
            if (_session.ServerUrl is null) await _navigator.ShowConnectAsync();
            else if (await _session.HasSessionAsync()) await _navigator.ShowShellAsync();
            else await _navigator.ShowLoginAsync();
        }
        catch (Exception exception)
        {
            MobileLog.InitialNavigationFailed(_logger, exception);
        }
    }
}
