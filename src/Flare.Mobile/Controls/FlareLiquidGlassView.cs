using System.Windows.Input;

namespace Flare.Mobile.Controls;

[ContentProperty(nameof(GlassContent))]
public class FlareLiquidGlassView : ContentView
{
    private readonly LiquidGlassSurface _surface;
    private readonly ContentView _contentHost;

    public static readonly BindableProperty GlassContentProperty = BindableProperty.Create(
        nameof(GlassContent), typeof(View), typeof(FlareLiquidGlassView), propertyChanged: GlassContentChanged);

    public static readonly BindableProperty BlurRadiusProperty = BindableProperty.Create(
        nameof(BlurRadius), typeof(double), typeof(FlareLiquidGlassView), 18d);

    public static readonly BindableProperty RefractionStrengthProperty = BindableProperty.Create(
        nameof(RefractionStrength), typeof(double), typeof(FlareLiquidGlassView), 8d);

    public static readonly BindableProperty TintColorProperty = BindableProperty.Create(
        nameof(TintColor), typeof(Color), typeof(FlareLiquidGlassView), Color.FromArgb("#17191D"));

    public static readonly BindableProperty TintOpacityProperty = BindableProperty.Create(
        nameof(TintOpacity), typeof(double), typeof(FlareLiquidGlassView), 0.42d);

    public static readonly BindableProperty CornerRadiusProperty = BindableProperty.Create(
        nameof(CornerRadius), typeof(double), typeof(FlareLiquidGlassView), 18d);

    public static readonly BindableProperty HighlightStrengthProperty = BindableProperty.Create(
        nameof(HighlightStrength), typeof(double), typeof(FlareLiquidGlassView), 0.65d);

    public static readonly BindableProperty ChromaticAberrationProperty = BindableProperty.Create(
        nameof(ChromaticAberration), typeof(double), typeof(FlareLiquidGlassView), 1.2d);

    public static readonly BindableProperty ShadowStrengthProperty = BindableProperty.Create(
        nameof(ShadowStrength), typeof(double), typeof(FlareLiquidGlassView), 0.45d);

    public static readonly BindableProperty InteractiveProperty = BindableProperty.Create(
        nameof(Interactive), typeof(bool), typeof(FlareLiquidGlassView), false);

    public static readonly BindableProperty EffectTypeProperty = BindableProperty.Create(
        nameof(EffectType), typeof(LiquidGlassEffectType), typeof(FlareLiquidGlassView), LiquidGlassEffectType.Regular);

    public FlareLiquidGlassView()
    {
        _surface = new LiquidGlassSurface { InputTransparent = true };
        _contentHost = new ContentView { BackgroundColor = Colors.Transparent };

        _surface.SetBinding(LiquidGlassSurface.BlurRadiusProperty, new Binding(nameof(BlurRadius), source: this));
        _surface.SetBinding(LiquidGlassSurface.RefractionStrengthProperty, new Binding(nameof(RefractionStrength), source: this));
        _surface.SetBinding(LiquidGlassSurface.TintColorProperty, new Binding(nameof(TintColor), source: this));
        _surface.SetBinding(LiquidGlassSurface.TintOpacityProperty, new Binding(nameof(TintOpacity), source: this));
        _surface.SetBinding(LiquidGlassSurface.CornerRadiusProperty, new Binding(nameof(CornerRadius), source: this));
        _surface.SetBinding(LiquidGlassSurface.HighlightStrengthProperty, new Binding(nameof(HighlightStrength), source: this));
        _surface.SetBinding(LiquidGlassSurface.ChromaticAberrationProperty, new Binding(nameof(ChromaticAberration), source: this));
        _surface.SetBinding(LiquidGlassSurface.ShadowStrengthProperty, new Binding(nameof(ShadowStrength), source: this));
        _surface.SetBinding(LiquidGlassSurface.EffectTypeProperty, new Binding(nameof(EffectType), source: this));

        var layers = new Grid();
        layers.Add(_surface);
        layers.Add(_contentHost);
        base.Content = layers;

        var pointer = new PointerGestureRecognizer();
        pointer.PointerPressed += PointerPressed;
        pointer.PointerMoved += PointerMoved;
        pointer.PointerReleased += PointerReleased;
        pointer.PointerExited += PointerReleased;
        GestureRecognizers.Add(pointer);
    }

    public View? GlassContent { get => (View?)GetValue(GlassContentProperty); set => SetValue(GlassContentProperty, value); }
    public double BlurRadius { get => (double)GetValue(BlurRadiusProperty); set => SetValue(BlurRadiusProperty, value); }
    public double RefractionStrength { get => (double)GetValue(RefractionStrengthProperty); set => SetValue(RefractionStrengthProperty, value); }
    public Color TintColor { get => (Color)GetValue(TintColorProperty); set => SetValue(TintColorProperty, value); }
    public double TintOpacity { get => (double)GetValue(TintOpacityProperty); set => SetValue(TintOpacityProperty, value); }
    public double CornerRadius { get => (double)GetValue(CornerRadiusProperty); set => SetValue(CornerRadiusProperty, value); }
    public double HighlightStrength { get => (double)GetValue(HighlightStrengthProperty); set => SetValue(HighlightStrengthProperty, value); }
    public double ChromaticAberration { get => (double)GetValue(ChromaticAberrationProperty); set => SetValue(ChromaticAberrationProperty, value); }
    public double ShadowStrength { get => (double)GetValue(ShadowStrengthProperty); set => SetValue(ShadowStrengthProperty, value); }
    public bool Interactive { get => (bool)GetValue(InteractiveProperty); set => SetValue(InteractiveProperty, value); }
    public LiquidGlassEffectType EffectType { get => (LiquidGlassEffectType)GetValue(EffectTypeProperty); set => SetValue(EffectTypeProperty, value); }

    protected LiquidGlassSurface GlassSurface => _surface;

    private static void GlassContentChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareLiquidGlassView)bindable)._contentHost.Content = (View?)newValue;

    private void PointerPressed(object? sender, PointerEventArgs eventArgs)
    {
        if (!Interactive || !IsEnabled) return;
        UpdateTouch(eventArgs);
        AnimateInteraction(pressed: true);
    }

    private void PointerMoved(object? sender, PointerEventArgs eventArgs)
    {
        if (Interactive && _surface.PressedProgress > 0) UpdateTouch(eventArgs);
    }

    private void PointerReleased(object? sender, PointerEventArgs eventArgs)
    {
        if (Interactive) AnimateInteraction(pressed: false);
    }

    private void UpdateTouch(PointerEventArgs eventArgs)
    {
        var point = eventArgs.GetPosition(this);
        if (point is null || Width <= 0 || Height <= 0) return;
        _surface.TouchX = Math.Clamp(point.Value.X / Width, 0, 1);
        _surface.TouchY = Math.Clamp(point.Value.Y / Height, 0, 1);
    }

    private void AnimateInteraction(bool pressed)
    {
        this.AbortAnimation("FlareGlassPress");
        var startScale = Scale;
        var startPress = _surface.PressedProgress;
        var endScale = pressed ? 0.985 : 1.0;
        var endPress = pressed ? 1.0 : 0.0;
        this.Animate(
            "FlareGlassPress",
            progress =>
            {
                Scale = startScale + ((endScale - startScale) * progress);
                _surface.PressedProgress = startPress + ((endPress - startPress) * progress);
            },
            16,
            pressed ? 100u : 170u,
            Easing.CubicOut);
    }
}
