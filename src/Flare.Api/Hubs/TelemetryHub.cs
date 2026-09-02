using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.SignalR;

namespace Flare.Api.Hubs;

[Authorize]
public sealed class TelemetryHub : Hub;
