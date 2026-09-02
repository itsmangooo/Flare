using System.Security.Claims;
using System.Text;
using System.Text.Json.Serialization;
using System.Threading.RateLimiting;
using Flare.Api.Configuration;
using Flare.Api.Data;
using Flare.Api.Hubs;
using Flare.Api.Infrastructure;
using Flare.Api.Services;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.AspNetCore.Diagnostics.HealthChecks;
using Microsoft.AspNetCore.HttpOverrides;
using Microsoft.AspNetCore.Http.Timeouts;
using Microsoft.AspNetCore.Identity;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Diagnostics.HealthChecks;
using Microsoft.IdentityModel.Tokens;
using Npgsql;

if (args.Contains("--healthcheck", StringComparer.Ordinal))
{
    using var probe = new HttpClient { Timeout = TimeSpan.FromSeconds(3) };
    try
    {
        using var response = await probe.GetAsync("http://127.0.0.1:8080/health/live");
        Environment.ExitCode = response.IsSuccessStatusCode ? 0 : 1;
    }
    catch (HttpRequestException) { Environment.ExitCode = 1; }
    catch (TaskCanceledException) { Environment.ExitCode = 1; }
    return;
}

var builder = WebApplication.CreateBuilder(args);
var connectionString = builder.Configuration.GetConnectionString("Postgres")
    ?? throw new InvalidOperationException("ConnectionStrings__Postgres is required.");
var postgresConnection = new NpgsqlConnectionStringBuilder(connectionString)
{
    // The chiseled runtime intentionally omits Kerberos. PostgreSQL TLS remains controlled by SSL Mode.
    GssEncryptionMode = GssEncryptionMode.Disable
}.ConnectionString;
var signingKey = builder.Configuration["FLARE_JWT_SIGNING_KEY"];
if (string.IsNullOrWhiteSpace(signingKey) && builder.Environment.IsDevelopment())
{
    signingKey = "development-only-key-replace-with-FLARE_JWT_SIGNING_KEY";
}
if (string.IsNullOrWhiteSpace(signingKey) || Encoding.UTF8.GetByteCount(signingKey) < 32)
{
    throw new InvalidOperationException("FLARE_JWT_SIGNING_KEY must contain at least 32 UTF-8 bytes.");
}

var jwtOptions = new JwtOptions
{
    SigningKey = signingKey,
    Issuer = builder.Configuration["FLARE_JWT_ISSUER"] ?? "Flare.Api",
    Audience = builder.Configuration["FLARE_JWT_AUDIENCE"] ?? "Flare.Mobile",
    AccessTokenMinutes = Math.Clamp(builder.Configuration.GetValue("FLARE_ACCESS_TOKEN_MINUTES", 15), 5, 60),
    RefreshTokenDays = Math.Clamp(builder.Configuration.GetValue("FLARE_REFRESH_TOKEN_DAYS", 30), 1, 90)
};
builder.Services.AddSingleton(Microsoft.Extensions.Options.Options.Create(jwtOptions));
builder.Services.Configure<HostMetricsOptions>(options =>
{
    options.HostName = builder.Configuration["FLARE_HOST_NAME"] ?? "homelab";
    options.ProcPath = builder.Configuration["HOST_PROC_PATH"] ?? "/host/proc";
    options.RootFileSystemPath = builder.Configuration["HOST_ROOTFS_PATH"] ?? "/host/rootfs";
});
builder.Services.Configure<CoolifyOptions>(options =>
{
    options.BaseUrl = builder.Configuration["COOLIFY_BASE_URL"];
    options.ApiToken = builder.Configuration["COOLIFY_API_TOKEN"];
});

builder.Services.AddDbContext<FlareDbContext>(options => options.UseNpgsql(postgresConnection));
builder.Services.AddIdentityCore<ApplicationUser>(options =>
    {
        options.Password.RequiredLength = 12;
        options.Password.RequireDigit = true;
        options.Password.RequireLowercase = true;
        options.Password.RequireUppercase = true;
        options.Password.RequireNonAlphanumeric = true;
        options.Password.RequiredUniqueChars = 6;
        options.Lockout.AllowedForNewUsers = true;
        options.Lockout.MaxFailedAccessAttempts = 5;
        options.Lockout.DefaultLockoutTimeSpan = TimeSpan.FromMinutes(15);
        options.User.RequireUniqueEmail = true;
    })
    .AddRoles<IdentityRole<Guid>>()
    .AddSignInManager()
    .AddEntityFrameworkStores<FlareDbContext>()
    .AddDefaultTokenProviders();

builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.MapInboundClaims = false;
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidIssuer = jwtOptions.Issuer,
            ValidateAudience = true,
            ValidAudience = jwtOptions.Audience,
            ValidateLifetime = true,
            ValidateIssuerSigningKey = true,
            IssuerSigningKey = new SymmetricSecurityKey(Encoding.UTF8.GetBytes(jwtOptions.SigningKey)),
            ClockSkew = TimeSpan.FromSeconds(30),
            NameClaimType = ClaimTypes.Email,
            RoleClaimType = ClaimTypes.Role
        };
        options.Events = new JwtBearerEvents
        {
            OnMessageReceived = context =>
            {
                if (context.HttpContext.Request.Path.StartsWithSegments("/hubs/telemetry")
                    && context.Request.Query.TryGetValue("access_token", out var token))
                {
                    context.Token = token;
                }
                return Task.CompletedTask;
            }
        };
    });
builder.Services.AddAuthorizationBuilder()
    .AddPolicy("Administrator", policy => policy.RequireRole("Administrator"));

builder.Services.AddRateLimiter(options =>
{
    options.RejectionStatusCode = StatusCodes.Status429TooManyRequests;
    options.GlobalLimiter = PartitionedRateLimiter.Create<HttpContext, string>(context =>
        RateLimitPartition.GetFixedWindowLimiter(context.Connection.RemoteIpAddress?.ToString() ?? "unknown",
            _ => new FixedWindowRateLimiterOptions
            {
                PermitLimit = 300,
                Window = TimeSpan.FromMinutes(1),
                QueueLimit = 0,
                AutoReplenishment = true
            }));
    options.AddPolicy("auth", context => RateLimitPartition.GetFixedWindowLimiter(
        context.Connection.RemoteIpAddress?.ToString() ?? "unknown",
        _ => new FixedWindowRateLimiterOptions
        {
            PermitLimit = 10,
            Window = TimeSpan.FromMinutes(1),
            QueueLimit = 0,
            AutoReplenishment = true
        }));
    options.AddPolicy("operations", context => RateLimitPartition.GetFixedWindowLimiter(
        $"{context.User.FindFirstValue(ClaimTypes.NameIdentifier) ?? "anonymous"}:{context.Connection.RemoteIpAddress}",
        _ => new FixedWindowRateLimiterOptions
        {
            PermitLimit = 30,
            Window = TimeSpan.FromMinutes(1),
            QueueLimit = 0,
            AutoReplenishment = true
        }));
});

builder.Services.AddProblemDetails();
builder.Services.AddExceptionHandler<GlobalExceptionHandler>();
builder.Services.AddRequestTimeouts(options => options.DefaultPolicy = new RequestTimeoutPolicy
{
    Timeout = TimeSpan.FromSeconds(30),
    TimeoutStatusCode = StatusCodes.Status503ServiceUnavailable
});
builder.Services.AddControllers().AddJsonOptions(options =>
    options.JsonSerializerOptions.Converters.Add(new JsonStringEnumConverter()));
builder.Services.AddSignalR().AddJsonProtocol(options =>
    options.PayloadSerializerOptions.Converters.Add(new JsonStringEnumConverter()));
builder.Services.AddHealthChecks().AddDbContextCheck<FlareDbContext>("postgres", HealthStatus.Unhealthy, ["ready"]);
builder.Services.AddSingleton(TimeProvider.System);
builder.Services.AddScoped<ITokenService, TokenService>();
builder.Services.AddScoped<IAuditService, AuditService>();
builder.Services.AddScoped<IActivityService, ActivityService>();
builder.Services.AddScoped<IOverviewService, OverviewService>();
builder.Services.AddSingleton<IHostMetricsService, HostMetricsService>();
builder.Services.AddSingleton<IDockerService, DockerService>();
builder.Services.AddHttpClient<ICoolifyService, CoolifyService>(client =>
{
    client.Timeout = TimeSpan.FromSeconds(15);
    client.DefaultRequestHeaders.UserAgent.ParseAdd("Flare/1.0");
});
builder.Services.AddHostedService<TelemetryPublisherService>();
builder.Services.AddHostedService<InfrastructureMonitorService>();

var app = builder.Build();
if (args.Contains("--migrate", StringComparer.Ordinal))
{
    await using var scope = app.Services.CreateAsyncScope();
    var database = scope.ServiceProvider.GetRequiredService<FlareDbContext>();
    await database.Database.MigrateAsync();
    return;
}

var forwardedHeaders = new ForwardedHeadersOptions
{
    ForwardedHeaders = ForwardedHeaders.XForwardedFor | ForwardedHeaders.XForwardedProto
};
if (builder.Configuration.GetValue("FLARE_TRUST_ALL_FORWARDERS", false))
{
    forwardedHeaders.KnownIPNetworks.Clear();
    forwardedHeaders.KnownProxies.Clear();
}
app.UseForwardedHeaders(forwardedHeaders);
app.UseMiddleware<CorrelationIdMiddleware>();
app.UseExceptionHandler();
app.UseRequestTimeouts();
if (!app.Environment.IsDevelopment())
{
    app.UseHsts();
    app.UseHttpsRedirection();
}
app.UseAuthentication();
app.UseRateLimiter();
app.UseAuthorization();
app.MapControllers();
app.MapHub<TelemetryHub>("/hubs/telemetry");
app.MapHealthChecks("/health/live", new HealthCheckOptions
{
    Predicate = _ => false,
    ResponseWriter = HealthResponseWriter.WriteAsync
});
app.MapHealthChecks("/health/ready", new HealthCheckOptions
{
    Predicate = registration => registration.Tags.Contains("ready"),
    ResponseWriter = HealthResponseWriter.WriteAsync
});

await app.RunAsync();

public partial class Program;
