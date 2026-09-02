using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class ContainerDetailPage : BindablePage, IDisposable
{
    private readonly ApiClient _api;
    private readonly string _id;
    private ContainerDetailResponse? _detail;
    private string _logs = "Loading logs…";
    private string _memoryLine = "—";
    private string _logPosition = "LATEST";
    private IReadOnlyList<string> _portLines = [];
    private IReadOnlyList<string> _labelLines = [];
    private DateTimeOffset? _oldestLogTimestamp;
    private bool _canLoadOlder;
    private bool _loadingLogs;
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
    {
        InitializeComponent();
        _api = api;
        _id = id;
        BindingContext = this;
    }

    protected override async void OnAppearing() { base.OnAppearing(); await LoadAsync(); }
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
        var selected = await DisplayActionSheetAsync(
            $"{label} {Detail?.Name ?? "the container"}?", "Cancel", null, label);
        if (selected != label) return;
        HapticFeedback.Default.Perform(HapticFeedbackType.LongPress);
        IsBusy = true; ErrorMessage = null;
        try
        {
            _ = await _api.PostAsync<ActionResponse>($"api/v1/containers/{_id}/{action}", null, true, CancellationToken.None);
            await Task.Delay(250);
            ApplyDetail(await _api.GetAsync<ContainerDetailResponse>($"api/v1/containers/{_id}", CancellationToken.None));
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }

    private async void StartClicked(object? sender, EventArgs eventArgs) => await RunActionAsync("start", "Start");
    private async void StopClicked(object? sender, EventArgs eventArgs) => await RunActionAsync("stop", "Stop");
    private async void RestartClicked(object? sender, EventArgs eventArgs) => await RunActionAsync("restart", "Restart");
    private async void TailChanged(object? sender, EventArgs eventArgs)
    {
        if (Detail is not null) await ReloadLogsAsync(null);
    }
    private async void LatestLogsClicked(object? sender, EventArgs eventArgs)
    {
        await ReloadLogsAsync(null);
    }
    private async void OlderLogsClicked(object? sender, EventArgs eventArgs)
    {
        if (_oldestLogTimestamp is not { } oldest) return;
        LiveSwitch.IsToggled = false;
        await ReloadLogsAsync(oldest.AddTicks(-1));
    }

    private async Task ReloadLogsAsync(DateTimeOffset? before)
    {
        ErrorMessage = null;
        try { await LoadLogsAsync(before, CancellationToken.None); }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
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
