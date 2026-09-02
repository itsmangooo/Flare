using Flare.Contracts;

namespace Flare.Api.Services;

public interface IDockerService
{
    Task<IReadOnlyList<ContainerSummaryResponse>> GetContainersAsync(CancellationToken cancellationToken);
    Task<ContainerDetailResponse?> GetContainerAsync(string id, CancellationToken cancellationToken);
    Task<LogPageResponse> GetLogsAsync(
        string id, int tail, DateTimeOffset? before, CancellationToken cancellationToken);
    Task StartAsync(string id, CancellationToken cancellationToken);
    Task StopAsync(string id, CancellationToken cancellationToken);
    Task RestartAsync(string id, CancellationToken cancellationToken);
}
