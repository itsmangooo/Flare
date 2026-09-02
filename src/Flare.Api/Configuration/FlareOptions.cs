namespace Flare.Api.Configuration;

public sealed class JwtOptions
{
    public const string SectionName = "Jwt";
    public string Issuer { get; init; } = "Flare.Api";
    public string Audience { get; init; } = "Flare.Mobile";
    public int AccessTokenMinutes { get; init; } = 15;
    public int RefreshTokenDays { get; init; } = 30;
    public required string SigningKey { get; init; }
}

public sealed class HostMetricsOptions
{
    public string HostName { get; set; } = "homelab";
    public string ProcPath { get; set; } = "/host/proc";
    public string RootFileSystemPath { get; set; } = "/host/rootfs";
}

public sealed class CoolifyOptions
{
    public string? BaseUrl { get; set; }
    public string? ApiToken { get; set; }
    public bool IsConfigured => Uri.TryCreate(BaseUrl, UriKind.Absolute, out var uri)
        && uri.Scheme == Uri.UriSchemeHttps && !string.IsNullOrWhiteSpace(ApiToken);
}
