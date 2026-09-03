using Microsoft.Maui.Controls.Shapes;

namespace Flare.Mobile.Controls;

public sealed class FlareStatusPill : FlareLiquidGlassView
{
    private readonly Ellipse _dot;
    private readonly Label _label;

    public static readonly BindableProperty TextProperty = BindableProperty.Create(
        nameof(Text), typeof(string), typeof(FlareStatusPill), string.Empty, propertyChanged: TextChanged);

    public static readonly BindableProperty StatusColorProperty = BindableProperty.Create(
        nameof(StatusColor), typeof(Color), typeof(FlareStatusPill), Color.FromArgb("#747980"), propertyChanged: StatusColorChanged);

    public FlareStatusPill()
    {
        HeightRequest = 32;
        CornerRadius = 16;
        BlurRadius = 12;
        RefractionStrength = 4;
        TintOpacity = 0.34;
        HighlightStrength = 0.48;
        ChromaticAberration = 0.5;
        Padding = new Thickness(10, 0);

        _dot = new Ellipse { WidthRequest = 7, HeightRequest = 7, Fill = new SolidColorBrush(Color.FromArgb("#747980")) };
        _label = new Label
        {
            FontSize = 10,
            FontAttributes = FontAttributes.Bold,
            CharacterSpacing = 1.1,
            VerticalTextAlignment = TextAlignment.Center
        };
        GlassContent = new HorizontalStackLayout
        {
            Spacing = 7,
            VerticalOptions = LayoutOptions.Center,
            Children = { _dot, _label }
        };
    }

    public string Text { get => (string)GetValue(TextProperty); set => SetValue(TextProperty, value); }
    public Color StatusColor { get => (Color)GetValue(StatusColorProperty); set => SetValue(StatusColorProperty, value); }

    private static void TextChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareStatusPill)bindable)._label.Text = (string?)newValue;

    private static void StatusColorChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareStatusPill)bindable)._dot.Fill = new SolidColorBrush((Color)newValue);
}
