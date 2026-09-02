using Flare.Contracts;

namespace Flare.Api.Services;

public interface IOverviewService
{
    Task<OverviewResponse> GetAsync(CancellationToken cancellationToken);
}
