using System.Security.Claims;
using Flare.Api.Services;
using Flare.Contracts;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.RateLimiting;

namespace Flare.Api.Controllers;

[ApiController]
[Authorize]
[Route("api/v1/coolify")]
public sealed class CoolifyController(ICoolifyService coolify, IAuditService audit) : ControllerBase
{
    [HttpGet("servers")]
    public Task<IReadOnlyList<CoolifyServerResponse>> Servers(CancellationToken cancellationToken) =>
        coolify.GetServersAsync(cancellationToken);

    [HttpGet("servers/{uuid}/resources")]
    public Task<IReadOnlyList<CoolifyResourceResponse>> Resources(string uuid, CancellationToken cancellationToken) =>
        coolify.GetServerResourcesAsync(uuid, cancellationToken);

    [HttpGet("applications")]
    public Task<IReadOnlyList<CoolifyApplicationResponse>> Applications(CancellationToken cancellationToken) =>
        coolify.GetApplicationsAsync(cancellationToken);

    [HttpGet("services")]
    public Task<IReadOnlyList<CoolifyServiceResponse>> Services(CancellationToken cancellationToken) =>
        coolify.GetServicesAsync(cancellationToken);

    [HttpGet("deployments")]
    public Task<PagedResponse<DeploymentResponse>> Deployments(
        [FromQuery] int page = 1, [FromQuery] int pageSize = 30, CancellationToken cancellationToken = default) =>
        coolify.GetDeploymentsAsync(page, pageSize, cancellationToken);

    [HttpGet("deployments/{uuid}")]
    public async Task<ActionResult<DeploymentResponse>> Deployment(string uuid, CancellationToken cancellationToken)
    {
        var deployment = await coolify.GetDeploymentAsync(uuid, cancellationToken);
        return deployment is null ? NotFound() : deployment;
    }

    [HttpPost("applications/{uuid}/start")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> StartApplication(string uuid, CancellationToken cancellationToken) =>
        PerformAsync("coolify.application.start", uuid, coolify.StartApplicationAsync, cancellationToken);

    [HttpPost("applications/{uuid}/stop")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> StopApplication(string uuid, CancellationToken cancellationToken) =>
        PerformAsync("coolify.application.stop", uuid, coolify.StopApplicationAsync, cancellationToken);

    [HttpPost("applications/{uuid}/restart")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> RestartApplication(string uuid, CancellationToken cancellationToken) =>
        PerformAsync("coolify.application.restart", uuid, coolify.RestartApplicationAsync, cancellationToken);

    [HttpPost("applications/{uuid}/redeploy")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> RedeployApplication(string uuid, CancellationToken cancellationToken) =>
        PerformAsync("coolify.application.redeploy", uuid, coolify.RedeployApplicationAsync, cancellationToken);

    [HttpPost("services/{uuid}/restart")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> RestartService(string uuid, CancellationToken cancellationToken) =>
        PerformAsync("coolify.service.restart", uuid, coolify.RestartServiceAsync, cancellationToken);

    private async Task<ActionResult<ActionResponse>> PerformAsync(
        string action, string uuid, Func<string, CancellationToken, Task<ActionResponse>> operation,
        CancellationToken cancellationToken)
    {
        try
        {
            var response = await operation(uuid, cancellationToken);
            await RecordAsync(action, uuid, OperationResult.Succeeded, null, cancellationToken);
            return Accepted(response);
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            await RecordAsync(action, uuid, OperationResult.Failed, exception.Message, cancellationToken);
            throw;
        }
    }

    private Task RecordAsync(string action, string target, OperationResult result, string? detail,
        CancellationToken cancellationToken) => audit.RecordAsync(
        Guid.Parse(User.FindFirstValue(ClaimTypes.NameIdentifier)!),
        User.FindFirstValue(ClaimTypes.Email), action, target, result,
        HttpContext.TraceIdentifier, detail, cancellationToken);
}
