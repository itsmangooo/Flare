namespace Flare.Mobile.Controls;

public sealed class MetricBarView : GraphicsView
{
    public static readonly BindableProperty ValueProperty = BindableProperty.Create(
        nameof(Value), typeof(double?), typeof(MetricBarView), null, propertyChanged: Redraw);
    public double? Value { get => (double?)GetValue(ValueProperty); set => SetValue(ValueProperty, value); }

    public MetricBarView()
    {
        HeightRequest = 8;
        Drawable = new MetricBarDrawable(this);
    }

    private static void Redraw(BindableObject bindable, object oldValue, object newValue) =>
        ((MetricBarView)bindable).Invalidate();

    private sealed class MetricBarDrawable(MetricBarView owner) : IDrawable
    {
        public void Draw(ICanvas canvas, RectF dirtyRect)
        {
            const int segments = 20;
            const float gap = 2;
            var width = (dirtyRect.Width - gap * (segments - 1)) / segments;
            var active = owner.Value is { } value ? (int)Math.Ceiling(Math.Clamp(value, 0, 100) / 100 * segments) : 0;
            for (var index = 0; index < segments; index++)
            {
                canvas.FillColor = Color.FromArgb(index < active ? "#F06449" : "#25282C");
                canvas.FillRectangle(index * (width + gap), 0, width, dirtyRect.Height);
            }
        }
    }
}
