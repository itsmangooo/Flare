namespace Flare.Api.Data;

public sealed class MetricSample
{
    public long Id { get; set; }
    public DateTimeOffset Timestamp { get; set; }
    public double? CpuPercent { get; set; }
    public double? MemoryPercent { get; set; }
}
