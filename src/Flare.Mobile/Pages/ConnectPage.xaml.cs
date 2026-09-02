using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class ConnectPage : BindablePage
{
    private readonly ConnectionService _connection;
    private readonly AppNavigator _navigator;
    private string _connectionState = string.Empty;
    public bool IsNotBusy => !IsBusy;
    public string ConnectionState { get => _connectionState; private set => Set(ref _connectionState, value); }

    public ConnectPage(ConnectionService connection, AppNavigator navigator)
    {
        InitializeComponent();
        _connection = connection;
        _navigator = navigator;
        BindingContext = this;
    }

    private async void ConnectClicked(object? sender, EventArgs eventArgs)
    {
        if (IsBusy) return;
        IsBusy = true;
        OnPropertyChanged(nameof(IsNotBusy));
        ErrorMessage = null;
        ConnectionState = "Connecting";
        var result = await _connection.ConnectAsync(ServerEntry.Text ?? string.Empty, CancellationToken.None);
        IsBusy = false;
        OnPropertyChanged(nameof(IsNotBusy));
        if (result.Success)
        {
            ConnectionState = "Connected";
            await Task.Delay(180);
            _navigator.ShowLogin();
        }
        else
        {
            ConnectionState = "Unreachable";
            ErrorMessage = result.Message;
        }
    }
}
