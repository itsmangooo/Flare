using System.Text.Json;
using Microsoft.Extensions.Diagnostics.HealthChecks;

namespace Flare.Api.Infrastructure;

public static class HealthResponseWriter
{
    public static Task WriteAsync(HttpContext context, HealthReport report)
    {
        context.Response.ContentType = "application/json";
        var payload = new
        {
            status = report.Status.ToString().ToLowerInvariant(),
            checks = report.Entries.ToDictionary(
                entry => entry.Key,
                entry => new { status = entry.Value.Status.ToString().ToLowerInvariant() })
        };
        return context.Response.WriteAsync(JsonSerializer.Serialize(payload));
    }
}
