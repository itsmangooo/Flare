using System.Reflection;
using Flare.Contracts;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Flare.Api.Controllers;

[ApiController]
[Route("api/v1/system")]
public sealed class SystemController(TimeProvider timeProvider) : ControllerBase
{
    [HttpGet("info")]
    [Authorize]
    public ServerInfoResponse Info() => new(
        "Flare",
        "v1",
        Assembly.GetExecutingAssembly().GetName().Version?.ToString(3) ?? "unknown",
        timeProvider.GetUtcNow());
}
