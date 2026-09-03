namespace Flare.Mobile.Controls;

public enum LiquidGlassEffectType
{
    Regular,
    Clear
}

public enum LiquidGlassRenderTier
{
    LayeredFallback,
    RenderEffect,
    RuntimeShader
}

public static class LiquidGlassFeaturePolicy
{
    public static LiquidGlassRenderTier Resolve(int androidApiLevel, bool runtimeShaderAvailable = true) =>
        androidApiLevel >= 33 && runtimeShaderAvailable
            ? LiquidGlassRenderTier.RuntimeShader
            : androidApiLevel >= 31
                ? LiquidGlassRenderTier.RenderEffect
                : LiquidGlassRenderTier.LayeredFallback;
}
