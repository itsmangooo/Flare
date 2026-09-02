using Flare.Api.Configuration;
using Flare.Api.Services;
using Microsoft.Extensions.Logging.Abstractions;
using Microsoft.Extensions.Options;

namespace Flare.Api.Tests;

public sealed class HostMetricsServiceTests : IDisposable
{
    private readonly string _directory = Path.Combine(Path.GetTempPath(), $"flare-tests-{Guid.NewGuid():N}");

    [Fact]
    public async Task ReadsHostProcMountAndDoesNotInventFirstSampleRates()
    {
        Directory.CreateDirectory(Path.Combine(_directory, "net"));
        await File.WriteAllTextAsync(Path.Combine(_directory, "stat"), "cpu  100 0 50 850 0 0 0 0 0 0\n");
        await File.WriteAllTextAsync(Path.Combine(_directory, "loadavg"), "2.14 1.50 1.00 1/100 1\n");
        await File.WriteAllTextAsync(Path.Combine(_directory, "meminfo"), "MemTotal: 16384 kB\nMemAvailable: 4096 kB\n");
        await File.WriteAllTextAsync(Path.Combine(_directory, "uptime"), "3600.00 0.00\n");
        await File.WriteAllTextAsync(Path.Combine(_directory, "net", "dev"),
            "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\neth0: 1000 1 0 0 0 0 0 0 2000 1 0 0 0 0 0 0\n");
        var service = new HostMetricsService(Options.Create(new HostMetricsOptions
        {
            HostName = "lab-01",
            ProcPath = _directory,
            RootFileSystemPath = _directory
        }), TimeProvider.System, NullLogger<HostMetricsService>.Instance);
        var metrics = await service.GetAsync(CancellationToken.None);
        Assert.Equal("lab-01", metrics.HostName);
        Assert.Equal(2.14, metrics.LoadAverage);
        Assert.Equal(16_777_216, metrics.MemoryTotalBytes);
        Assert.Equal(12_582_912, metrics.MemoryUsedBytes);
        Assert.Equal(TimeSpan.FromHours(1), metrics.Uptime);
        Assert.Null(metrics.CpuPercent);
        Assert.Null(metrics.NetworkReceiveBytesPerSecond);
    }

    public void Dispose()
    {
        if (Directory.Exists(_directory)) Directory.Delete(_directory, recursive: true);
    }
}
