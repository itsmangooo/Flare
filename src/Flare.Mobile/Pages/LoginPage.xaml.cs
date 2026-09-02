using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class LoginPage : BindablePage
{
    private readonly AuthService _auth;
    private readonly AppNavigator _navigator;
    private readonly SessionStore _session;
    public bool IsNotBusy => !IsBusy;
    public string ServerUrl => _session.ServerUrl ?? string.Empty;

    public LoginPage(AuthService auth, AppNavigator navigator, SessionStore session)
    {
        InitializeComponent();
        _auth = auth;
        _navigator = navigator;
        _session = session;
        BindingContext = this;
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
            _navigator.ShowShell();
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; OnPropertyChanged(nameof(IsNotBusy)); }
    }

    private void ChangeServerClicked(object? sender, EventArgs eventArgs) => _navigator.ChangeServer();
}
