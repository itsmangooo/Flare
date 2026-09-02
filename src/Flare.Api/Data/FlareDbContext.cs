using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Identity.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore;

namespace Flare.Api.Data;

public sealed class FlareDbContext(DbContextOptions<FlareDbContext> options)
    : IdentityDbContext<ApplicationUser, IdentityRole<Guid>, Guid>(options)
{
    public DbSet<RefreshToken> RefreshTokens => Set<RefreshToken>();
    public DbSet<AuditEvent> AuditEvents => Set<AuditEvent>();
    public DbSet<InfrastructureEvent> InfrastructureEvents => Set<InfrastructureEvent>();
    public DbSet<MetricSample> MetricSamples => Set<MetricSample>();

    protected override void OnModelCreating(ModelBuilder builder)
    {
        base.OnModelCreating(builder);

        builder.Entity<ApplicationUser>(entity =>
        {
            entity.ToTable("Users");
            entity.HasMany(x => x.RefreshTokens).WithOne(x => x.User).HasForeignKey(x => x.UserId);
        });
        builder.Entity<IdentityRole<Guid>>().ToTable("Roles");
        builder.Entity<IdentityUserRole<Guid>>().ToTable("UserRoles");
        builder.Entity<IdentityUserClaim<Guid>>().ToTable("UserClaims");
        builder.Entity<IdentityUserLogin<Guid>>().ToTable("UserLogins");
        builder.Entity<IdentityRoleClaim<Guid>>().ToTable("RoleClaims");
        builder.Entity<IdentityUserToken<Guid>>().ToTable("UserTokens");

        builder.Entity<RefreshToken>(entity =>
        {
            entity.ToTable("RefreshTokens");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => x.TokenHash).IsUnique();
            entity.HasIndex(x => new { x.UserId, x.FamilyId });
            entity.Property(x => x.TokenHash).HasMaxLength(64);
            entity.Property(x => x.CreatedByIp).HasMaxLength(64);
            entity.Property(x => x.RevokedByIp).HasMaxLength(64);
        });

        builder.Entity<AuditEvent>(entity =>
        {
            entity.ToTable("AuditEvents");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => x.Timestamp);
            entity.Property(x => x.Action).HasMaxLength(100);
            entity.Property(x => x.Target).HasMaxLength(256);
            entity.Property(x => x.Actor).HasMaxLength(254);
            entity.Property(x => x.CorrelationId).HasMaxLength(128);
            entity.Property(x => x.Detail).HasMaxLength(1000);
        });

        builder.Entity<InfrastructureEvent>(entity =>
        {
            entity.ToTable("InfrastructureEvents");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => x.Timestamp);
            entity.Property(x => x.Action).HasMaxLength(100);
            entity.Property(x => x.Target).HasMaxLength(256);
            entity.Property(x => x.Detail).HasMaxLength(1000);
        });

        builder.Entity<MetricSample>(entity =>
        {
            entity.ToTable("MetricSamples");
            entity.HasKey(x => x.Id);
            entity.HasIndex(x => x.Timestamp);
        });
    }
}
