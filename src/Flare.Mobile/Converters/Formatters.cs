using System.Globalization;
using Flare.Contracts;

namespace Flare.Mobile.Converters;

public sealed class PercentConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture) =>
        value is double number ? $"{number:0.#}%" : "—";
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}

public sealed class BytesConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        if (value is not long && value is not double) return "—";
        var bytes = value is long integer ? integer : (double)value;
        var units = new[] { "B", "KB", "MB", "GB", "TB" };
        var size = bytes;
        var unit = 0;
        while (size >= 1024 && unit < units.Length - 1) { size /= 1024; unit++; }
        return $"{size:0.#} {units[unit]}{(parameter?.ToString() == "rate" ? "/s" : string.Empty)}";
    }
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}

public sealed class DurationConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        if (value is not TimeSpan duration) return "—";
        if (duration.TotalDays >= 1) return $"{(int)duration.TotalDays}d {duration.Hours:00}h";
        if (duration.TotalHours >= 1) return $"{(int)duration.TotalHours}h {duration.Minutes:00}m";
        return $"{Math.Max(0, (int)duration.TotalMinutes)}m";
    }
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}

public sealed class TimestampConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        if (value is not DateTimeOffset timestamp) return string.Empty;
        var elapsed = DateTimeOffset.UtcNow - timestamp;
        if (elapsed.TotalMinutes < 1) return "now";
        if (elapsed.TotalHours < 1) return $"{(int)elapsed.TotalMinutes}m";
        if (elapsed.TotalDays < 1) return $"{(int)elapsed.TotalHours}h";
        return timestamp.LocalDateTime.ToString("MMM d", culture);
    }
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}

public sealed class ExactTimestampConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture) =>
        value is DateTimeOffset timestamp ? timestamp.LocalDateTime.ToString("yyyy-MM-dd HH:mm:ss", culture) : "—";
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}

public sealed class UptimeSinceConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        if (value is not DateTimeOffset started) return string.Empty;
        var duration = DateTimeOffset.UtcNow - started;
        if (duration.TotalDays >= 1) return $"· {(int)duration.TotalDays}d";
        if (duration.TotalHours >= 1) return $"· {(int)duration.TotalHours}h";
        return $"· {Math.Max(0, (int)duration.TotalMinutes)}m";
    }
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}

public sealed class StatusColorConverter : IValueConverter
{
    public object Convert(object? value, Type targetType, object? parameter, CultureInfo culture)
    {
        var status = value?.ToString()?.ToLowerInvariant();
        if (value is OperationResult result) return Color.FromArgb(result == OperationResult.Succeeded ? "#57A773" : "#D46161");
        if (value is ContainerState state) return Color.FromArgb(state == ContainerState.Running ? "#57A773" : "#747980");
        if (value is DataFreshness freshness) return Color.FromArgb(freshness == DataFreshness.Live ? "#57A773" : freshness == DataFreshness.Reconnecting ? "#D6A84B" : "#747980");
        if (status?.Contains("unhealthy", StringComparison.Ordinal) == true
            || status?.Contains("failed", StringComparison.Ordinal) == true
            || status?.Contains("error", StringComparison.Ordinal) == true
            || status?.Contains("unreachable", StringComparison.Ordinal) == true)
            return Color.FromArgb("#D46161");
        return Color.FromArgb(status?.Contains("running", StringComparison.Ordinal) == true
            || status?.Contains("finished", StringComparison.Ordinal) == true
            || status?.Contains("success", StringComparison.Ordinal) == true
            || status?.Contains("healthy", StringComparison.Ordinal) == true
            || status?.Contains("connected", StringComparison.Ordinal) == true ? "#57A773" : "#D6A84B");
    }
    public object ConvertBack(object? value, Type targetType, object? parameter, CultureInfo culture) => throw new NotSupportedException();
}
