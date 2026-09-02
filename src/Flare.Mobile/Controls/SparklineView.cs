using Flare.Contracts;

namespace Flare.Mobile.Controls;

public sealed class SparklineView : GraphicsView
{
    public static readonly BindableProperty PointsProperty = BindableProperty.Create(
        nameof(Points), typeof(IReadOnlyList<MetricPoint>), typeof(SparklineView), null, propertyChanged: Redraw);
    public static readonly BindableProperty MetricProperty = BindableProperty.Create(
        nameof(Metric), typeof(string), typeof(SparklineView), "cpu", propertyChanged: Redraw);
    public IReadOnlyList<MetricPoint>? Points { get => (IReadOnlyList<MetricPoint>?)GetValue(PointsProperty); set => SetValue(PointsProperty, value); }
    public string Metric { get => (string)GetValue(MetricProperty); set => SetValue(MetricProperty, value); }

    public SparklineView()
    {
        HeightRequest = 34;
        Drawable = new SparklineDrawable(this);
    }

    private static void Redraw(BindableObject bindable, object oldValue, object newValue) => ((SparklineView)bindable).Invalidate();

    private sealed class SparklineDrawable(SparklineView owner) : IDrawable
    {
        public void Draw(ICanvas canvas, RectF dirtyRect)
        {
            var values = owner.Points?.Select(point => owner.Metric == "memory" ? point.MemoryPercent : point.CpuPercent)
                .Where(value => value is not null).Select(value => value!.Value).ToArray();
            if (values is not { Length: > 1 }) return;
            var path = new PathF();
            for (var index = 0; index < values.Length; index++)
            {
                var x = index * dirtyRect.Width / (values.Length - 1);
                var y = dirtyRect.Height - (float)Math.Clamp(values[index], 0, 100) / 100 * dirtyRect.Height;
                if (index == 0) path.MoveTo(x, y); else path.LineTo(x, y);
            }
            canvas.StrokeColor = Color.FromArgb("#F06449");
            canvas.StrokeSize = 1.5f;
            canvas.DrawPath(path);
        }
    }
}
