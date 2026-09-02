using Flare.Api.Data;
using Flare.Api.Hubs;
using Flare.Contracts;
using Microsoft.AspNetCore.SignalR;
using Microsoft.EntityFrameworkCore;

namespace Flare.Api.Services;

public sealed partial class TelemetryPublisherService(
    IServiceScopeFactory scopeFactory,
    IHubContext<TelemetryHub> hub,
    TimeProvider timeProvider,
    ILogger<TelemetryPublisherService> logger) : BackgroundService
{
    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(3), timeProvider);
        var lastPersisted = DateTimeOffset.MinValue;
        var lastPruned = DateTimeOffset.MinValue;
        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                using var scope = scopeFactory.CreateScope();
                var overview = scope.ServiceProvider.GetRequiredService<IOverviewService>();
                var snapshot = await overview.GetAsync(stoppingToken);
                await hub.Clients.All.SendAsync("SnapshotUpdated", snapshot, stoppingToken);

                var now = timeProvider.GetUtcNow();
                var database = scope.ServiceProvider.GetRequiredService<FlareDbContext>();
                if (now - lastPersisted >= TimeSpan.FromSeconds(15))
                {
                    database.MetricSamples.Add(new MetricSample
                    {
                        Timestamp = snapshot.GeneratedAt,
                        CpuPercent = snapshot.Host.CpuPercent,
                        MemoryPercent = Percentage(snapshot.Host.MemoryUsedBytes, snapshot.Host.MemoryTotalBytes)
                    });
                    await database.SaveChangesAsync(stoppingToken);
                    lastPersisted = now;
                }
                if (now - lastPruned >= TimeSpan.FromHours(1))
                {
                    var cutoff = now.AddDays(-1);
                    await database.MetricSamples.Where(sample => sample.Timestamp < cutoff)
                        .ExecuteDeleteAsync(stoppingToken);
                    lastPruned = now;
                }
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                break;
            }
            catch (Exception exception)
            {
                LogPublicationFailure(logger, exception);
            }

            if (!await timer.WaitForNextTickAsync(stoppingToken)) break;
        }
    }

    private static double? Percentage(long? used, long? total) =>
        used is { } usedValue && total is > 0 ? usedValue * 100d / total : null;

    [LoggerMessage(LogLevel.Warning, "Telemetry publication failed; the next scheduled sample will retry.")]
    private static partial void LogPublicationFailure(ILogger logger, Exception exception);
}
