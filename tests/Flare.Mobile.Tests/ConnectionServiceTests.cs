using System.Net;
using System.Security.Authentication;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging.Abstractions;

namespace Flare.Mobile.Tests;

public sealed class ConnectionServiceTests
{
    [Fact]
    public async Task SuccessfulProbeStoresAndReturnsNormalizedOrigin()
    {
        var store = new TestServerUrlStore();
        Uri? requestedUri = null;
        using var httpClient = CreateClient((request, _) =>
        {
            requestedUri = request.RequestUri;
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK));
        });
        var service = CreateService(httpClient, store);

        var result = await service.ConnectAsync(" https://flare.example.com/some/path/ ", CancellationToken.None);

        Assert.True(result.Success);
        Assert.Equal(ConnectionOutcome.Connected, result.Outcome);
        Assert.Equal("Connected", result.StateLabel);
        Assert.Equal("https://flare.example.com", store.ServerUrl);
        Assert.Equal(new Uri("https://flare.example.com/health/ready"), requestedUri);
    }

    [Fact]
    public async Task NonSuccessHealthResponseIsReportedAsNotReady()
    {
        var store = new TestServerUrlStore();
        using var httpClient = CreateClient((_, _) =>
            Task.FromResult(new HttpResponseMessage(HttpStatusCode.ServiceUnavailable)));

        var result = await CreateService(httpClient, store)
            .ConnectAsync("https://flare.example.com", CancellationToken.None);

        Assert.False(result.Success);
        Assert.Equal(ConnectionOutcome.NotReady, result.Outcome);
        Assert.Equal("Not ready", result.StateLabel);
        Assert.Null(store.ServerUrl);
    }

    [Fact]
    public async Task AndroidRedirectLoopWebExceptionDoesNotEscape()
    {
        using var httpClient = CreateClient((_, _) => Task.FromException<HttpResponseMessage>(
            new WebException(
                "Maximum automatic redirections exceeded (allowed 50, redirected 50 times)",
                WebExceptionStatus.ProtocolError)));

        var result = await CreateService(httpClient, new TestServerUrlStore())
            .ConnectAsync("https://flare.example.com", CancellationToken.None);

        Assert.False(result.Success);
        Assert.Equal(ConnectionOutcome.InvalidResponse, result.Outcome);
        Assert.Equal("Invalid response", result.StateLabel);
        Assert.Contains("redirect", result.Message, StringComparison.OrdinalIgnoreCase);
    }

    [Fact]
    public async Task TlsFailureIsReportedSeparately()
    {
        using var httpClient = CreateClient((_, _) => Task.FromException<HttpResponseMessage>(
            new HttpRequestException("TLS failed", new AuthenticationException("Untrusted certificate"))));

        var result = await CreateService(httpClient, new TestServerUrlStore())
            .ConnectAsync("https://flare.example.com", CancellationToken.None);

        Assert.Equal(ConnectionOutcome.TlsError, result.Outcome);
        Assert.Equal("TLS error", result.StateLabel);
    }

    [Fact]
    public async Task TimeoutIsReportedSeparately()
    {
        using var httpClient = CreateClient((_, _) =>
            Task.FromException<HttpResponseMessage>(new TaskCanceledException("Timed out")));

        var result = await CreateService(httpClient, new TestServerUrlStore())
            .ConnectAsync("https://flare.example.com", CancellationToken.None);

        Assert.Equal(ConnectionOutcome.Timeout, result.Outcome);
        Assert.Equal("Timeout", result.StateLabel);
    }

    [Fact]
    public async Task CallerCancellationStillPropagates()
    {
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        using var httpClient = CreateClient((_, token) =>
            Task.FromCanceled<HttpResponseMessage>(token));

        await Assert.ThrowsAnyAsync<OperationCanceledException>(() =>
            CreateService(httpClient, new TestServerUrlStore())
                .ConnectAsync("https://flare.example.com", cancellation.Token));
    }

    [Fact]
    public async Task StorageFailureIsHandledAfterSuccessfulProbe()
    {
        using var httpClient = CreateClient((_, _) =>
            Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)));

        var result = await CreateService(httpClient, new TestServerUrlStore { ThrowOnSave = true })
            .ConnectAsync("https://flare.example.com", CancellationToken.None);

        Assert.Equal(ConnectionOutcome.StorageError, result.Outcome);
        Assert.Equal("Connection error", result.StateLabel);
    }

    private static ConnectionService CreateService(HttpClient client, IServerUrlStore store) =>
        new(client, store, NullLogger<ConnectionService>.Instance);

    private static HttpClient CreateClient(
        Func<HttpRequestMessage, CancellationToken, Task<HttpResponseMessage>> responseFactory) =>
        new(new StubHttpMessageHandler(responseFactory)) { Timeout = TimeSpan.FromSeconds(2) };

    private sealed class TestServerUrlStore : IServerUrlStore
    {
        public string? ServerUrl { get; private set; }
        public bool ThrowOnSave { get; init; }

        public void SaveServerUrl(string url)
        {
            if (ThrowOnSave)
            {
                throw new InvalidOperationException("Storage unavailable");
            }

            ServerUrl = url;
        }
    }

    private sealed class StubHttpMessageHandler(
        Func<HttpRequestMessage, CancellationToken, Task<HttpResponseMessage>> responseFactory)
        : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request,
            CancellationToken cancellationToken) => responseFactory(request, cancellationToken);
    }
}
