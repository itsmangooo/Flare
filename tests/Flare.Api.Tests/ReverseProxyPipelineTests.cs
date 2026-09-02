using System.Net;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Hosting;

namespace Flare.Api.Tests;

public sealed class ReverseProxyPipelineTests : IClassFixture<ReverseProxyPipelineTests.FlareApiFactory>
{
    private readonly FlareApiFactory _factory;

    public ReverseProxyPipelineTests(FlareApiFactory factory) => _factory = factory;

    [Fact]
    public async Task ProductionInternalHttpRequestIsServedWithoutHttpsRedirect()
    {
        using var client = _factory.CreateClient(new WebApplicationFactoryClientOptions
        {
            AllowAutoRedirect = false,
            BaseAddress = new Uri("http://flare.example.com")
        });

        using var response = await client.GetAsync("/health/live");

        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        Assert.Null(response.Headers.Location);
    }

    [Fact]
    public async Task ForwardedHttpsRequestReceivesHstsWithoutRedirect()
    {
        using var client = _factory.CreateClient(new WebApplicationFactoryClientOptions
        {
            AllowAutoRedirect = false,
            BaseAddress = new Uri("http://flare.example.com")
        });
        using var request = new HttpRequestMessage(HttpMethod.Get, "/health/live");
        request.Headers.TryAddWithoutValidation("X-Forwarded-Proto", "https");

        using var response = await client.SendAsync(request);

        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
        Assert.Null(response.Headers.Location);
        Assert.True(response.Headers.Contains("Strict-Transport-Security"));
    }

    public sealed class FlareApiFactory : WebApplicationFactory<Program>
    {
        protected override void ConfigureWebHost(IWebHostBuilder builder)
        {
            builder.UseEnvironment(Environments.Production);
            builder.UseSetting("ConnectionStrings:Postgres",
                "Host=127.0.0.1;Port=1;Database=flare;Username=flare;Password=test;Timeout=1");
            builder.UseSetting("FLARE_JWT_SIGNING_KEY", "test-only-signing-key-at-least-32-bytes");
            builder.UseSetting("FLARE_TRUST_ALL_FORWARDERS", "true");
            builder.ConfigureServices(services =>
            {
                var hostedServices = services
                    .Where(descriptor => descriptor.ServiceType == typeof(IHostedService))
                    .ToArray();
                foreach (var hostedService in hostedServices)
                {
                    services.Remove(hostedService);
                }
            });
        }
    }
}
