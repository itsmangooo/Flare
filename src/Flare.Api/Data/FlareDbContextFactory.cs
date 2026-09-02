using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Design;

namespace Flare.Api.Data;

public sealed class FlareDbContextFactory : IDesignTimeDbContextFactory<FlareDbContext>
{
    public FlareDbContext CreateDbContext(string[] args)
    {
        var connectionString = Environment.GetEnvironmentVariable("ConnectionStrings__Postgres")
            ?? "Host=localhost;Database=flare_design;Username=flare_design;Password=not-used-for-migration-generation";
        var builder = new DbContextOptionsBuilder<FlareDbContext>();
        builder.UseNpgsql(connectionString);
        return new FlareDbContext(builder.Options);
    }
}
