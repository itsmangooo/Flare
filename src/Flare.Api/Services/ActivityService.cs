using Flare.Api.Data;
using Flare.Contracts;
using Microsoft.EntityFrameworkCore;

namespace Flare.Api.Services;

public sealed class ActivityService(FlareDbContext database) : IActivityService
{
    public async Task<PagedResponse<ActivityEventResponse>> GetAsync(
        int page, int pageSize, CancellationToken cancellationToken)
    {
        page = Math.Max(page, 1);
        pageSize = Math.Clamp(pageSize, 1, 100);
        var take = page * pageSize + 1;
        var audit = await database.AuditEvents.AsNoTracking()
            .OrderByDescending(item => item.Timestamp).Take(take).ToArrayAsync(cancellationToken);
        var infrastructure = await database.InfrastructureEvents.AsNoTracking()
            .OrderByDescending(item => item.Timestamp).Take(take).ToArrayAsync(cancellationToken);

        var combined = audit.Select(item => new ActivityEventResponse(
                item.Id,
                item.Action.StartsWith("auth.", StringComparison.Ordinal) ? ActivityKind.Security : ActivityKind.Infrastructure,
                DisplayAction(item.Action),
                item.Target,
                item.Timestamp,
                item.Result,
                item.Actor))
            .Concat(infrastructure.Select(item => new ActivityEventResponse(
                item.Id, ActivityKind.Infrastructure, DisplayAction(item.Action), item.Target,
                item.Timestamp, item.Result, null)))
            .OrderByDescending(item => item.Timestamp)
            .ToArray();
        var offset = (page - 1) * pageSize;
        var items = combined.Skip(offset).Take(pageSize).ToArray();
        return new PagedResponse<ActivityEventResponse>(items, page, pageSize, combined.Length > offset + items.Length);
    }

    private static string DisplayAction(string action) => action switch
    {
        "auth.bootstrap" => "Administrator bootstrapped",
        "auth.login" => "Login",
        "auth.logout" => "Logout",
        "auth.refresh" => "Session refreshed",
        "container.start" => "Container started",
        "container.stop" => "Container stopped",
        "container.restart" => "Container restarted",
        "container.state.changed" => "Container state changed",
        "coolify.application.start" => "Application started",
        "coolify.application.stop" => "Application stopped",
        "coolify.application.restart" => "Application restarted",
        "coolify.application.redeploy" => "Deployment triggered",
        "coolify.service.restart" => "Service restarted",
        "deployment.failed" => "Deployment failed",
        "deployment.status.changed" => "Deployment status changed",
        "server.disconnected" => "Server disconnected",
        "server.reconnected" => "Server reconnected",
        _ => action
    };
}
