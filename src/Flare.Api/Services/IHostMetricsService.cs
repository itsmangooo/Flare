using Flare.Contracts;

namespace Flare.Api.Services;

public interface IHostMetricsService
{
    Task<HostMetricsResponse> GetAsync(CancellationToken cancellationToken);
}
