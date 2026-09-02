using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Platforms.Android;

public sealed class AndroidLogProvider : ILoggerProvider
{
    public ILogger CreateLogger(string categoryName) => new AndroidLogger(categoryName);

    public void Dispose()
    {
        GC.SuppressFinalize(this);
    }

    private sealed class AndroidLogger(string categoryName) : ILogger
    {
        public IDisposable BeginScope<TState>(TState state) where TState : notnull => EmptyScope.Instance;

        public bool IsEnabled(LogLevel logLevel) => logLevel >= LogLevel.Information;

        public void Log<TState>(
            LogLevel logLevel,
            EventId eventId,
            TState state,
            Exception? exception,
            Func<TState, Exception?, string> formatter)
        {
            if (!IsEnabled(logLevel))
            {
                return;
            }

            var category = categoryName[(categoryName.LastIndexOf('.') + 1)..];
            var message = $"[{category}:{eventId.Id}] {formatter(state, exception)}";
            if (exception is not null)
            {
                message = $"{message}{Environment.NewLine}{exception}";
            }

            global::Android.Util.Log.WriteLine(ToPriority(logLevel), "Flare", message);
        }

        private static global::Android.Util.LogPriority ToPriority(LogLevel level) => level switch
        {
            LogLevel.Trace or LogLevel.Debug => global::Android.Util.LogPriority.Debug,
            LogLevel.Information => global::Android.Util.LogPriority.Info,
            LogLevel.Warning => global::Android.Util.LogPriority.Warn,
            LogLevel.Error => global::Android.Util.LogPriority.Error,
            LogLevel.Critical => global::Android.Util.LogPriority.Assert,
            _ => global::Android.Util.LogPriority.Verbose
        };
    }

    private sealed class EmptyScope : IDisposable
    {
        public static EmptyScope Instance { get; } = new();
        public void Dispose()
        {
        }
    }
}
