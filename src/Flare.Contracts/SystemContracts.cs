namespace Flare.Contracts;

public sealed record ServerInfoResponse(string Name, string ApiVersion, string ServerVersion, DateTimeOffset ServerTime);
