using Flare.Contracts;

namespace Flare.Api.Data;

public sealed class InfrastructureEvent
{
    public Guid Id { get; set; }
    public required string Action { get; set; }
    public required string Target { get; set; }
    public DateTimeOffset Timestamp { get; set; }
    public OperationResult Result { get; set; }
    public string? Detail { get; set; }
}
