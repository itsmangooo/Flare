using Flare.Contracts;

namespace Flare.Api.Services;

public interface ICoolifyService
{
    bool IsConfigured { get; }
    Task<IReadOnlyList<CoolifyServerResponse>> GetServersAsync(CancellationToken cancellationToken);
    Task<IReadOnlyList<CoolifyResourceResponse>> GetServerResourcesAsync(string serverUuid, CancellationToken cancellationToken);
    Task<IReadOnlyList<CoolifyApplicationResponse>> GetApplicationsAsync(CancellationToken cancellationToken);
    Task<IReadOnlyList<CoolifyServiceResponse>> GetServicesAsync(CancellationToken cancellationToken);
    Task<PagedResponse<DeploymentResponse>> GetDeploymentsAsync(int page, int pageSize, CancellationToken cancellationToken);
    Task<DeploymentResponse?> GetDeploymentAsync(string uuid, CancellationToken cancellationToken);
    Task<ActionResponse> StartApplicationAsync(string uuid, CancellationToken cancellationToken);
    Task<ActionResponse> StopApplicationAsync(string uuid, CancellationToken cancellationToken);
    Task<ActionResponse> RestartApplicationAsync(string uuid, CancellationToken cancellationToken);
    Task<ActionResponse> RestartServiceAsync(string uuid, CancellationToken cancellationToken);
    Task<ActionResponse> RedeployApplicationAsync(string uuid, CancellationToken cancellationToken);
}
