namespace Flare.Mobile.Controls;

public sealed class LiquidGlassSurface : View
{
    public static readonly BindableProperty BlurRadiusProperty = BindableProperty.Create(
        nameof(BlurRadius), typeof(double), typeof(LiquidGlassSurface), 18d);

    public static readonly BindableProperty RefractionStrengthProperty = BindableProperty.Create(
        nameof(RefractionStrength), typeof(double), typeof(LiquidGlassSurface), 8d);

    public static readonly BindableProperty TintColorProperty = BindableProperty.Create(
        nameof(TintColor), typeof(Color), typeof(LiquidGlassSurface), Color.FromArgb("#17191D"));

    public static readonly BindableProperty TintOpacityProperty = BindableProperty.Create(
        nameof(TintOpacity), typeof(double), typeof(LiquidGlassSurface), 0.42d);

    public static readonly BindableProperty CornerRadiusProperty = BindableProperty.Create(
        nameof(CornerRadius), typeof(double), typeof(LiquidGlassSurface), 18d);

    public static readonly BindableProperty HighlightStrengthProperty = BindableProperty.Create(
        nameof(HighlightStrength), typeof(double), typeof(LiquidGlassSurface), 0.65d);

    public static readonly BindableProperty ChromaticAberrationProperty = BindableProperty.Create(
        nameof(ChromaticAberration), typeof(double), typeof(LiquidGlassSurface), 1.2d);

    public static readonly BindableProperty ShadowStrengthProperty = BindableProperty.Create(
        nameof(ShadowStrength), typeof(double), typeof(LiquidGlassSurface), 0.45d);

    public static readonly BindableProperty EffectTypeProperty = BindableProperty.Create(
        nameof(EffectType), typeof(LiquidGlassEffectType), typeof(LiquidGlassSurface), LiquidGlassEffectType.Regular);

    public static readonly BindableProperty TouchXProperty = BindableProperty.Create(
        nameof(TouchX), typeof(double), typeof(LiquidGlassSurface), 0.25d);

    public static readonly BindableProperty TouchYProperty = BindableProperty.Create(
        nameof(TouchY), typeof(double), typeof(LiquidGlassSurface), 0.15d);

    public static readonly BindableProperty PressedProgressProperty = BindableProperty.Create(
        nameof(PressedProgress), typeof(double), typeof(LiquidGlassSurface), 0d);

    public double BlurRadius { get => (double)GetValue(BlurRadiusProperty); set => SetValue(BlurRadiusProperty, value); }
    public double RefractionStrength { get => (double)GetValue(RefractionStrengthProperty); set => SetValue(RefractionStrengthProperty, value); }
    public Color TintColor { get => (Color)GetValue(TintColorProperty); set => SetValue(TintColorProperty, value); }
    public double TintOpacity { get => (double)GetValue(TintOpacityProperty); set => SetValue(TintOpacityProperty, value); }
    public double CornerRadius { get => (double)GetValue(CornerRadiusProperty); set => SetValue(CornerRadiusProperty, value); }
    public double HighlightStrength { get => (double)GetValue(HighlightStrengthProperty); set => SetValue(HighlightStrengthProperty, value); }
    public double ChromaticAberration { get => (double)GetValue(ChromaticAberrationProperty); set => SetValue(ChromaticAberrationProperty, value); }
    public double ShadowStrength { get => (double)GetValue(ShadowStrengthProperty); set => SetValue(ShadowStrengthProperty, value); }
    public LiquidGlassEffectType EffectType { get => (LiquidGlassEffectType)GetValue(EffectTypeProperty); set => SetValue(EffectTypeProperty, value); }
    public double TouchX { get => (double)GetValue(TouchXProperty); set => SetValue(TouchXProperty, value); }
    public double TouchY { get => (double)GetValue(TouchYProperty); set => SetValue(TouchYProperty, value); }
    public double PressedProgress { get => (double)GetValue(PressedProgressProperty); set => SetValue(PressedProgressProperty, value); }
}
