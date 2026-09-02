using Flare.Api.Data;
using Flare.Contracts;

namespace Flare.Api.Services;

public sealed class AuditService(FlareDbContext database, TimeProvider timeProvider) : IAuditService
{
    public async Task RecordAsync(Guid? userId, string? actor, string action, string target,
        OperationResult result, string correlationId, string? detail, CancellationToken cancellationToken)
    {
        database.AuditEvents.Add(new AuditEvent
        {
            Id = Guid.NewGuid(),
            UserId = userId,
            Actor = Truncate(actor, 254),
            Action = Truncate(action, 100)!,
            Target = Truncate(target, 256)!,
            Timestamp = timeProvider.GetUtcNow(),
            Result = result,
            CorrelationId = Truncate(correlationId, 128)!,
            Detail = Truncate(detail, 1000)
        });
        await database.SaveChangesAsync(cancellationToken);
    }

    private static string? Truncate(string? value, int length) => string.IsNullOrWhiteSpace(value)
        ? value
        : value.Length <= length ? value : value[..length];
}
