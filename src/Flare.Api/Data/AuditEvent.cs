using Flare.Contracts;

namespace Flare.Api.Data;

public sealed class AuditEvent
{
    public Guid Id { get; set; }
    public Guid? UserId { get; set; }
    public string? Actor { get; set; }
    public required string Action { get; set; }
    public required string Target { get; set; }
    public DateTimeOffset Timestamp { get; set; }
    public OperationResult Result { get; set; }
    public required string CorrelationId { get; set; }
    public string? Detail { get; set; }
}
