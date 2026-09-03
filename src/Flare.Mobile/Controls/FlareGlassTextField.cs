namespace Flare.Mobile.Controls;

public sealed class FlareGlassTextField : FlareLiquidGlassView
{
    private readonly Entry _entry;

    public static readonly BindableProperty TextProperty = BindableProperty.Create(
        nameof(Text), typeof(string), typeof(FlareGlassTextField), string.Empty,
        BindingMode.TwoWay, propertyChanged: TextChangedFromOutside);

    public static readonly BindableProperty PlaceholderProperty = BindableProperty.Create(
        nameof(Placeholder), typeof(string), typeof(FlareGlassTextField), string.Empty, propertyChanged: PlaceholderChanged);

    public static readonly BindableProperty IsPasswordProperty = BindableProperty.Create(
        nameof(IsPassword), typeof(bool), typeof(FlareGlassTextField), false, propertyChanged: IsPasswordChanged);

    public static readonly BindableProperty KeyboardProperty = BindableProperty.Create(
        nameof(Keyboard), typeof(Keyboard), typeof(FlareGlassTextField), Keyboard.Default, propertyChanged: KeyboardChanged);

    public static readonly BindableProperty ReturnTypeProperty = BindableProperty.Create(
        nameof(ReturnType), typeof(ReturnType), typeof(FlareGlassTextField), ReturnType.Default, propertyChanged: ReturnTypeChanged);

    public event EventHandler? Completed;
    public event EventHandler<TextChangedEventArgs>? TextChanged;

    public FlareGlassTextField()
    {
        HeightRequest = 52;
        CornerRadius = 13;
        BlurRadius = 18;
        RefractionStrength = 5;
        TintOpacity = 0.38;
        HighlightStrength = 0.5;
        ChromaticAberration = 0.65;
        Interactive = true;

        _entry = new Entry
        {
            BackgroundColor = Colors.Transparent,
            TextColor = Color.FromArgb("#F3F4F6"),
            PlaceholderColor = Color.FromArgb("#656A72"),
            FontSize = 14,
            Margin = new Thickness(12, 0)
        };
        _entry.TextChanged += EntryTextChanged;
        _entry.Completed += (_, _) => Completed?.Invoke(this, EventArgs.Empty);
        GlassContent = _entry;

        var tap = new TapGestureRecognizer();
        tap.Tapped += (_, _) => _entry.Focus();
        GestureRecognizers.Add(tap);
    }

    public string Text { get => (string)GetValue(TextProperty); set => SetValue(TextProperty, value); }
    public string Placeholder { get => (string)GetValue(PlaceholderProperty); set => SetValue(PlaceholderProperty, value); }
    public bool IsPassword { get => (bool)GetValue(IsPasswordProperty); set => SetValue(IsPasswordProperty, value); }
    public Keyboard Keyboard { get => (Keyboard)GetValue(KeyboardProperty); set => SetValue(KeyboardProperty, value); }
    public ReturnType ReturnType { get => (ReturnType)GetValue(ReturnTypeProperty); set => SetValue(ReturnTypeProperty, value); }

    public new bool Focus() => _entry.Focus();

    private void EntryTextChanged(object? sender, TextChangedEventArgs eventArgs)
    {
        if (Text != eventArgs.NewTextValue) SetValue(TextProperty, eventArgs.NewTextValue ?? string.Empty);
        TextChanged?.Invoke(this, eventArgs);
    }

    private static void TextChangedFromOutside(BindableObject bindable, object oldValue, object newValue)
    {
        var field = (FlareGlassTextField)bindable;
        var text = (string?)newValue ?? string.Empty;
        if (field._entry.Text != text) field._entry.Text = text;
    }

    private static void PlaceholderChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassTextField)bindable)._entry.Placeholder = (string?)newValue;

    private static void IsPasswordChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassTextField)bindable)._entry.IsPassword = (bool)newValue;

    private static void KeyboardChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassTextField)bindable)._entry.Keyboard = (Keyboard)newValue;

    private static void ReturnTypeChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassTextField)bindable)._entry.ReturnType = (ReturnType)newValue;
}
