using Flare.Mobile.Pages;
using Microsoft.Extensions.Logging;
using Microsoft.Maui.ApplicationModel;

namespace Flare.Mobile.Services;

public sealed class AppNavigator(
    IServiceProvider provider,
    LiveTelemetryService telemetry,
    SessionStore sessionStore,
    ILogger<AppNavigator> logger) : IDisposable
{
    private readonly SemaphoreSlim _navigationGate = new(1, 1);
    private Window? _window;

    public void Attach(Window window) => _window = window;

    public Task ShowConnectAsync(CancellationToken cancellationToken = default) =>
        NavigateRootAsync<ConnectPage>("ConnectPage", resetTelemetry: false, startTelemetry: false, cancellationToken);

    public Task ShowLoginAsync(CancellationToken cancellationToken = default) =>
        NavigateRootAsync<LoginPage>("LoginPage", resetTelemetry: true, startTelemetry: false, cancellationToken);

    public Task ShowShellAsync(CancellationToken cancellationToken = default) =>
        NavigateRootAsync<AppShell>("AppShell", resetTelemetry: false, startTelemetry: true, cancellationToken);

    public Task ShowSettingsAsync() => MainThread.InvokeOnMainThreadAsync(async () =>
    {
        MobileLog.ResolvingPage(logger, "SettingsPage");
        await Shell.Current.Navigation.PushAsync(provider.GetRequiredService<SettingsPage>());
        MobileLog.NavigationCompleted(logger, "SettingsPage");
    });

    public async Task ChangeServerAsync(CancellationToken cancellationToken = default)
    {
        sessionStore.ClearServer();
        await NavigateRootAsync<ConnectPage>("ConnectPage", resetTelemetry: true, startTelemetry: false,
            cancellationToken);
    }

    private async Task NavigateRootAsync<TPage>(
        string destination,
        bool resetTelemetry,
        bool startTelemetry,
        CancellationToken cancellationToken)
        where TPage : Page
    {
        await _navigationGate.WaitAsync(cancellationToken);
        try
        {
            if (resetTelemetry)
            {
                try
                {
                    await telemetry.ResetAsync(cancellationToken);
                }
                catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
                {
                    throw;
                }
                catch (Exception exception)
                {
                    // Telemetry cleanup is best-effort and must not prevent authentication navigation.
                    MobileLog.TelemetryResetFailed(logger, destination, exception);
                }
            }

            await MainThread.InvokeOnMainThreadAsync(() =>
            {
                MobileLog.ResolvingPage(logger, destination);
                var page = provider.GetRequiredService<TPage>();
                if (_window is null)
                {
                    throw new InvalidOperationException("The application window is not attached to the navigator.");
                }

                _window.Page = page;
                MobileLog.NavigationCompleted(logger, destination);
            });

            if (startTelemetry)
            {
                try
                {
                    await telemetry.StartAsync(cancellationToken);
                }
                catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
                {
                    throw;
                }
                catch (Exception exception)
                {
                    // AppShell exposes offline/stale states; telemetry startup is allowed to fail independently.
                    MobileLog.TelemetryStartFailed(logger, destination, exception);
                }
            }
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            MobileLog.NavigationCanceled(logger, destination);
            throw;
        }
        catch (Exception exception)
        {
            MobileLog.NavigationFailed(logger, destination, exception);
            throw;
        }
        finally
        {
            _navigationGate.Release();
        }
    }

    public void Dispose()
    {
        _navigationGate.Dispose();
    }
}
