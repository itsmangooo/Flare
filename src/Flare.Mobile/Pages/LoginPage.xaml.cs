using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class LoginPage : BindablePage
{
    private readonly AuthService _auth;
    private readonly AppNavigator _navigator;
    private readonly SessionStore _session;
    private readonly ILogger<LoginPage> _logger;
    public bool IsNotBusy => !IsBusy;
    public string ServerUrl => _session.ServerUrl ?? string.Empty;

    public LoginPage(AuthService auth, AppNavigator navigator, SessionStore session, ILogger<LoginPage> logger)
    {
        InitializeComponent();
        _auth = auth;
        _navigator = navigator;
        _session = session;
        _logger = logger;
        BindingContext = this;
        MobileLog.LoginPageInitialized(_logger, ServerUrl);
    }

    private async void LoginClicked(object? sender, EventArgs eventArgs)
    {
        if (IsBusy) return;
        IsBusy = true;
        OnPropertyChanged(nameof(IsNotBusy));
        ErrorMessage = null;
        try
        {
            await _auth.LoginAsync(EmailEntry.Text?.Trim() ?? string.Empty, PasswordEntry.Text ?? string.Empty,
                CancellationToken.None);
            PasswordEntry.Text = string.Empty;
            await _navigator.ShowShellAsync();
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        catch (Exception exception)
        {
            MobileLog.LoginFlowFailed(_logger, exception);
            ErrorMessage = "Flare could not complete sign in. Try again.";
        }
        finally { IsBusy = false; OnPropertyChanged(nameof(IsNotBusy)); }
    }

    private async void ChangeServerClicked(object? sender, EventArgs eventArgs)
    {
        try
        {
            await _navigator.ChangeServerAsync();
        }
        catch (Exception exception)
        {
            MobileLog.ChangeServerFailed(_logger, exception);
            ErrorMessage = "The server could not be changed. Try again.";
        }
    }
}
