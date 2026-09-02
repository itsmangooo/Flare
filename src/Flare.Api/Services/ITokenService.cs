using Flare.Api.Data;
using Flare.Contracts;

namespace Flare.Api.Services;

public interface ITokenService
{
    Task<TokenResponse> IssueAsync(ApplicationUser user, string? clientIp, CancellationToken cancellationToken);
    Task<TokenResponse> RotateAsync(string refreshToken, string? clientIp, CancellationToken cancellationToken);
    Task<bool> RevokeAsync(string refreshToken, string? clientIp, CancellationToken cancellationToken);
}
