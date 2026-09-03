using Microsoft.Extensions.Logging;
using Microsoft.Maui.Devices;

namespace Flare.Mobile.Services;

public interface IActionFeedback
{
    void TryPerformLongPress(string operation);
}

public sealed class SafeActionFeedback : IActionFeedback
{
    private readonly ILogger<SafeActionFeedback> _logger;
    private readonly Action _performLongPress;

    public SafeActionFeedback(ILogger<SafeActionFeedback> logger)
        : this(logger, PerformLongPress)
    {
    }

    internal SafeActionFeedback(ILogger<SafeActionFeedback> logger, Action performLongPress)
    {
        _logger = logger;
        _performLongPress = performLongPress;
    }

    public void TryPerformLongPress(string operation)
    {
        try
        {
            _performLongPress();
        }
        catch (Exception exception)
        {
            // Haptics are optional. Device/ROM support must never decide whether an operation runs.
            MobileLog.HapticFeedbackFailed(_logger, operation, exception);
        }
    }

    private static void PerformLongPress()
    {
        var haptics = HapticFeedback.Default;
        if (haptics.IsSupported)
        {
            haptics.Perform(HapticFeedbackType.LongPress);
        }
    }
}
