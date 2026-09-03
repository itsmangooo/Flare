namespace Flare.Mobile.Controls;

public sealed class FlareGlassNavigationBar : ContentView
{
    private static readonly NavDestination[] Destinations =
    [
        new("overview", "Overview", "overview.svg"),
        new("containers", "Containers", "containers.svg"),
        new("deployments", "Deployments", "deployments.svg"),
        new("activity", "Activity", "activity.svg")
    ];

    private readonly Grid _items;
    private readonly FlareLiquidGlassView _selection;
    private readonly List<Label> _labels = [];
    private readonly List<Image> _icons = [];
    private int _selectedIndex;
    private bool _navigating;

    public static readonly BindableProperty SelectedRouteProperty = BindableProperty.Create(
        nameof(SelectedRoute), typeof(string), typeof(FlareGlassNavigationBar), "overview", propertyChanged: SelectedRouteChanged);

    public FlareGlassNavigationBar()
    {
        HeightRequest = 68;
        HorizontalOptions = LayoutOptions.Fill;
        MaximumWidthRequest = 440;

        _selection = new FlareLiquidGlassView
        {
            EffectType = LiquidGlassEffectType.Clear,
            CornerRadius = 19,
            BlurRadius = 12,
            RefractionStrength = 9,
            TintColor = Color.FromArgb("#5D261C"),
            TintOpacity = 0.38,
            HighlightStrength = 0.85,
            ChromaticAberration = 1.25,
            ShadowStrength = 0.2,
            HorizontalOptions = LayoutOptions.Start,
            VerticalOptions = LayoutOptions.Fill,
            Margin = new Thickness(0, 4)
        };

        _items = new Grid
        {
            Padding = new Thickness(5),
            ColumnSpacing = 0,
            ColumnDefinitions =
            {
                new ColumnDefinition(GridLength.Star),
                new ColumnDefinition(GridLength.Star),
                new ColumnDefinition(GridLength.Star),
                new ColumnDefinition(GridLength.Star)
            }
        };
        _items.Add(_selection);
        Grid.SetColumnSpan(_selection, 4);

        for (var index = 0; index < Destinations.Length; index++)
        {
            var destination = Destinations[index];
            var icon = new Image { Source = destination.Icon, WidthRequest = 20, HeightRequest = 20, Opacity = 0.7 };
            var label = new Label
            {
                Text = destination.Label,
                FontSize = 9,
                CharacterSpacing = 0.35,
                HorizontalTextAlignment = TextAlignment.Center,
                TextColor = Color.FromArgb("#92979F")
            };
            _icons.Add(icon);
            _labels.Add(label);

            var item = new VerticalStackLayout
            {
                Spacing = 2,
                HorizontalOptions = LayoutOptions.Fill,
                VerticalOptions = LayoutOptions.Center,
                Children = { icon, label }
            };
            var tap = new TapGestureRecognizer { CommandParameter = index };
            tap.Tapped += DestinationTapped;
            item.GestureRecognizers.Add(tap);
            _items.Add(item, index);
        }

        var glass = new FlareLiquidGlassView
        {
            CornerRadius = 24,
            BlurRadius = 22,
            RefractionStrength = 8,
            TintColor = Color.FromArgb("#121418"),
            TintOpacity = 0.56,
            HighlightStrength = 0.68,
            ChromaticAberration = 1.0,
            ShadowStrength = 0.68,
            GlassContent = _items
        };
        Content = glass;
        SizeChanged += (_, _) => UpdateSelection(animated: false);
    }

    public string SelectedRoute { get => (string)GetValue(SelectedRouteProperty); set => SetValue(SelectedRouteProperty, value); }

    private void DestinationTapped(object? sender, TappedEventArgs eventArgs)
    {
        if (_navigating || eventArgs.Parameter is not int index || index < 0 || index >= Destinations.Length) return;
        _ = NavigateAsync(index);
    }

    private async Task NavigateAsync(int index)
    {
        _navigating = true;
        try
        {
            _selectedIndex = index;
            SelectedRoute = Destinations[index].Route;
            UpdateSelection(animated: true);
            await Task.Delay(90);
            if (Shell.Current is not null)
            {
                SelectShellContent(Shell.Current, Destinations[index].Route);
            }
        }
        catch (Exception exception)
        {
            System.Diagnostics.Debug.WriteLine($"Flare glass navigation failed: {exception}");
        }
        finally
        {
            _navigating = false;
        }
    }

    private static void SelectShellContent(Shell shell, string route)
    {
        foreach (var shellItem in shell.Items)
        {
            foreach (var section in shellItem.Items)
            {
                var content = section.Items.FirstOrDefault(item => item.Route == route);
                if (content is null) continue;
                shell.CurrentItem = shellItem;
                shellItem.CurrentItem = section;
                section.CurrentItem = content;
                return;
            }
        }

        throw new InvalidOperationException($"The Flare navigation route '{route}' is not registered.");
    }

    private void UpdateSelection(bool animated)
    {
        _selectedIndex = Math.Max(0, Array.FindIndex(Destinations, item => item.Route == SelectedRoute));
        var available = Math.Max(0, Width - 10);
        var slotWidth = available / Destinations.Length;
        if (slotWidth <= 0) return;
        _selection.WidthRequest = slotWidth;
        var target = _selectedIndex * slotWidth;
        if (animated)
        {
            _selection.AbortAnimation("FlareGlassTabSlide");
            var start = _selection.TranslationX;
            _selection.Animate("FlareGlassTabSlide", value =>
                _selection.TranslationX = start + ((target - start) * value), 16, 190, Easing.CubicOut);
        }
        else
        {
            _selection.TranslationX = target;
        }

        for (var index = 0; index < _labels.Count; index++)
        {
            var selected = index == _selectedIndex;
            _labels[index].TextColor = Color.FromArgb(selected ? "#FFF0EB" : "#92979F");
            _labels[index].FontAttributes = selected ? FontAttributes.Bold : FontAttributes.None;
            _icons[index].Opacity = selected ? 1 : 0.62;
            _icons[index].Scale = selected ? 1.04 : 1;
        }
    }

    private static void SelectedRouteChanged(BindableObject bindable, object oldValue, object newValue) =>
        ((FlareGlassNavigationBar)bindable).UpdateSelection(animated: false);

    private sealed record NavDestination(string Route, string Label, string Icon);
}
