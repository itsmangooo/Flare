using Flare.Api.Services;
using Flare.Contracts;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Flare.Api.Controllers;

[ApiController]
[Authorize]
[Route("api/v1/activity")]
public sealed class ActivityController(IActivityService activity) : ControllerBase
{
    [HttpGet]
    public Task<PagedResponse<ActivityEventResponse>> Get(
        [FromQuery] int page = 1, [FromQuery] int pageSize = 30, CancellationToken cancellationToken = default) =>
        activity.GetAsync(page, pageSize, cancellationToken);
}
