using System.Net;
using System.Text;
using Flare.Api.Configuration;
using Flare.Api.Services;
using Microsoft.Extensions.Logging.Abstractions;
using Microsoft.Extensions.Options;

namespace Flare.Api.Tests;

public sealed class CoolifyServiceTests
{
    [Fact]
    public async Task UsesDocumentedServerResourcesEndpointAndBearerAuthentication()
    {
        var handler = new RecordingHandler("[{\"uuid\":\"app-1\",\"name\":\"api\",\"type\":\"application\",\"status\":\"running\"}]");
        var service = Create(handler);
        var resources = await service.GetServerResourcesAsync("server-1", CancellationToken.None);
        Assert.Equal("https://coolify.example/api/v1/servers/server-1/resources", handler.RequestUri?.ToString());
        Assert.Equal("secret-token", handler.Authorization);
        Assert.Single(resources);
        Assert.Equal("api", resources[0].Name);
    }

    [Fact]
    public async Task RedeployUsesOnlyTheDocumentedDeployAction()
    {
        var handler = new RecordingHandler("{\"deployments\":[{\"message\":\"queued\",\"deployment_uuid\":\"dep-1\"}]}");
        var service = Create(handler);
        var result = await service.RedeployApplicationAsync("app-1", CancellationToken.None);
        Assert.Equal(HttpMethod.Post, handler.Method);
        Assert.Equal("https://coolify.example/api/v1/deploy?uuid=app-1&force=false", handler.RequestUri?.ToString());
        Assert.Equal("dep-1", result.OperationId);
    }

    private static CoolifyService Create(HttpMessageHandler handler) => new(
        new HttpClient(handler),
        Options.Create(new CoolifyOptions { BaseUrl = "https://coolify.example", ApiToken = "secret-token" }),
        NullLogger<CoolifyService>.Instance);

    private sealed class RecordingHandler(string response) : HttpMessageHandler
    {
        public Uri? RequestUri { get; private set; }
        public HttpMethod? Method { get; private set; }
        public string? Authorization { get; private set; }

        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            RequestUri = request.RequestUri;
            Method = request.Method;
            Authorization = request.Headers.Authorization?.Parameter;
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(response, Encoding.UTF8, "application/json")
            });
        }
    }
}
