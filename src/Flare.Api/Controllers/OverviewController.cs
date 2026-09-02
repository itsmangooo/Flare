using Flare.Api.Services;
using Flare.Contracts;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Flare.Api.Controllers;

[ApiController]
[Authorize]
[Route("api/v1/overview")]
public sealed class OverviewController(IOverviewService overview) : ControllerBase
{
    [HttpGet]
    public Task<OverviewResponse> Get(CancellationToken cancellationToken) => overview.GetAsync(cancellationToken);
}
