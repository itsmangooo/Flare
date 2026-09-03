namespace Flare.Mobile.Controls;

public sealed class FlareGlassCard : FlareLiquidGlassView
{
    public FlareGlassCard()
    {
        Padding = new Thickness(16);
        CornerRadius = 14;
        BlurRadius = 20;
        RefractionStrength = 7;
        TintOpacity = 0.44;
        HighlightStrength = 0.55;
        ShadowStrength = 0.5;
    }
}
