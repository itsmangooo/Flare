using System.Windows.Input;

namespace Flare.Mobile.Controls;

public sealed class FlareGlassButton : FlareLiquidGlassView
{
    private readonly Label _label;

    public static readonly BindableProperty TextProperty = BindableProperty.Create(
        nameof(Text), typeof(string), typeof(FlareGlassButton), string.Empty, propertyChanged: TextChanged);

    public static readonly BindableProperty TextColorProperty = BindableProperty.Create(
        nameof(TextColor), typeof(Color), typeof(FlareGlassButton), Color.FromArgb("#F3F4F6"), propertyChanged: TextColorChanged);

    public static readonly BindableProperty FontSizeProperty = BindableProperty.Create(
        nameof(FontSize), typeof(double), typeof(FlareGlassButton), 13d, propertyChanged: FontSizeChanged);

    public static readonly BindableProperty FontAttributesProperty = BindableProperty.Create(
        nameof(FontAttributes), typeof(FontAttributes), typeof(FlareGlassButton), FontAttributes.Bold, propertyChanged: FontAttributesChanged);

    public static readonly BindableProperty CommandProperty = BindableProperty.Create(
        nameof(Command), typeof(ICommand), typeof(FlareGlassButton));

    public static readonly BindableProperty CommandParameterProperty = BindableProperty.Create(
        nameof(CommandParameter), typeof(object), typeof(FlareGlassButton));

    public static readonly BindableProperty IsPrimaryProperty = BindableProperty.Create(
        nameof(IsPrimary), typeof(bool), typeof(FlareGlassButton), false, propertyChanged: AppearanceChanged);

    public static readonly BindableProperty IsDestructiveProperty = BindableProperty.Create(
        nameof(IsDestructive), typeof(bool), typeof(FlareGlassButton), false, propertyChanged: AppearanceChanged);

    public event EventHandler? Clicked;

    public FlareGlassButton()
    {
        HeightRequest = 44;
        CornerRadius = 12;
        BlurRadius = 14;
        RefractionStrength = 5;
        TintOpacity = 0.46;
        HighlightStrength = 0.58;
        ChromaticAberration = 0.8;
        Interactive = true;

        _label = new Label
        {
            HorizontalTextAlignment = TextAlignment.Center,
            VerticalTextAlignment = TextAlignment.Center,
            FontAttributes = FontAttributes.Bold,
            FontSize = 13,
            TextColor = Color.FromArgb("#F3F4F6"),
            Margin = new Thickness(10, 0)
        };
        GlassContent = _label;

        var tap = new TapGestureRecognizer();
        tap.Tapped += Tapped;
        GestureRecognizers.Add(tap);
        ApplyAppearance();
    }

    public string Text { get => (string)GetValue(TextProperty); set => SetValue(TextProperty, value); }
    public Color TextColor { get => (Color)GetValue(TextColorProperty); set => SetValue(TextColorProperty, value); }
    public double FontSize { get => (double)GetValue(FontSizeProperty); set => SetValue(FontSizeProperty, value); }
    public FontAttributes FontAttributes { get => (FontAttributes)GetValue(FontAttributesProperty); set => SetValue(FontAttributesProperty, value); }
    public ICommand? Command { get => (ICommand?)GetValue(CommandProperty); set => SetValue(CommandProperty, value); }
    public object? CommandParameter { get => GetValue(CommandParameterProperty); set => SetValue(CommandParameterProperty, value); }
    public bool IsPrimary { get => (bool)GetValue(IsPrimaryProperty); set => SetValue(IsPrimaryProperty, value); }
    public bool IsDestructive { get => (bool)GetValue(IsDestructiveProperty); set => SetValue(IsDestructiveProperty, value); }

    private void Tapped(object? sender, TappedEventArgs eventArgs)
    {
        if (!IsEnabled) return;
        if (Command?.CanExecute(CommandParameter) == true) Command.Execute(CommandParameter);
        Clicked?.Invoke(this, EventArgs.Empty);
    }

    private void ApplyAppearance()
    {
        if (IsDestructive)
        {
            TintColor = Color.FromArgb("#5A2020");
            TintOpacity = 0.62;
            _label.TextColor = Color.FromArgb("#FFD5D1");
            return;
        }

        if (IsPrimary)
        {
            TintColor = Color.FromArgb("#8D301F");
            TintOpacity = 0.72;
            HighlightStrength = 0.78;
            _label.TextColor = Color.FromArgb("#FFF4F0");
            return;
        }

        TintColor = Color.FromArgb("#17191D");
        TintOpacity = 0.48;
        HighlightStrength = 0.56;
        _label.TextColor = TextColor;
    }

    private static void TextChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassButton)bindable)._label.Text = (string?)newValue;

    private static void TextColorChanged(BindableObject bindable, object oldValue, object newValue)
    {
        var button = (FlareGlassButton)bindable;
        if (!button.IsPrimary && !button.IsDestructive) button._label.TextColor = (Color)newValue;
    }

    private static void FontSizeChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassButton)bindable)._label.FontSize = (double)newValue;

    private static void FontAttributesChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassButton)bindable)._label.FontAttributes = (FontAttributes)newValue;

    private static void AppearanceChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassButton)bindable).ApplyAppearance();
}
