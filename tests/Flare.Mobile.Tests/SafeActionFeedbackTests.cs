using Flare.Mobile.Services;
using Microsoft.Extensions.Logging.Abstractions;

namespace Flare.Mobile.Tests;

public sealed class SafeActionFeedbackTests
{
    [Fact]
    public void HapticPlatformFailureNeverEscapes()
    {
        var feedback = new SafeActionFeedback(
            NullLogger<SafeActionFeedback>.Instance,
            () => throw new InvalidOperationException("Haptics unavailable"));

        var exception = Record.Exception(() => feedback.TryPerformLongPress("container.restart"));

        Assert.Null(exception);
    }

    [Fact]
    public void SupportedHapticIsPerformedOnce()
    {
        var calls = 0;
        var feedback = new SafeActionFeedback(
            NullLogger<SafeActionFeedback>.Instance,
            () => calls++);

        feedback.TryPerformLongPress("container.start");

        Assert.Equal(1, calls);
    }
}
