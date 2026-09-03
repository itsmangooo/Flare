namespace Flare.Mobile.Controls;

public sealed class FlareGlassBottomSheet : ContentView
{
    internal FlareGlassBottomSheet(string title, string message, string action, bool destructive)
    {
        var actionButton = new FlareGlassButton
        {
            Text = action,
            IsPrimary = !destructive,
            IsDestructive = destructive
        };
        var cancelButton = new FlareGlassButton { Text = "Cancel" };

        var content = new VerticalStackLayout
        {
            Spacing = 12,
            Children =
            {
                new BoxView
                {
                    WidthRequest = 34,
                    HeightRequest = 3,
                    CornerRadius = 1.5f,
                    Color = Color.FromArgb("#596068"),
                    HorizontalOptions = LayoutOptions.Center,
                    Margin = new Thickness(0, 0, 0, 4)
                },
                new Label { Text = title, FontSize = 19, FontAttributes = FontAttributes.Bold },
                new Label { Text = message, FontSize = 13, TextColor = Color.FromArgb("#A8ADB5") },
                actionButton,
                cancelButton
            }
        };

        Content = new FlareLiquidGlassView
        {
            GlassContent = content,
            Padding = new Thickness(20, 14, 20, 22),
            CornerRadius = 26,
            BlurRadius = 26,
            RefractionStrength = 10,
            TintColor = Color.FromArgb("#15171B"),
            TintOpacity = 0.67,
            HighlightStrength = 0.8,
            ChromaticAberration = 1.25,
            ShadowStrength = 0.8
        };

        actionButton.Clicked += (_, _) => Completed?.Invoke(this, true);
        cancelButton.Clicked += (_, _) => Completed?.Invoke(this, false);
    }

    internal event EventHandler<bool>? Completed;

    public static Task<bool> ShowConfirmationAsync(
        Page owner,
        string title,
        string message,
        string action,
        bool destructive = false) =>
        GlassDialogHost.ShowAsync(owner, new FlareGlassBottomSheet(title, message, action, destructive), bottomAligned: true);
}

public sealed class FlareGlassModal : ContentView
{
    internal FlareGlassModal(string title, string message, string action, bool destructive)
    {
        var actionButton = new FlareGlassButton
        {
            Text = action,
            IsPrimary = !destructive,
            IsDestructive = destructive
        };
        var cancelButton = new FlareGlassButton { Text = "Cancel" };

        Content = new FlareLiquidGlassView
        {
            CornerRadius = 22,
            BlurRadius = 24,
            RefractionStrength = 9,
            TintOpacity = 0.64,
            HighlightStrength = 0.76,
            ChromaticAberration = 1.1,
            ShadowStrength = 0.78,
            Padding = new Thickness(20),
            GlassContent = new VerticalStackLayout
            {
                Spacing = 12,
                Children =
                {
                    new Label { Text = title, FontSize = 19, FontAttributes = FontAttributes.Bold },
                    new Label { Text = message, FontSize = 13, TextColor = Color.FromArgb("#A8ADB5") },
                    actionButton,
                    cancelButton
                }
            }
        };

        actionButton.Clicked += (_, _) => Completed?.Invoke(this, true);
        cancelButton.Clicked += (_, _) => Completed?.Invoke(this, false);
    }

    internal event EventHandler<bool>? Completed;

    public static Task<bool> ShowConfirmationAsync(
        Page owner,
        string title,
        string message,
        string action,
        bool destructive = false) =>
        GlassDialogHost.ShowAsync(owner, new FlareGlassModal(title, message, action, destructive), bottomAligned: false);
}

internal static class GlassDialogHost
{
    public static async Task<bool> ShowAsync(Page owner, ContentView dialog, bool bottomAligned)
    {
        var completion = new TaskCompletionSource<bool>(TaskCreationOptions.RunContinuationsAsynchronously);
        var overlay = new GlassDialogPage(dialog, bottomAligned, completion);

        switch (dialog)
        {
            case FlareGlassBottomSheet sheet:
                sheet.Completed += overlay.Complete;
                break;
            case FlareGlassModal modal:
                modal.Completed += overlay.Complete;
                break;
        }

        await owner.Navigation.PushModalAsync(overlay, false);
        return await completion.Task;
    }

    private sealed class GlassDialogPage : ContentPage
    {
        private readonly TaskCompletionSource<bool> _completion;
        private int _completed;

        public GlassDialogPage(ContentView dialog, bool bottomAligned, TaskCompletionSource<bool> completion)
        {
            _completion = completion;
            BackgroundColor = Colors.Transparent;
            Shell.SetPresentationMode(this, PresentationMode.NotAnimated);

            var scrim = new BoxView { Color = Color.FromArgb("#78000000") };
            var dismiss = new TapGestureRecognizer();
            dismiss.Tapped += (_, _) => _ = FinishAsync(false);
            scrim.GestureRecognizers.Add(dismiss);

            var grid = new Grid();
            grid.Add(scrim);
            dialog.Margin = bottomAligned ? new Thickness(12, 0, 12, 12) : new Thickness(24);
            dialog.MaximumWidthRequest = 520;
            dialog.HorizontalOptions = LayoutOptions.Fill;
            dialog.VerticalOptions = bottomAligned ? LayoutOptions.End : LayoutOptions.Center;
            grid.Add(dialog);
            Content = grid;
        }

        public void Complete(object? sender, bool result) => _ = FinishAsync(result);

        protected override bool OnBackButtonPressed()
        {
            _ = FinishAsync(false);
            return true;
        }

        protected override void OnDisappearing()
        {
            base.OnDisappearing();
            if (Interlocked.Exchange(ref _completed, 1) == 0) _completion.TrySetResult(false);
        }

        private async Task FinishAsync(bool result)
        {
            if (Interlocked.Exchange(ref _completed, 1) != 0) return;
            _completion.TrySetResult(result);
            try
            {
                await Navigation.PopModalAsync(false);
            }
            catch (Exception exception)
            {
                System.Diagnostics.Debug.WriteLine($"Flare glass dialog dismissal failed: {exception}");
            }
        }
    }
}
