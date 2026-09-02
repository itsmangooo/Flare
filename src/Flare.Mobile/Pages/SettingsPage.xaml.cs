using Flare.Contracts;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class SettingsPage : BindablePage
{
    private readonly ApiClient _api;
    private readonly AuthService _auth;
    private readonly SessionStore _session;
    private readonly AppNavigator _navigator;
    private readonly ILogger<SettingsPage> _logger;
    private string _connectivity = "Checking…";
    private string _apiVersion = "—";
    private string _serverVersion = "—";
    public string Email => _session.UserEmail ?? "Unknown account";
    public string ServerUrl => _session.ServerUrl ?? "Not configured";
    public string Connectivity { get => _connectivity; private set => Set(ref _connectivity, value); }
    public string ApiVersion { get => _apiVersion; private set => Set(ref _apiVersion, value); }
    public string ServerVersion { get => _serverVersion; private set => Set(ref _serverVersion, value); }
    public string MobileVersion { get; }

    public SettingsPage(ApiClient api, AuthService auth, SessionStore session, AppNavigator navigator,
        ILogger<SettingsPage> logger)
    {
        InitializeComponent(); _api = api; _auth = auth; _session = session; _navigator = navigator;
        _logger = logger;
        MobileVersion = AppInfo.Current.VersionString;
        ThemePicker.SelectedIndex = Preferences.Default.Get("flare.theme", "Dark") == "System" ? 1 : 0;
        BindingContext = this;
    }
    protected override async void OnAppearing() { base.OnAppearing(); await ReconnectAsync(); }

    private async Task ReconnectAsync()
    {
        if (IsBusy) return; IsBusy = true; ErrorMessage = null; Connectivity = "Connecting…";
        try
        {
            var info = await _api.GetAsync<ServerInfoResponse>("api/v1/system/info", CancellationToken.None);
            ApiVersion = info.ApiVersion; ServerVersion = info.ServerVersion; Connectivity = "Connected";
        }
        catch (FlareApiException exception) { Connectivity = "Unreachable"; ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }
    private async void ReconnectClicked(object? sender, EventArgs eventArgs) => await ReconnectAsync();
    private async void LogoutClicked(object? sender, EventArgs eventArgs)
    {
        if (!await DisplayAlertAsync("Log out?", "The secure session tokens will be removed from this device.", "Logout", "Cancel")) return;
        try { await _auth.LogoutAsync(CancellationToken.None); } catch (FlareApiException) { _session.ClearAuthentication(); }
        try { await _navigator.ShowLoginAsync(); }
        catch (Exception exception)
        {
            MobileLog.LogoutNavigationFailed(_logger, exception);
            ErrorMessage = "Signed out, but the sign-in screen could not be opened.";
        }
    }
    private async void ChangeServerClicked(object? sender, EventArgs eventArgs)
    {
        try { await _navigator.ChangeServerAsync(); }
        catch (Exception exception)
        {
            MobileLog.ChangeServerFailed(_logger, exception);
            ErrorMessage = "The server could not be changed. Try again.";
        }
    }
    private void ThemeChanged(object? sender, EventArgs eventArgs)
    {
        if (ThemePicker.SelectedIndex < 0) return;
        var theme = ThemePicker.SelectedIndex == 1 ? "System" : "Dark";
        Preferences.Default.Set("flare.theme", theme);
        if (Application.Current is not null) Application.Current.UserAppTheme = theme == "System" ? AppTheme.Unspecified : AppTheme.Dark;
    }
}
