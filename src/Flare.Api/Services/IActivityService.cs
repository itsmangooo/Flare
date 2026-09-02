using Flare.Contracts;

namespace Flare.Api.Services;

public interface IActivityService
{
    Task<PagedResponse<ActivityEventResponse>> GetAsync(int page, int pageSize, CancellationToken cancellationToken);
}
