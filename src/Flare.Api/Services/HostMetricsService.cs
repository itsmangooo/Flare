using System.Globalization;
using Flare.Api.Configuration;
using Flare.Contracts;
using Microsoft.Extensions.Options;

namespace Flare.Api.Services;

public sealed partial class HostMetricsService(
    IOptions<HostMetricsOptions> options,
    TimeProvider timeProvider,
    ILogger<HostMetricsService> logger) : IHostMetricsService
{
    private readonly object _sampleLock = new();
    private CpuSample? _previousCpu;
    private NetworkSample? _previousNetwork;

    public async Task<HostMetricsResponse> GetAsync(CancellationToken cancellationToken)
    {
        var now = timeProvider.GetUtcNow();
        double? cpu = null;
        double? load = null;
        long? memoryUsed = null;
        long? memoryTotal = null;
        long? diskUsed = null;
        long? diskTotal = null;
        double? networkReceive = null;
        double? networkTransmit = null;
        TimeSpan? uptime = null;

        try
        {
            var procPath = options.Value.ProcPath;
            var stat = await ReadFirstLineAsync(Path.Combine(procPath, "stat"), cancellationToken);
            if (TryParseCpu(stat, out var cpuSample))
            {
                lock (_sampleLock)
                {
                    if (_previousCpu is { } previous)
                    {
                        var totalDelta = cpuSample.Total >= previous.Total ? cpuSample.Total - previous.Total : 0;
                        var idleDelta = cpuSample.Idle >= previous.Idle ? cpuSample.Idle - previous.Idle : 0;
                        var activeDelta = totalDelta >= idleDelta ? totalDelta - idleDelta : 0;
                        cpu = totalDelta > 0 ? Math.Clamp(activeDelta * 100d / totalDelta, 0, 100) : null;
                    }
                    _previousCpu = cpuSample;
                }
            }

            var loadLine = await ReadFirstLineAsync(Path.Combine(procPath, "loadavg"), cancellationToken);
            if (loadLine is not null && double.TryParse(loadLine.Split(' ', StringSplitOptions.RemoveEmptyEntries)[0],
                    NumberStyles.Float, CultureInfo.InvariantCulture, out var parsedLoad))
            {
                load = parsedLoad;
            }

            var memInfo = await File.ReadAllLinesAsync(Path.Combine(procPath, "meminfo"), cancellationToken);
            var memory = ParseMemInfo(memInfo);
            memoryTotal = memory.TotalBytes;
            memoryUsed = memory.TotalBytes is { } total && memory.AvailableBytes is { } available
                ? Math.Max(0, total - available) : null;

            var uptimeLine = await ReadFirstLineAsync(Path.Combine(procPath, "uptime"), cancellationToken);
            if (uptimeLine is not null && double.TryParse(uptimeLine.Split(' ', StringSplitOptions.RemoveEmptyEntries)[0],
                    NumberStyles.Float, CultureInfo.InvariantCulture, out var uptimeSeconds))
            {
                uptime = TimeSpan.FromSeconds(uptimeSeconds);
            }

            var networkLines = await File.ReadAllLinesAsync(Path.Combine(procPath, "net", "dev"), cancellationToken);
            var network = ParseNetwork(networkLines, now);
            lock (_sampleLock)
            {
                if (_previousNetwork is { } previous)
                {
                    var seconds = (network.Timestamp - previous.Timestamp).TotalSeconds;
                    if (seconds > 0)
                    {
                        networkReceive = Math.Max(0, network.ReceiveBytes - previous.ReceiveBytes) / seconds;
                        networkTransmit = Math.Max(0, network.TransmitBytes - previous.TransmitBytes) / seconds;
                    }
                }
                _previousNetwork = network;
            }
        }
        catch (Exception exception) when (exception is IOException or UnauthorizedAccessException or FormatException
            or OverflowException)
        {
            LogProcUnavailable(logger, exception);
        }

        try
        {
            var rootPath = Path.GetFullPath(options.Value.RootFileSystemPath);
            if (!Directory.Exists(rootPath)) throw new IOException("Configured host filesystem mount does not exist.");
            var drive = DriveInfo.GetDrives()
                .Where(candidate => rootPath.StartsWith(
                    Path.GetFullPath(candidate.RootDirectory.FullName), StringComparison.Ordinal))
                .OrderByDescending(candidate => candidate.RootDirectory.FullName.Length)
                .FirstOrDefault();
            if (drive is { IsReady: true })
            {
                diskTotal = drive.TotalSize;
                diskUsed = drive.TotalSize - drive.AvailableFreeSpace;
            }
        }
        catch (Exception exception) when (exception is IOException or UnauthorizedAccessException or ArgumentException)
        {
            LogDiskUnavailable(logger, exception);
        }

        return new HostMetricsResponse(options.Value.HostName, now, cpu, load, memoryUsed, memoryTotal,
            diskUsed, diskTotal, networkReceive, networkTransmit, uptime);
    }

    private static async Task<string?> ReadFirstLineAsync(string path, CancellationToken cancellationToken)
    {
        using var reader = new StreamReader(path);
        return await reader.ReadLineAsync(cancellationToken);
    }

    private static bool TryParseCpu(string? line, out CpuSample sample)
    {
        sample = default;
        if (line is null || !line.StartsWith("cpu ", StringComparison.Ordinal)) return false;
        var values = line.Split(' ', StringSplitOptions.RemoveEmptyEntries).Skip(1)
            .Select(value => ulong.Parse(value, CultureInfo.InvariantCulture)).ToArray();
        if (values.Length < 5) return false;
        var idle = values[3] + values[4];
        var total = values.Aggregate(0UL, (sum, value) => sum + value);
        sample = new CpuSample(total, idle);
        return true;
    }

    private static (long? TotalBytes, long? AvailableBytes) ParseMemInfo(IEnumerable<string> lines)
    {
        long? total = null;
        long? available = null;
        foreach (var line in lines)
        {
            var parts = line.Split([':', ' '], StringSplitOptions.RemoveEmptyEntries);
            if (parts.Length < 2 || !long.TryParse(parts[1], CultureInfo.InvariantCulture, out var kibibytes)) continue;
            if (parts[0] == "MemTotal") total = checked(kibibytes * 1024);
            if (parts[0] == "MemAvailable") available = checked(kibibytes * 1024);
        }
        return (total, available);
    }

    private static NetworkSample ParseNetwork(IEnumerable<string> lines, DateTimeOffset now)
    {
        long receive = 0;
        long transmit = 0;
        foreach (var line in lines.Skip(2))
        {
            var parts = line.Split([':', ' '], StringSplitOptions.RemoveEmptyEntries);
            if (parts.Length < 10 || parts[0] == "lo") continue;
            receive += long.Parse(parts[1], CultureInfo.InvariantCulture);
            transmit += long.Parse(parts[9], CultureInfo.InvariantCulture);
        }
        return new NetworkSample(receive, transmit, now);
    }

    private readonly record struct CpuSample(ulong Total, ulong Idle);
    private readonly record struct NetworkSample(long ReceiveBytes, long TransmitBytes, DateTimeOffset Timestamp);

    [LoggerMessage(LogLevel.Debug, "Host /proc telemetry is unavailable; nullable metrics will be returned.")]
    private static partial void LogProcUnavailable(ILogger logger, Exception exception);

    [LoggerMessage(LogLevel.Debug, "Host disk telemetry is unavailable; nullable metrics will be returned.")]
    private static partial void LogDiskUnavailable(ILogger logger, Exception exception);
}
