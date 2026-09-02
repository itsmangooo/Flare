using Flare.Api.Configuration;
using Flare.Api.Data;
using Flare.Api.Services;
using Microsoft.AspNetCore.Identity;
using Microsoft.Data.Sqlite;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Options;
using Microsoft.IdentityModel.Tokens;

namespace Flare.Api.Tests;

public sealed class TokenServiceTests : IAsyncLifetime, IDisposable
{
    private readonly SqliteConnection _connection = new("Data Source=:memory:");
    private ServiceProvider _provider = null!;

    public async Task InitializeAsync()
    {
        await _connection.OpenAsync();
        var services = new ServiceCollection();
        services.AddLogging();
        services.AddSingleton(TimeProvider.System);
        services.AddSingleton(Options.Create(new JwtOptions
        {
            SigningKey = "a-test-key-that-is-long-enough-for-hmac-sha256-only",
            AccessTokenMinutes = 15,
            RefreshTokenDays = 30
        }));
        services.AddDbContext<FlareDbContext>(options => options.UseSqlite(_connection));
        services.AddIdentityCore<ApplicationUser>().AddRoles<IdentityRole<Guid>>()
            .AddEntityFrameworkStores<FlareDbContext>();
        services.AddScoped<ITokenService, TokenService>();
        _provider = services.BuildServiceProvider();
        await using var scope = _provider.CreateAsyncScope();
        await scope.ServiceProvider.GetRequiredService<FlareDbContext>().Database.EnsureCreatedAsync();
    }

    public async Task DisposeAsync()
    {
        await _provider.DisposeAsync();
    }

    public void Dispose() => _connection.Dispose();

    [Fact]
    public async Task RefreshTokensAreHashedRotatedAndReuseRevokesTheFamily()
    {
        string firstPlainText;
        string secondPlainText;
        Guid familyId;
        await using (var scope = _provider.CreateAsyncScope())
        {
            var manager = scope.ServiceProvider.GetRequiredService<UserManager<ApplicationUser>>();
            var user = new ApplicationUser
            {
                Id = Guid.NewGuid(),
                UserName = "admin@example.test",
                Email = "admin@example.test",
                EmailConfirmed = true
            };
            Assert.True((await manager.CreateAsync(user, "Valid-password-123!")).Succeeded);
            var service = scope.ServiceProvider.GetRequiredService<ITokenService>();
            var issued = await service.IssueAsync(user, "127.0.0.1", CancellationToken.None);
            firstPlainText = issued.RefreshToken;
            var rotated = await service.RotateAsync(firstPlainText, "127.0.0.1", CancellationToken.None);
            secondPlainText = rotated.RefreshToken;
            var database = scope.ServiceProvider.GetRequiredService<FlareDbContext>();
            var tokens = (await database.RefreshTokens.ToArrayAsync()).OrderBy(token => token.CreatedAt).ToArray();
            Assert.Equal(2, tokens.Length);
            Assert.DoesNotContain(tokens, token => token.TokenHash == firstPlainText || token.TokenHash == secondPlainText);
            Assert.NotNull(tokens[0].RevokedAt);
            Assert.Equal(tokens[1].Id, tokens[0].ReplacedById);
            familyId = tokens[0].FamilyId;
        }

        await using (var scope = _provider.CreateAsyncScope())
        {
            var service = scope.ServiceProvider.GetRequiredService<ITokenService>();
            await Assert.ThrowsAsync<SecurityTokenException>(() => service.RotateAsync(
                firstPlainText, "127.0.0.1", CancellationToken.None));
        }

        await using (var scope = _provider.CreateAsyncScope())
        {
            var database = scope.ServiceProvider.GetRequiredService<FlareDbContext>();
            Assert.All(await database.RefreshTokens.Where(token => token.FamilyId == familyId).ToArrayAsync(),
                token => Assert.NotNull(token.RevokedAt));
        }
    }
}
