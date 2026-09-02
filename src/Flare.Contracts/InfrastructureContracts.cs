namespace Flare.Contracts;

public enum DataFreshness { Live, Reconnecting, Stale, Offline }
public enum ContainerState { Running, Stopped, Paused, Restarting, Dead, Unknown }
public enum HealthState { Healthy, Unhealthy, Starting, None, Unknown }
public enum OperationResult { Succeeded, Failed }
public enum ActivityKind { Infrastructure, Security }

public sealed record HostMetricsResponse(
    string HostName,
    DateTimeOffset ObservedAt,
    double? CpuPercent,
    double? LoadAverage,
    long? MemoryUsedBytes,
    long? MemoryTotalBytes,
    long? DiskUsedBytes,
    long? DiskTotalBytes,
    double? NetworkReceiveBytesPerSecond,
    double? NetworkTransmitBytesPerSecond,
    TimeSpan? Uptime);

public sealed record MetricPoint(DateTimeOffset Timestamp, double? CpuPercent, double? MemoryPercent);

public sealed record ContainerSummaryResponse(
    string Id,
    string Name,
    string Image,
    ContainerState State,
    HealthState Health,
    DateTimeOffset? StartedAt,
    double? CpuPercent,
    long? MemoryBytes);

public sealed record ContainerTotalsResponse(int? Running, int? Stopped, int? Unhealthy);

public sealed record PortBindingResponse(int PrivatePort, int? PublicPort, string Protocol, string? HostIp);

public sealed record ContainerDetailResponse(
    string Id,
    string ShortId,
    string Name,
    string Image,
    ContainerState State,
    HealthState Health,
    DateTimeOffset CreatedAt,
    DateTimeOffset? StartedAt,
    int RestartCount,
    double? CpuPercent,
    long? MemoryBytes,
    long? MemoryLimitBytes,
    long? NetworkReceiveBytes,
    long? NetworkTransmitBytes,
    IReadOnlyList<PortBindingResponse> Ports,
    IReadOnlyDictionary<string, string> Labels);

public sealed record LogPageResponse(
    IReadOnlyList<string> Lines,
    DateTimeOffset RetrievedAt,
    bool Truncated,
    int RequestedTail,
    DateTimeOffset? OldestTimestamp,
    DateTimeOffset? NewestTimestamp);

public sealed record ActionResponse(string Message, string? OperationId = null);

public sealed record OverviewResponse(
    DateTimeOffset GeneratedAt,
    DataFreshness Freshness,
    HostMetricsResponse Host,
    ContainerTotalsResponse Containers,
    IReadOnlyList<MetricPoint> History,
    IReadOnlyList<ActivityEventResponse> RecentActivity);

public sealed record ActivityEventResponse(
    Guid Id,
    ActivityKind Kind,
    string Action,
    string Target,
    DateTimeOffset Timestamp,
    OperationResult Result,
    string? Actor);
