using Flare.Contracts;
using Microsoft.AspNetCore.SignalR.Client;

namespace Flare.Mobile.Services;

public sealed class LiveTelemetryService(ApiClient api, SessionStore sessionStore) : IDisposable, IAsyncDisposable
{
    private readonly SemaphoreSlim _gate = new(1, 1);
    private HubConnection? _connection;
    private CancellationTokenSource? _freshnessCancellation;
    private DateTimeOffset? _lastSnapshotAt;
    private DataFreshness _freshness = DataFreshness.Offline;
    public event EventHandler<OverviewResponse>? SnapshotReceived;
    public event EventHandler<DataFreshness>? FreshnessChanged;

    public async Task StartAsync(CancellationToken cancellationToken = default)
    {
        await _gate.WaitAsync(cancellationToken);
        try
        {
            if (_connection?.State is HubConnectionState.Connected or HubConnectionState.Connecting) return;
            if (sessionStore.ServerUrl is null || !await sessionStore.HasSessionAsync()) return;
            _connection ??= BuildConnection();
            await _connection.StartAsync(cancellationToken);
            SetFreshness(DataFreshness.Live);
            var fresh = await api.GetAsync<OverviewResponse>("api/v1/overview", cancellationToken);
            PublishSnapshot(fresh);
            StartFreshnessMonitor();
        }
        catch (Exception) when (!cancellationToken.IsCancellationRequested)
        {
            SetFreshness(DataFreshness.Offline);
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task StopAsync()
    {
        await _gate.WaitAsync();
        try
        {
            StopFreshnessMonitor();
            if (_connection is not null) await _connection.StopAsync();
            SetFreshness(DataFreshness.Stale);
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task ResetAsync()
    {
        await StopAsync();
        await _gate.WaitAsync();
        try
        {
            if (_connection is not null) await _connection.DisposeAsync();
            _connection = null;
            _lastSnapshotAt = null;
        }
        finally { _gate.Release(); }
    }

    private HubConnection BuildConnection()
    {
        var connection = new HubConnectionBuilder()
            .WithUrl($"{sessionStore.ServerUrl}/hubs/telemetry", options =>
                options.AccessTokenProvider = () => api.GetValidAccessTokenAsync(CancellationToken.None))
            .WithAutomaticReconnect([TimeSpan.Zero, TimeSpan.FromSeconds(2), TimeSpan.FromSeconds(5),
                TimeSpan.FromSeconds(10), TimeSpan.FromSeconds(30)])
            .Build();
        connection.On<OverviewResponse>("SnapshotUpdated", PublishSnapshot);
        connection.Reconnecting += _ =>
        {
            SetFreshness(DataFreshness.Reconnecting);
            return Task.CompletedTask;
        };
        connection.Reconnected += async _ =>
        {
            SetFreshness(DataFreshness.Live);
            try { PublishSnapshot(await api.GetAsync<OverviewResponse>("api/v1/overview", CancellationToken.None)); }
            catch (FlareApiException) { SetFreshness(DataFreshness.Stale); }
        };
        connection.Closed += _ =>
        {
            StopFreshnessMonitor();
            SetFreshness(DataFreshness.Offline);
            return Task.CompletedTask;
        };
        return connection;
    }

    private void PublishSnapshot(OverviewResponse snapshot)
    {
        _lastSnapshotAt = DateTimeOffset.UtcNow;
        SetFreshness(snapshot.Freshness);
        SnapshotReceived?.Invoke(this, snapshot);
    }

    private void StartFreshnessMonitor()
    {
        StopFreshnessMonitor();
        _freshnessCancellation = new CancellationTokenSource();
        _ = MonitorFreshnessAsync(_freshnessCancellation.Token);
    }

    private void StopFreshnessMonitor()
    {
        _freshnessCancellation?.Cancel();
        _freshnessCancellation?.Dispose();
        _freshnessCancellation = null;
    }

    private async Task MonitorFreshnessAsync(CancellationToken cancellationToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(5));
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                if (_connection?.State == HubConnectionState.Connected
                    && (_lastSnapshotAt is null || DateTimeOffset.UtcNow - _lastSnapshotAt > TimeSpan.FromSeconds(10)))
                {
                    SetFreshness(DataFreshness.Stale);
                }
            }
        }
        catch (OperationCanceledException) { }
    }

    private void SetFreshness(DataFreshness freshness)
    {
        if (_freshness == freshness) return;
        _freshness = freshness;
        FreshnessChanged?.Invoke(this, freshness);
    }

    public void Dispose()
    {
        StopFreshnessMonitor();
        _gate.Dispose();
        GC.SuppressFinalize(this);
    }

    public async ValueTask DisposeAsync()
    {
        StopFreshnessMonitor();
        if (_connection is not null) await _connection.DisposeAsync();
        _gate.Dispose();
        GC.SuppressFinalize(this);
    }
}
