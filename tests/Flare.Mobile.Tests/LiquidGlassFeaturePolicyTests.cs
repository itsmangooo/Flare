using Flare.Mobile.Controls;

namespace Flare.Mobile.Tests;

public sealed class LiquidGlassFeaturePolicyTests
{
    [Theory]
    [InlineData(26)]
    [InlineData(30)]
    public void OlderAndroidUsesLayeredFallback(int apiLevel) =>
        Assert.Equal(LiquidGlassRenderTier.LayeredFallback, LiquidGlassFeaturePolicy.Resolve(apiLevel));

    [Theory]
    [InlineData(31)]
    [InlineData(32)]
    public void AndroidTwelveUsesRenderEffect(int apiLevel) =>
        Assert.Equal(LiquidGlassRenderTier.RenderEffect, LiquidGlassFeaturePolicy.Resolve(apiLevel));

    [Theory]
    [InlineData(33)]
    [InlineData(36)]
    public void ModernAndroidUsesRuntimeShader(int apiLevel) =>
        Assert.Equal(LiquidGlassRenderTier.RuntimeShader, LiquidGlassFeaturePolicy.Resolve(apiLevel));

    [Fact]
    public void RuntimeShaderFailureFallsBackToRenderEffect() =>
        Assert.Equal(LiquidGlassRenderTier.RenderEffect, LiquidGlassFeaturePolicy.Resolve(35, runtimeShaderAvailable: false));
}
