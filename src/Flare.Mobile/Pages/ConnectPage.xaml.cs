using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class ConnectPage : BindablePage
{
    private readonly ConnectionService _connection;
    private readonly AppNavigator _navigator;
    private readonly ILogger<ConnectPage> _logger;
    private CancellationTokenSource? _connectCancellation;
    private int _connectInProgress;
    private string _connectionState = string.Empty;
    public bool IsNotBusy => !IsBusy;
    public string ConnectionState { get => _connectionState; private set => Set(ref _connectionState, value); }

    public ConnectPage(ConnectionService connection, AppNavigator navigator, ILogger<ConnectPage> logger)
    {
        InitializeComponent();
        _connection = connection;
        _navigator = navigator;
        _logger = logger;
        BindingContext = this;
    }

    private async void ConnectClicked(object? sender, EventArgs eventArgs)
    {
        if (Interlocked.Exchange(ref _connectInProgress, 1) != 0)
        {
            MobileLog.RepeatedConnectIgnored(_logger);
            return;
        }

        var cancellation = new CancellationTokenSource();
        _connectCancellation = cancellation;
        IsBusy = true;
        OnPropertyChanged(nameof(IsNotBusy));
        ErrorMessage = null;
        ConnectionState = "Connecting";

        try
        {
            var result = await _connection.ConnectAsync(ServerEntry.Text ?? string.Empty, cancellation.Token);
            ConnectionState = result.StateLabel;
            if (!result.Success)
            {
                ErrorMessage = result.Message;
                return;
            }

            await Task.Delay(180, cancellation.Token);
            MobileLog.LoginNavigationStarted(_logger);
            await _navigator.ShowLoginAsync(cancellation.Token);
        }
        catch (OperationCanceledException) when (cancellation.IsCancellationRequested)
        {
            MobileLog.ConnectFlowCanceled(_logger);
        }
        catch (Exception exception)
        {
            MobileLog.ConnectFlowFailed(_logger, exception);
            ErrorMessage = ConnectionState == "Connected"
                ? "Connected, but the sign-in screen could not be opened. Try again."
                : "Flare could not complete the connection. Try again.";
        }
        finally
        {
            if (ReferenceEquals(_connectCancellation, cancellation))
            {
                _connectCancellation = null;
            }

            cancellation.Dispose();
            IsBusy = false;
            OnPropertyChanged(nameof(IsNotBusy));
            Interlocked.Exchange(ref _connectInProgress, 0);
        }
    }

    protected override void OnDisappearing()
    {
        base.OnDisappearing();
        _connectCancellation?.Cancel();
    }
}
