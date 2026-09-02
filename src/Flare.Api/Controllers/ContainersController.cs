using System.Security.Claims;
using Flare.Api.Services;
using Flare.Contracts;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.RateLimiting;

namespace Flare.Api.Controllers;

[ApiController]
[Authorize]
[Route("api/v1/containers")]
public sealed class ContainersController(IDockerService docker, IAuditService audit) : ControllerBase
{
    [HttpGet]
    public Task<IReadOnlyList<ContainerSummaryResponse>> Get(CancellationToken cancellationToken) =>
        docker.GetContainersAsync(cancellationToken);

    [HttpGet("{id}")]
    public async Task<ActionResult<ContainerDetailResponse>> GetById(
        string id, CancellationToken cancellationToken)
    {
        var container = await docker.GetContainerAsync(id, cancellationToken);
        return container is null ? NotFound() : container;
    }

    [HttpGet("{id}/logs")]
    public Task<LogPageResponse> Logs(
        string id, [FromQuery] int tail = 300, [FromQuery] DateTimeOffset? before = null,
        CancellationToken cancellationToken = default) =>
        docker.GetLogsAsync(id, tail, before, cancellationToken);

    [HttpPost("{id}/start")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> Start(string id, CancellationToken cancellationToken) =>
        PerformAsync("container.start", id, token => docker.StartAsync(id, token), cancellationToken);

    [HttpPost("{id}/stop")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> Stop(string id, CancellationToken cancellationToken) =>
        PerformAsync("container.stop", id, token => docker.StopAsync(id, token), cancellationToken);

    [HttpPost("{id}/restart")]
    [Authorize(Policy = "Administrator")]
    [EnableRateLimiting("operations")]
    public Task<ActionResult<ActionResponse>> Restart(string id, CancellationToken cancellationToken) =>
        PerformAsync("container.restart", id, token => docker.RestartAsync(id, token), cancellationToken);

    private async Task<ActionResult<ActionResponse>> PerformAsync(
        string action, string id, Func<CancellationToken, Task> operation, CancellationToken cancellationToken)
    {
        try
        {
            await operation(cancellationToken);
            await RecordAsync(action, id, OperationResult.Succeeded, null, cancellationToken);
            return Accepted(new ActionResponse("Container operation accepted."));
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            await RecordAsync(action, id, OperationResult.Failed, exception.Message, cancellationToken);
            throw;
        }
    }

    private Task RecordAsync(string action, string target, OperationResult result, string? detail,
        CancellationToken cancellationToken) => audit.RecordAsync(
        Guid.Parse(User.FindFirstValue(ClaimTypes.NameIdentifier)!),
        User.FindFirstValue(ClaimTypes.Email), action, target, result,
        HttpContext.TraceIdentifier, detail, cancellationToken);
}
