using System.Data;
using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Security.Cryptography;
using System.Text;
using Flare.Api.Configuration;
using Flare.Api.Data;
using Flare.Contracts;
using Microsoft.AspNetCore.Identity;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Options;
using Microsoft.IdentityModel.Tokens;

namespace Flare.Api.Services;

public sealed class TokenService(
    FlareDbContext database,
    UserManager<ApplicationUser> userManager,
    IOptions<JwtOptions> options,
    TimeProvider timeProvider) : ITokenService
{
    private readonly JwtOptions _options = options.Value;

    public async Task<TokenResponse> IssueAsync(
        ApplicationUser user, string? clientIp, CancellationToken cancellationToken)
    {
        var now = timeProvider.GetUtcNow();
        var refresh = CreateRefreshToken(user.Id, Guid.NewGuid(), now, clientIp);
        database.RefreshTokens.Add(refresh.Entity);
        await database.SaveChangesAsync(cancellationToken);
        return await CreateResponseAsync(user, refresh.PlainText, refresh.Entity.ExpiresAt, now);
    }

    public async Task<TokenResponse> RotateAsync(
        string refreshToken, string? clientIp, CancellationToken cancellationToken)
    {
        var now = timeProvider.GetUtcNow();
        var hash = Hash(refreshToken);
        await using var transaction = await database.Database.BeginTransactionAsync(
            IsolationLevel.Serializable, cancellationToken);

        var current = await database.RefreshTokens
            .Include(token => token.User)
            .SingleOrDefaultAsync(token => token.TokenHash == hash, cancellationToken)
            ?? throw new SecurityTokenException("Refresh token is invalid.");

        if (!current.IsActive(now))
        {
            if (current.RevokedAt is not null)
            {
                await database.RefreshTokens
                    .Where(token => token.UserId == current.UserId && token.FamilyId == current.FamilyId
                        && token.RevokedAt == null)
                    .ExecuteUpdateAsync(setters => setters
                        .SetProperty(token => token.RevokedAt, now)
                        .SetProperty(token => token.RevokedByIp, NormalizeIp(clientIp)), cancellationToken);
                await transaction.CommitAsync(cancellationToken);
            }

            throw new SecurityTokenException("Refresh token has expired or was already used.");
        }

        var replacement = CreateRefreshToken(current.UserId, current.FamilyId, now, clientIp);
        current.RevokedAt = now;
        current.RevokedByIp = NormalizeIp(clientIp);
        current.ReplacedById = replacement.Entity.Id;
        database.RefreshTokens.Add(replacement.Entity);
        await database.SaveChangesAsync(cancellationToken);
        await transaction.CommitAsync(cancellationToken);

        return await CreateResponseAsync(current.User, replacement.PlainText, replacement.Entity.ExpiresAt, now);
    }

    public async Task<bool> RevokeAsync(
        string refreshToken, string? clientIp, CancellationToken cancellationToken)
    {
        var hash = Hash(refreshToken);
        var token = await database.RefreshTokens.SingleOrDefaultAsync(
            candidate => candidate.TokenHash == hash, cancellationToken);
        if (token is null || token.RevokedAt is not null)
        {
            return false;
        }

        token.RevokedAt = timeProvider.GetUtcNow();
        token.RevokedByIp = NormalizeIp(clientIp);
        await database.SaveChangesAsync(cancellationToken);
        return true;
    }

    private async Task<TokenResponse> CreateResponseAsync(
        ApplicationUser user, string refreshToken, DateTimeOffset refreshExpiresAt, DateTimeOffset now)
    {
        var roles = await userManager.GetRolesAsync(user);
        var accessExpiresAt = now.AddMinutes(_options.AccessTokenMinutes);
        var claims = new List<Claim>
        {
            new(JwtRegisteredClaimNames.Sub, user.Id.ToString()),
            new(JwtRegisteredClaimNames.Email, user.Email!),
            new(JwtRegisteredClaimNames.Jti, Guid.NewGuid().ToString("N")),
            new(ClaimTypes.NameIdentifier, user.Id.ToString()),
            new(ClaimTypes.Email, user.Email!)
        };
        claims.AddRange(roles.Select(role => new Claim(ClaimTypes.Role, role)));

        var credentials = new SigningCredentials(
            new SymmetricSecurityKey(Encoding.UTF8.GetBytes(_options.SigningKey)),
            SecurityAlgorithms.HmacSha256);
        var jwt = new JwtSecurityToken(
            issuer: _options.Issuer,
            audience: _options.Audience,
            claims: claims,
            notBefore: now.UtcDateTime,
            expires: accessExpiresAt.UtcDateTime,
            signingCredentials: credentials);

        return new TokenResponse(
            new JwtSecurityTokenHandler().WriteToken(jwt),
            accessExpiresAt,
            refreshToken,
            refreshExpiresAt,
            new UserResponse(user.Id, user.Email!, roles.ToArray()));
    }

    private (RefreshToken Entity, string PlainText) CreateRefreshToken(
        Guid userId, Guid familyId, DateTimeOffset now, string? clientIp)
    {
        var plainText = Base64UrlEncoder.Encode(RandomNumberGenerator.GetBytes(64));
        return (new RefreshToken
        {
            Id = Guid.NewGuid(),
            TokenHash = Hash(plainText),
            FamilyId = familyId,
            UserId = userId,
            CreatedAt = now,
            ExpiresAt = now.AddDays(_options.RefreshTokenDays),
            CreatedByIp = NormalizeIp(clientIp)
        }, plainText);
    }

    private static string Hash(string token) => Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes(token)));
    private static string? NormalizeIp(string? ip) => ip is { Length: > 64 } ? ip[..64] : ip;
}
