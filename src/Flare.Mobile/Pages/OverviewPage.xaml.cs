using System.Collections.ObjectModel;
using System.Globalization;
using System.Windows.Input;
using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class OverviewPage : BindablePage
{
    private readonly ApiClient _api;
    private readonly LiveTelemetryService _telemetry;
    private readonly AppNavigator _navigator;
    private bool _subscribed;
    private string _hostName = "Homelab";
    private string _uptime = "—";
    private DataFreshness _freshness = DataFreshness.Offline;
    private double? _cpuPercent;
    private double? _loadAverage;
    private double? _memoryPercent;
    private double? _diskPercent;
    private string _memoryLine = "—";
    private string _diskLine = "—";
    private double? _networkReceive;
    private double? _networkTransmit;
    private string _running = "—";
    private string _stopped = "—";
    private string _unhealthy = "—";
    private IReadOnlyList<MetricPoint> _history = [];

    public string HostName { get => _hostName; private set => Set(ref _hostName, value); }
    public string Uptime { get => _uptime; private set => Set(ref _uptime, value); }
    public DataFreshness Freshness { get => _freshness; private set => Set(ref _freshness, value); }
    public double? CpuPercent { get => _cpuPercent; private set => Set(ref _cpuPercent, value); }
    public double? LoadAverage { get => _loadAverage; private set => Set(ref _loadAverage, value); }
    public double? MemoryPercent { get => _memoryPercent; private set => Set(ref _memoryPercent, value); }
    public double? DiskPercent { get => _diskPercent; private set => Set(ref _diskPercent, value); }
    public string MemoryLine { get => _memoryLine; private set => Set(ref _memoryLine, value); }
    public string DiskLine { get => _diskLine; private set => Set(ref _diskLine, value); }
    public double? NetworkReceive { get => _networkReceive; private set => Set(ref _networkReceive, value); }
    public double? NetworkTransmit { get => _networkTransmit; private set => Set(ref _networkTransmit, value); }
    public string Running { get => _running; private set => Set(ref _running, value); }
    public string Stopped { get => _stopped; private set => Set(ref _stopped, value); }
    public string Unhealthy { get => _unhealthy; private set => Set(ref _unhealthy, value); }
    public IReadOnlyList<MetricPoint> History { get => _history; private set => Set(ref _history, value); }
    public ObservableCollection<ActivityEventResponse> RecentActivity { get; } = [];
    public ICommand RefreshCommand { get; }

    public OverviewPage() : this(AppServices.Get<ApiClient>(), AppServices.Get<LiveTelemetryService>(), AppServices.Get<AppNavigator>()) { }

    public OverviewPage(ApiClient api, LiveTelemetryService telemetry, AppNavigator navigator)
    {
        InitializeComponent();
        _api = api;
        _telemetry = telemetry;
        _navigator = navigator;
        RefreshCommand = new Command(async () => await LoadAsync());
        BindingContext = this;
    }

    protected override async void OnAppearing()
    {
        base.OnAppearing();
        if (!_subscribed)
        {
            _telemetry.SnapshotReceived += TelemetrySnapshotReceived;
            _telemetry.FreshnessChanged += TelemetryFreshnessChanged;
            _subscribed = true;
        }
        await LoadAsync();
    }

    protected override void OnDisappearing()
    {
        if (_subscribed)
        {
            _telemetry.SnapshotReceived -= TelemetrySnapshotReceived;
            _telemetry.FreshnessChanged -= TelemetryFreshnessChanged;
            _subscribed = false;
        }
        base.OnDisappearing();
    }

    private async Task LoadAsync()
    {
        if (IsBusy) return;
        IsBusy = true;
        ErrorMessage = null;
        try { Apply(await _api.GetAsync<OverviewResponse>("api/v1/overview", CancellationToken.None)); }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; Freshness = DataFreshness.Offline; }
        finally { IsBusy = false; }
    }

    private void TelemetrySnapshotReceived(object? sender, OverviewResponse snapshot) =>
        MainThread.BeginInvokeOnMainThread(() => Apply(snapshot));

    private void TelemetryFreshnessChanged(object? sender, DataFreshness freshness) =>
        MainThread.BeginInvokeOnMainThread(() => Freshness = freshness);

    private void Apply(OverviewResponse snapshot)
    {
        HostName = snapshot.Host.HostName;
        Uptime = FormatDuration(snapshot.Host.Uptime);
        Freshness = snapshot.Freshness;
        CpuPercent = snapshot.Host.CpuPercent;
        LoadAverage = snapshot.Host.LoadAverage;
        MemoryPercent = Percentage(snapshot.Host.MemoryUsedBytes, snapshot.Host.MemoryTotalBytes);
        DiskPercent = Percentage(snapshot.Host.DiskUsedBytes, snapshot.Host.DiskTotalBytes);
        MemoryLine = Pair(snapshot.Host.MemoryUsedBytes, snapshot.Host.MemoryTotalBytes);
        DiskLine = Pair(snapshot.Host.DiskUsedBytes, snapshot.Host.DiskTotalBytes);
        NetworkReceive = snapshot.Host.NetworkReceiveBytesPerSecond;
        NetworkTransmit = snapshot.Host.NetworkTransmitBytesPerSecond;
        Running = snapshot.Containers.Running?.ToString(CultureInfo.CurrentCulture) ?? "—";
        Stopped = snapshot.Containers.Stopped?.ToString(CultureInfo.CurrentCulture) ?? "—";
        Unhealthy = snapshot.Containers.Unhealthy?.ToString(CultureInfo.CurrentCulture) ?? "—";
        History = snapshot.History;
        RecentActivity.Clear();
        foreach (var item in snapshot.RecentActivity) RecentActivity.Add(item);
    }

    private async void SettingsClicked(object? sender, EventArgs eventArgs) => await _navigator.ShowSettingsAsync();
    private static double? Percentage(long? used, long? total) => used is { } u && total is > 0 ? u * 100d / total : null;
    private static string Pair(long? used, long? total) => used is { } u && total is { } t ? $"{FormatBytes(u)} / {FormatBytes(t)}" : "Unavailable";
    private static string FormatBytes(long bytes) => bytes >= 1L << 30 ? $"{bytes / (double)(1L << 30):0.#} GB" : $"{bytes / (double)(1L << 20):0.#} MB";
    private static string FormatDuration(TimeSpan? duration) => duration is { } value ? $"{(int)value.TotalDays}d {value.Hours:00}h" : "Unavailable";
}
