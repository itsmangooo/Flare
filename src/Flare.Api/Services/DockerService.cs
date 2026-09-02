using System.Globalization;
using System.Text;
using System.Text.RegularExpressions;
using Docker.DotNet;
using Docker.DotNet.Models;
using Flare.Api.Infrastructure;
using Flare.Contracts;
using ContractContainerState = Flare.Contracts.ContainerState;

namespace Flare.Api.Services;

public sealed partial class DockerService : IDockerService, IDisposable
{
    private readonly DockerClient _client;
    private readonly TimeProvider _timeProvider;
    private readonly ILogger<DockerService> _logger;

    public DockerService(IConfiguration configuration, TimeProvider timeProvider, ILogger<DockerService> logger)
    {
        var host = configuration["DOCKER_HOST"];
        if (string.IsNullOrWhiteSpace(host))
        {
            host = OperatingSystem.IsWindows()
                ? "npipe://./pipe/docker_engine"
                : "unix:///var/run/docker.sock";
        }

        _client = new DockerClientConfiguration(new Uri(host)).CreateClient();
        _timeProvider = timeProvider;
        _logger = logger;
    }

    public async Task<IReadOnlyList<ContainerSummaryResponse>> GetContainersAsync(CancellationToken cancellationToken)
    {
        try
        {
            var containers = await _client.Containers.ListContainersAsync(
                new ContainersListParameters { All = true }, cancellationToken);
            var results = await Task.WhenAll(containers.Select(container => ToSummaryAsync(container, cancellationToken)));
            return results.OrderBy(container => container.Name, StringComparer.OrdinalIgnoreCase).ToArray();
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            throw Unavailable(exception);
        }
    }

    public async Task<ContainerDetailResponse?> GetContainerAsync(string id, CancellationToken cancellationToken)
    {
        ValidateId(id);
        try
        {
            var inspect = await _client.Containers.InspectContainerAsync(id, cancellationToken);
            var stats = inspect.State.Running ? await TryGetStatsAsync(id, cancellationToken) : null;
            var ports = new List<PortBindingResponse>();
            if (inspect.NetworkSettings.Ports is not null)
            {
                foreach (var (key, bindings) in inspect.NetworkSettings.Ports)
                {
                    var pieces = key.Split('/', 2);
                    _ = int.TryParse(pieces[0], CultureInfo.InvariantCulture, out var privatePort);
                    var protocol = pieces.Length == 2 ? pieces[1] : "tcp";
                    if (bindings is null || bindings.Count == 0)
                    {
                        ports.Add(new PortBindingResponse(privatePort, null, protocol, null));
                        continue;
                    }

                    ports.AddRange(bindings.Select(binding => new PortBindingResponse(
                        privatePort,
                        int.TryParse(binding.HostPort, CultureInfo.InvariantCulture, out var publicPort) ? publicPort : null,
                        protocol,
                        binding.HostIP)));
                }
            }

            return new ContainerDetailResponse(
                inspect.ID,
                inspect.ID[..Math.Min(12, inspect.ID.Length)],
                inspect.Name.TrimStart('/'),
                inspect.Config.Image,
                ParseState(inspect.State.Status),
                ParseHealth(inspect.State.Health?.Status),
                inspect.Created,
                ParseTimestamp(inspect.State.StartedAt),
                checked((int)Math.Min(inspect.RestartCount, int.MaxValue)),
                stats?.CpuPercent,
                stats?.MemoryBytes,
                stats?.MemoryLimitBytes,
                stats?.NetworkReceiveBytes,
                stats?.NetworkTransmitBytes,
                ports,
                SanitizeLabels(inspect.Config.Labels));
        }
        catch (DockerContainerNotFoundException)
        {
            return null;
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            throw Unavailable(exception);
        }
    }

    public async Task<LogPageResponse> GetLogsAsync(
        string id, int tail, DateTimeOffset? before, CancellationToken cancellationToken)
    {
        ValidateId(id);
        tail = Math.Clamp(tail, 1, 2_000);
        try
        {
            using var stream = await _client.Containers.GetContainerLogsAsync(id, false,
                new ContainerLogsParameters
                {
                    ShowStdout = true,
                    ShowStderr = true,
                    Timestamps = true,
                    Tail = tail.ToString(CultureInfo.InvariantCulture),
                    Until = before?.ToUniversalTime().ToString("O", CultureInfo.InvariantCulture)
                }, cancellationToken);
            var (output, byteTruncated) = await ReadBoundedLogOutputAsync(stream, cancellationToken);
            var lines = output
                .Split(['\r', '\n'], StringSplitOptions.RemoveEmptyEntries)
                .Select((text, index) => new ParsedLogLine(text, ParseLogTimestamp(text), index))
                .OrderBy(line => line.Timestamp ?? DateTimeOffset.MinValue)
                .ThenBy(line => line.OriginalIndex)
                .TakeLast(tail)
                .ToArray();
            return new LogPageResponse(
                lines.Select(line => line.Text).ToArray(),
                _timeProvider.GetUtcNow(),
                byteTruncated || lines.Length >= tail,
                tail,
                lines.FirstOrDefault()?.Timestamp,
                lines.LastOrDefault()?.Timestamp);
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            throw Unavailable(exception);
        }
    }

    private static async Task<(string Text, bool Truncated)> ReadBoundedLogOutputAsync(
        MultiplexedStream stream, CancellationToken cancellationToken)
    {
        const int maximumReadBytes = 8 * 1024 * 1024;
        const int maximumResponseBytes = 2 * 1024 * 1024;
        var buffer = new byte[16 * 1024];
        using var captured = new MemoryStream(capacity: maximumReadBytes);
        var reachedLimit = false;
        while (captured.Length < maximumReadBytes)
        {
            var available = (int)Math.Min(buffer.Length, maximumReadBytes - captured.Length);
            var result = await stream.ReadOutputAsync(buffer, 0, available, cancellationToken);
            if (result.EOF) break;
            if (result.Count == 0) continue;
            await captured.WriteAsync(buffer.AsMemory(0, result.Count), cancellationToken);
        }
        if (captured.Length >= maximumReadBytes) reachedLimit = true;

        var bytes = captured.GetBuffer().AsSpan(0, checked((int)captured.Length));
        if (bytes.Length > maximumResponseBytes)
        {
            reachedLimit = true;
            bytes = bytes[^maximumResponseBytes..];
            var firstNewline = bytes.IndexOf((byte)'\n');
            if (firstNewline >= 0) bytes = bytes[(firstNewline + 1)..];
        }
        return (Encoding.UTF8.GetString(bytes), reachedLimit);
    }

    public async Task StartAsync(string id, CancellationToken cancellationToken)
    {
        ValidateId(id);
        try
        {
            _ = await _client.Containers.StartContainerAsync(id, new ContainerStartParameters(), cancellationToken);
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            throw Unavailable(exception);
        }
    }

    public async Task StopAsync(string id, CancellationToken cancellationToken)
    {
        ValidateId(id);
        try
        {
            _ = await _client.Containers.StopContainerAsync(id,
                new ContainerStopParameters { WaitBeforeKillSeconds = 20 }, cancellationToken);
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            throw Unavailable(exception);
        }
    }

    public async Task RestartAsync(string id, CancellationToken cancellationToken)
    {
        ValidateId(id);
        try
        {
            await _client.Containers.RestartContainerAsync(id,
                new ContainerRestartParameters { WaitBeforeKillSeconds = 20 }, cancellationToken);
        }
        catch (Exception exception) when (exception is not OperationCanceledException)
        {
            throw Unavailable(exception);
        }
    }

    public void Dispose() => _client.Dispose();

    private async Task<ContainerSummaryResponse> ToSummaryAsync(
        ContainerListResponse container, CancellationToken cancellationToken)
    {
        var state = ParseState(container.State);
        DateTimeOffset? startedAt = null;
        var health = ParseHealthFromStatus(container.Status);
        try
        {
            var inspect = await _client.Containers.InspectContainerAsync(container.ID, cancellationToken);
            startedAt = ParseTimestamp(inspect.State.StartedAt);
            health = ParseHealth(inspect.State.Health?.Status);
        }
        catch (DockerContainerNotFoundException) { }
        var stats = state == ContractContainerState.Running
            ? await TryGetStatsAsync(container.ID, cancellationToken)
            : null;
        return new ContainerSummaryResponse(
            container.ID,
            container.Names.FirstOrDefault()?.TrimStart('/') ?? container.ID[..Math.Min(12, container.ID.Length)],
            container.Image,
            state,
            health,
            startedAt,
            stats?.CpuPercent,
            stats?.MemoryBytes);
    }

    private async Task<Stats?> TryGetStatsAsync(string id, CancellationToken cancellationToken)
    {
        try
        {
            using var timeout = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            timeout.CancelAfter(TimeSpan.FromSeconds(3));
            ContainerStatsResponse? result = null;
            await _client.Containers.GetContainerStatsAsync(id,
                new ContainerStatsParameters { Stream = false },
                new InlineProgress<ContainerStatsResponse>(value => result = value), timeout.Token);
            if (result is null)
            {
                return null;
            }

            var cpuDelta = result.CPUStats.CPUUsage.TotalUsage >= result.PreCPUStats.CPUUsage.TotalUsage
                ? result.CPUStats.CPUUsage.TotalUsage - result.PreCPUStats.CPUUsage.TotalUsage
                : 0;
            var systemDelta = result.CPUStats.SystemUsage >= result.PreCPUStats.SystemUsage
                ? result.CPUStats.SystemUsage - result.PreCPUStats.SystemUsage
                : 0;
            var onlineCpus = result.CPUStats.OnlineCPUs > 0
                ? result.CPUStats.OnlineCPUs
                : (uint)Math.Max(result.CPUStats.CPUUsage.PercpuUsage?.Count ?? 1, 1);
            var cpu = systemDelta > 0 ? cpuDelta / (double)systemDelta * onlineCpus * 100 : (double?)null;
            var cache = result.MemoryStats.Stats?.TryGetValue("inactive_file", out var inactive) == true
                ? inactive
                : result.MemoryStats.Stats?.TryGetValue("cache", out var cached) == true ? cached : 0;
            var memory = result.MemoryStats.Usage >= cache ? result.MemoryStats.Usage - cache : result.MemoryStats.Usage;
            var rx = result.Networks?.Values.Aggregate(0UL, (sum, network) => sum + network.RxBytes) ?? 0;
            var tx = result.Networks?.Values.Aggregate(0UL, (sum, network) => sum + network.TxBytes) ?? 0;
            return new Stats(cpu, checked((long)memory), checked((long)result.MemoryStats.Limit),
                checked((long)rx), checked((long)tx));
        }
        catch (Exception exception) when (exception is not OperationCanceledException || !cancellationToken.IsCancellationRequested)
        {
            LogStatisticsUnavailable(_logger, id, exception);
            return null;
        }
    }

    private InfrastructureUnavailableException Unavailable(Exception exception)
    {
        LogDockerFailure(_logger, exception);
        return new InfrastructureUnavailableException("Docker Engine is unavailable.", exception);
    }

    private static DateTimeOffset? ParseTimestamp(string? value) =>
        DateTimeOffset.TryParse(value, CultureInfo.InvariantCulture, DateTimeStyles.AssumeUniversal, out var parsed)
            ? parsed : null;

    private static DateTimeOffset? ParseLogTimestamp(string line)
    {
        var separator = line.IndexOf(' ');
        var value = separator > 0 ? line[..separator] : line;
        return DateTimeOffset.TryParse(value, CultureInfo.InvariantCulture,
            DateTimeStyles.AssumeUniversal, out var parsed) ? parsed : null;
    }

    private static Dictionary<string, string> SanitizeLabels(IDictionary<string, string>? labels)
    {
        if (labels is null) return new Dictionary<string, string>();
        return labels
            .Where(label => label.Key.Length <= 256 && !SensitiveLabelRegex().IsMatch(label.Key))
            .OrderBy(label => label.Key, StringComparer.OrdinalIgnoreCase)
            .Take(30)
            .ToDictionary(
                label => label.Key,
                label => label.Value.Length <= 512 ? label.Value : label.Value[..512],
                StringComparer.Ordinal);
    }

    private static ContractContainerState ParseState(string? state) => state?.ToLowerInvariant() switch
    {
        "running" => ContractContainerState.Running,
        "exited" or "created" => ContractContainerState.Stopped,
        "paused" => ContractContainerState.Paused,
        "restarting" => ContractContainerState.Restarting,
        "dead" or "removing" => ContractContainerState.Dead,
        _ => ContractContainerState.Unknown
    };

    private static HealthState ParseHealth(string? health) => health?.ToLowerInvariant() switch
    {
        "healthy" => HealthState.Healthy,
        "unhealthy" => HealthState.Unhealthy,
        "starting" => HealthState.Starting,
        null or "" or "none" => HealthState.None,
        _ => HealthState.Unknown
    };

    private static HealthState ParseHealthFromStatus(string? status)
    {
        if (status?.Contains("(healthy)", StringComparison.OrdinalIgnoreCase) == true) return HealthState.Healthy;
        if (status?.Contains("(unhealthy)", StringComparison.OrdinalIgnoreCase) == true) return HealthState.Unhealthy;
        if (status?.Contains("(health: starting)", StringComparison.OrdinalIgnoreCase) == true) return HealthState.Starting;
        return HealthState.None;
    }

    private static void ValidateId(string id)
    {
        if (!ContainerIdRegex().IsMatch(id))
        {
            throw new ArgumentException("Container identifier is invalid.", nameof(id));
        }
    }

    [GeneratedRegex("^[a-fA-F0-9]{12,64}$", RegexOptions.CultureInvariant)]
    private static partial Regex ContainerIdRegex();

    [GeneratedRegex("token|secret|pass(word|wd)?|credential|private[-_. ]?key|api[-_. ]?key",
        RegexOptions.IgnoreCase | RegexOptions.CultureInvariant)]
    private static partial Regex SensitiveLabelRegex();

    private sealed record Stats(double? CpuPercent, long MemoryBytes, long MemoryLimitBytes,
        long NetworkReceiveBytes, long NetworkTransmitBytes);
    private sealed record ParsedLogLine(string Text, DateTimeOffset? Timestamp, int OriginalIndex);

    private sealed class InlineProgress<T>(Action<T> callback) : IProgress<T>
    {
        public void Report(T value) => callback(value);
    }

    [LoggerMessage(LogLevel.Debug, "Container statistics were unavailable for {ContainerId}.")]
    private static partial void LogStatisticsUnavailable(ILogger logger, string containerId, Exception exception);

    [LoggerMessage(LogLevel.Debug, "Docker Engine operation failed; Docker-backed data will be unavailable.")]
    private static partial void LogDockerFailure(ILogger logger, Exception exception);
}
