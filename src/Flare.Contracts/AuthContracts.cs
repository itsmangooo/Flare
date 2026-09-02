using System.ComponentModel.DataAnnotations;

namespace Flare.Contracts;

public sealed record BootstrapRequest(
    [Required, EmailAddress, MaxLength(254)] string Email,
    [Required, MinLength(12), MaxLength(128)] string Password,
    [Required, MaxLength(512)] string BootstrapToken);

public sealed record LoginRequest(
    [Required, EmailAddress, MaxLength(254)] string Email,
    [Required, MaxLength(128)] string Password);

public sealed record RefreshRequest([Required, MaxLength(512)] string RefreshToken);

public sealed record LogoutRequest([Required, MaxLength(512)] string RefreshToken);

public sealed record TokenResponse(
    string AccessToken,
    DateTimeOffset AccessTokenExpiresAt,
    string RefreshToken,
    DateTimeOffset RefreshTokenExpiresAt,
    UserResponse User);

public sealed record UserResponse(Guid Id, string Email, IReadOnlyList<string> Roles);

public sealed record BootstrapStatusResponse(bool Required);
