using Flare.Contracts;

namespace Flare.Api.Services;

public interface IAuditService
{
    Task RecordAsync(Guid? userId, string? actor, string action, string target,
        OperationResult result, string correlationId, string? detail, CancellationToken cancellationToken);
}
