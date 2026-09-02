using Flare.Api.Data;
using Flare.Api.Infrastructure;
using Flare.Contracts;
using Microsoft.EntityFrameworkCore;

namespace Flare.Api.Services;

public sealed class OverviewService(
    IHostMetricsService hostMetrics,
    IDockerService docker,
    IActivityService activity,
    FlareDbContext database,
    TimeProvider timeProvider) : IOverviewService
{
    public async Task<OverviewResponse> GetAsync(CancellationToken cancellationToken)
    {
        var host = await hostMetrics.GetAsync(cancellationToken);
        ContainerTotalsResponse totals;
        var dockerAvailable = true;
        try
        {
            var containers = await docker.GetContainersAsync(cancellationToken);
            totals = new ContainerTotalsResponse(
                containers.Count(item => item.State == ContainerState.Running),
                containers.Count(item => item.State != ContainerState.Running),
                containers.Count(item => item.Health == HealthState.Unhealthy));
        }
        catch (InfrastructureUnavailableException)
        {
            dockerAvailable = false;
            totals = new ContainerTotalsResponse(null, null, null);
        }

        var cutoff = timeProvider.GetUtcNow().AddHours(-1);
        var history = await database.MetricSamples.AsNoTracking()
            .Where(sample => sample.Timestamp >= cutoff)
            .OrderBy(sample => sample.Timestamp)
            .Select(sample => new MetricPoint(sample.Timestamp, sample.CpuPercent, sample.MemoryPercent))
            .ToArrayAsync(cancellationToken);
        var recent = (await activity.GetAsync(1, 5, cancellationToken)).Items;
        var hostAvailable = host.CpuPercent is not null || host.MemoryTotalBytes is not null || host.Uptime is not null;
        var freshness = hostAvailable || dockerAvailable ? DataFreshness.Live : DataFreshness.Offline;
        return new OverviewResponse(timeProvider.GetUtcNow(), freshness, host, totals, history, recent);
    }
}
