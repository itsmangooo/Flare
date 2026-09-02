namespace Flare.Contracts;

public sealed record CoolifyServerResponse(string Uuid, string Name, bool? IsReachable, bool? IsUsable);
public sealed record CoolifyResourceResponse(string Uuid, string Name, string Type, string? Status);
public sealed record CoolifyApplicationResponse(string Uuid, string Name, string? Status, string? Fqdn, string? GitBranch);
public sealed record CoolifyServiceResponse(string Uuid, string Name, string? Status, string? Description);

public sealed record DeploymentResponse(
    string Uuid,
    string ResourceUuid,
    string ResourceName,
    string? Status,
    string? Branch,
    string? Commit,
    string? CommitMessage,
    DateTimeOffset? StartedAt,
    DateTimeOffset? FinishedAt,
    TimeSpan? Duration,
    string? Logs);

public sealed record PagedResponse<T>(IReadOnlyList<T> Items, int Page, int PageSize, bool HasMore);
