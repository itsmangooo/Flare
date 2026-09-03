using Flare.Contracts;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class ContainerDetailPage : BindablePage, IDisposable
{
    private readonly ApiClient _api;
    private readonly string _id;
    private readonly IActionFeedback _actionFeedback;
    private readonly ILogger<ContainerDetailPage> _logger;
    private ContainerDetailResponse? _detail;
    private string _logs = "Loading logs…";
    private string _memoryLine = "—";
    private string _logPosition = "LATEST";
    private IReadOnlyList<string> _portLines = [];
    private IReadOnlyList<string> _labelLines = [];
    private DateTimeOffset? _oldestLogTimestamp;
    private bool _canLoadOlder;
    private bool _loadingLogs;
    private int _actionInProgress;
    private CancellationTokenSource? _liveCancellation;
    public ContainerDetailResponse? Detail { get => _detail; private set => Set(ref _detail, value); }
    public string Logs { get => _logs; private set => Set(ref _logs, value); }
    public string MemoryLine { get => _memoryLine; private set => Set(ref _memoryLine, value); }
    public string LogPosition { get => _logPosition; private set => Set(ref _logPosition, value); }
    public IReadOnlyList<string> PortLines { get => _portLines; private set { if (Set(ref _portLines, value)) OnPropertyChanged(nameof(HasPorts)); } }
    public IReadOnlyList<string> LabelLines { get => _labelLines; private set { if (Set(ref _labelLines, value)) OnPropertyChanged(nameof(HasLabels)); } }
    public bool HasPorts => PortLines.Count > 0;
    public bool HasLabels => LabelLines.Count > 0;
    public bool CanLoadOlder { get => _canLoadOlder; private set => Set(ref _canLoadOlder, value); }

    public ContainerDetailPage(ApiClient api, string id)
        : this(api, id, AppServices.Get<IActionFeedback>(), AppServices.Get<ILogger<ContainerDetailPage>>())
    {
    }

    internal ContainerDetailPage(
        ApiClient api,
        string id,
        IActionFeedback actionFeedback,
        ILogger<ContainerDetailPage> logger)
    {
        InitializeComponent();
        _api = api;
        _id = id;
        _actionFeedback = actionFeedback;
        _logger = logger;
        BindingContext = this;
    }

    protected override void OnAppearing()
    {
        base.OnAppearing();
        _ = LoadAsync();
    }
    protected override void OnDisappearing() { _liveCancellation?.Cancel(); base.OnDisappearing(); }

    private async Task LoadAsync()
    {
        if (IsBusy) return;
        IsBusy = true; ErrorMessage = null;
        try
        {
            ApplyDetail(await _api.GetAsync<ContainerDetailResponse>($"api/v1/containers/{_id}", CancellationToken.None));
            await LoadLogsAsync(null, CancellationToken.None);
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        catch (Exception exception)
        {
            MobileLog.UiActionFailed(_logger, "container.details.load", exception);
            ErrorMessage = "Flare could not load this container. Try again.";
        }
        finally { IsBusy = false; }
    }

    private async Task LoadLogsAsync(DateTimeOffset? before, CancellationToken cancellationToken)
    {
        if (_loadingLogs) return;
        _loadingLogs = true;
        try
        {
            var tail = int.TryParse(TailPicker.SelectedItem?.ToString(), out var count) ? count : 300;
            var cursor = before is null
                ? string.Empty
                : $"&before={Uri.EscapeDataString(before.Value.ToUniversalTime().ToString("O"))}";
            var page = await _api.GetAsync<LogPageResponse>(
                $"api/v1/containers/{_id}/logs?tail={tail}{cursor}", cancellationToken);
            Logs = page.Lines.Count == 0 ? "No log output." : string.Join(Environment.NewLine, page.Lines);
            _oldestLogTimestamp = page.OldestTimestamp;
            CanLoadOlder = page.Truncated && page.OldestTimestamp is not null;
            LogPosition = before is null ? "LATEST" : "HISTORY";
        }
        finally { _loadingLogs = false; }
    }

    private void ApplyDetail(ContainerDetailResponse detail)
    {
        Detail = detail;
        MemoryLine = detail.MemoryBytes is { } used
            ? detail.MemoryLimitBytes is { } limit ? $"{FormatBytes(used)} / {FormatBytes(limit)}" : FormatBytes(used)
            : "Unavailable";
        PortLines = detail.Ports.OrderBy(port => port.PrivatePort).Select(port =>
            port.PublicPort is { } publicPort
                ? $"{port.PrivatePort}/{port.Protocol}  →  {port.HostIp ?? "*"}:{publicPort}"
                : $"{port.PrivatePort}/{port.Protocol}").ToArray();
        LabelLines = detail.Labels.OrderBy(label => label.Key, StringComparer.OrdinalIgnoreCase)
            .Select(label => $"{label.Key}={label.Value}").ToArray();
    }

    private async Task RunActionAsync(string action, string label)
    {
        var operation = $"container.{action}";
        if (IsBusy || Interlocked.CompareExchange(ref _actionInProgress, 1, 0) != 0)
        {
            MobileLog.RepeatedActionIgnored(_logger, operation);
            return;
        }

        try
        {
            await RunUiActionSafelyAsync(_logger, operation, async () =>
            {
                var selected = await DisplayActionSheetAsync(
                    $"{label} {Detail?.Name ?? "the container"}?", "Cancel", null, label);
                if (selected != label) return;

                IsBusy = true;
                ErrorMessage = null;
                try
                {
                    _actionFeedback.TryPerformLongPress(operation);
                    await _api.PostAsync<ActionResponse>(
                        $"api/v1/containers/{_id}/{action}", null, true, CancellationToken.None);
                    await Task.Delay(250);
                    ApplyDetail(await _api.GetAsync<ContainerDetailResponse>(
                        $"api/v1/containers/{_id}", CancellationToken.None));
                }
                finally
                {
                    IsBusy = false;
                }
            });
        }
        finally
        {
            Interlocked.Exchange(ref _actionInProgress, 0);
        }
    }

    private void StartClicked(object? sender, EventArgs eventArgs) => _ = RunActionAsync("start", "Start");
    private void StopClicked(object? sender, EventArgs eventArgs) => _ = RunActionAsync("stop", "Stop");
    private void RestartClicked(object? sender, EventArgs eventArgs) => _ = RunActionAsync("restart", "Restart");
    private void TailChanged(object? sender, EventArgs eventArgs)
    {
        if (Detail is not null) _ = ReloadLogsAsync(null);
    }
    private void LatestLogsClicked(object? sender, EventArgs eventArgs)
    {
        _ = ReloadLogsAsync(null);
    }
    private void OlderLogsClicked(object? sender, EventArgs eventArgs)
    {
        if (_oldestLogTimestamp is not { } oldest) return;
        LiveSwitch.IsToggled = false;
        _ = ReloadLogsAsync(oldest.AddTicks(-1));
    }

    private async Task ReloadLogsAsync(DateTimeOffset? before)
    {
        ErrorMessage = null;
        await RunUiActionSafelyAsync(
            _logger,
            "container.logs.load",
            () => LoadLogsAsync(before, CancellationToken.None));
    }
    private void LiveToggled(object? sender, ToggledEventArgs eventArgs)
    {
        _liveCancellation?.Cancel();
        if (!eventArgs.Value) return;
        _liveCancellation = new CancellationTokenSource();
        _ = FollowLogsAsync(_liveCancellation.Token);
    }

    private async Task FollowLogsAsync(CancellationToken cancellationToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(5));
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                try
                {
                    await LoadLogsAsync(null, cancellationToken);
                    ErrorMessage = null;
                }
                catch (FlareApiException exception) { ErrorMessage = exception.Message; }
                catch (Exception exception)
                {
                    MobileLog.UiActionFailed(_logger, "container.logs.follow", exception);
                    ErrorMessage = "Flare could not refresh the container logs. Try again.";
                }
            }
        }
        catch (OperationCanceledException) { }
    }

    public void Dispose()
    {
        _liveCancellation?.Cancel();
        _liveCancellation?.Dispose();
        _liveCancellation = null;
        GC.SuppressFinalize(this);
    }

    private static string FormatBytes(long bytes) => bytes >= 1L << 30
        ? $"{bytes / (double)(1L << 30):0.#} GB"
        : $"{bytes / (double)(1L << 20):0.#} MB";
}
