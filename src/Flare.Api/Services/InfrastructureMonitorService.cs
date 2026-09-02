using Flare.Api.Data;
using Flare.Api.Infrastructure;
using Flare.Contracts;

namespace Flare.Api.Services;

public sealed partial class InfrastructureMonitorService(
    IServiceScopeFactory scopeFactory,
    TimeProvider timeProvider,
    ILogger<InfrastructureMonitorService> logger) : BackgroundService
{
    private Dictionary<string, ContainerState>? _containerStates;
    private Dictionary<string, string?>? _deploymentStates;
    private bool? _dockerWasAvailable;

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(15), timeProvider);
        while (await timer.WaitForNextTickAsync(stoppingToken))
        {
            try
            {
                using var scope = scopeFactory.CreateScope();
                var database = scope.ServiceProvider.GetRequiredService<FlareDbContext>();
                await ObserveDockerAsync(scope.ServiceProvider.GetRequiredService<IDockerService>(), database, stoppingToken);
                var coolify = scope.ServiceProvider.GetRequiredService<ICoolifyService>();
                if (coolify.IsConfigured) await ObserveCoolifyAsync(coolify, database, stoppingToken);
                await database.SaveChangesAsync(stoppingToken);
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested) { break; }
            catch (Exception exception) { LogMonitoringFailure(logger, exception); }
        }
    }

    private async Task ObserveDockerAsync(IDockerService docker, FlareDbContext database, CancellationToken cancellationToken)
    {
        try
        {
            var containers = await docker.GetContainersAsync(cancellationToken);
            if (_dockerWasAvailable == false)
                Add(database, "server.reconnected", "Docker host", OperationResult.Succeeded, null);
            _dockerWasAvailable = true;
            var current = containers.ToDictionary(item => item.Id, item => item.State);
            if (_containerStates is not null)
            {
                foreach (var container in containers)
                {
                    if (_containerStates.TryGetValue(container.Id, out var previous) && previous != container.State)
                        Add(database, "container.state.changed", container.Name,
                            container.State is ContainerState.Dead or ContainerState.Stopped ? OperationResult.Failed : OperationResult.Succeeded,
                            $"{previous} -> {container.State}");
                }
            }
            _containerStates = current;
        }
        catch (InfrastructureUnavailableException)
        {
            if (_dockerWasAvailable == true) Add(database, "server.disconnected", "Docker host", OperationResult.Failed, null);
            _dockerWasAvailable = false;
        }
    }

    private async Task ObserveCoolifyAsync(ICoolifyService coolify, FlareDbContext database, CancellationToken cancellationToken)
    {
        var deployments = await coolify.GetDeploymentsAsync(1, 30, cancellationToken);
        var current = deployments.Items.ToDictionary(item => item.Uuid, item => item.Status);
        if (_deploymentStates is not null)
        {
            foreach (var deployment in deployments.Items)
            {
                if (_deploymentStates.TryGetValue(deployment.Uuid, out var previous)
                    && !string.Equals(previous, deployment.Status, StringComparison.OrdinalIgnoreCase))
                {
                    var failed = deployment.Status?.Contains("fail", StringComparison.OrdinalIgnoreCase) == true;
                    Add(database, failed ? "deployment.failed" : "deployment.status.changed", deployment.ResourceName,
                        failed ? OperationResult.Failed : OperationResult.Succeeded,
                        $"{previous ?? "unknown"} -> {deployment.Status ?? "unknown"}");
                }
            }
        }
        _deploymentStates = current;
    }

    private void Add(FlareDbContext database, string action, string target, OperationResult result, string? detail) =>
        database.InfrastructureEvents.Add(new InfrastructureEvent
        {
            Id = Guid.NewGuid(),
            Action = action,
            Target = target,
            Result = result,
            Detail = detail,
            Timestamp = timeProvider.GetUtcNow()
        });

    [LoggerMessage(LogLevel.Warning, "Infrastructure event monitoring failed and will retry.")]
    private static partial void LogMonitoringFailure(ILogger logger, Exception exception);
}
