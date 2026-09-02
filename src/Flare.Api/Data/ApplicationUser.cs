using Microsoft.AspNetCore.Identity;

namespace Flare.Api.Data;

public sealed class ApplicationUser : IdentityUser<Guid>
{
    public DateTimeOffset CreatedAt { get; set; } = DateTimeOffset.UtcNow;
    public ICollection<RefreshToken> RefreshTokens { get; set; } = [];
}
