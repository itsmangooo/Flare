using System.Collections.ObjectModel;
using System.Windows.Input;
using Flare.Contracts;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class CoolifyResourcesPage : BindablePage
{
    private readonly ApiClient _api;
    private readonly IActionFeedback _actionFeedback;
    private readonly ILogger<CoolifyResourcesPage> _logger;
    private int _actionInProgress;
    public ObservableCollection<CoolifyServerResponse> Servers { get; } = [];
    public ObservableCollection<CoolifyResourceResponse> ServerResources { get; } = [];
    public ObservableCollection<CoolifyApplicationResponse> Applications { get; } = [];
    public ObservableCollection<CoolifyServiceResponse> Services { get; } = [];
    public ICommand RefreshCommand { get; }
    public CoolifyResourcesPage(ApiClient api)
        : this(api, AppServices.Get<IActionFeedback>(), AppServices.Get<ILogger<CoolifyResourcesPage>>())
    {
    }

    internal CoolifyResourcesPage(
        ApiClient api,
        IActionFeedback actionFeedback,
        ILogger<CoolifyResourcesPage> logger)
    {
        InitializeComponent();
        _api = api;
        _actionFeedback = actionFeedback;
        _logger = logger;
        RefreshCommand = new Command(() => _ = LoadAsync());
        BindingContext = this;
    }
    protected override void OnAppearing()
    {
        base.OnAppearing();
        _ = LoadAsync();
    }
    private async Task LoadAsync()
    {
        if (IsBusy) return; IsBusy = true; ErrorMessage = null;
        try
        {
            var servers = _api.GetAsync<IReadOnlyList<CoolifyServerResponse>>("api/v1/coolify/servers", CancellationToken.None);
            var applications = _api.GetAsync<IReadOnlyList<CoolifyApplicationResponse>>("api/v1/coolify/applications", CancellationToken.None);
            var services = _api.GetAsync<IReadOnlyList<CoolifyServiceResponse>>("api/v1/coolify/services", CancellationToken.None);
            await Task.WhenAll(servers, applications, services);
            var serverItems = await servers;
            var resourceTasks = serverItems.Select(server => _api.GetAsync<IReadOnlyList<CoolifyResourceResponse>>(
                $"api/v1/coolify/servers/{server.Uuid}/resources", CancellationToken.None));
            var resources = (await Task.WhenAll(resourceTasks)).SelectMany(items => items);
            Replace(Servers, serverItems); Replace(ServerResources, resources);
            Replace(Applications, await applications); Replace(Services, await services);
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        catch (Exception exception)
        {
            MobileLog.UiActionFailed(_logger, "coolify.resources.load", exception);
            ErrorMessage = "Flare could not load Coolify resources. Try again.";
        }
        finally { IsBusy = false; }
    }
    private async Task ApplicationActionAsync(CoolifyApplicationResponse application, string action, string label)
    {
        await RunActionAsync(
            $"api/v1/coolify/applications/{application.Uuid}/{action}",
            $"{label} {application.Name}?",
            label,
            $"coolify.application.{action}");
    }

    private async Task RunActionAsync(string path, string prompt, string label, string operation)
    {
        if (IsBusy || Interlocked.CompareExchange(ref _actionInProgress, 1, 0) != 0)
        {
            MobileLog.RepeatedActionIgnored(_logger, operation);
            return;
        }

        try
        {
            await RunUiActionSafelyAsync(_logger, operation, async () =>
            {
                if (await DisplayActionSheetAsync(prompt, "Cancel", null, label) != label) return;

                IsBusy = true;
                ErrorMessage = null;
                try
                {
                    _actionFeedback.TryPerformLongPress(operation);
                    await _api.PostAsync<ActionResponse>(path, null, true, CancellationToken.None);
                    await Task.Delay(200);
                }
                finally
                {
                    IsBusy = false;
                }
            });
        }
        finally
        {
            Interlocked.Exchange(ref _actionInProgress, 0);
        }
    }

    private void StartApplicationClicked(object? sender, EventArgs eventArgs) =>
        QueueApplicationAction(sender, "start", "Start");

    private void StopApplicationClicked(object? sender, EventArgs eventArgs) =>
        QueueApplicationAction(sender, "stop", "Stop");

    private void RestartApplicationClicked(object? sender, EventArgs eventArgs) =>
        QueueApplicationAction(sender, "restart", "Restart");

    private void RedeployApplicationClicked(object? sender, EventArgs eventArgs) =>
        QueueApplicationAction(sender, "redeploy", "Redeploy");

    private void QueueApplicationAction(object? sender, string action, string label)
    {
        if (sender is Button { CommandParameter: CoolifyApplicationResponse application })
        {
            _ = ApplicationActionAsync(application, action, label);
        }
    }

    private void RestartServiceClicked(object? sender, EventArgs eventArgs)
    {
        if (sender is Button { CommandParameter: CoolifyServiceResponse service })
        {
            _ = RunActionAsync(
                $"api/v1/coolify/services/{service.Uuid}/restart",
                $"Restart {service.Name}?",
                "Restart",
                "coolify.service.restart");
        }
    }

    private static void Replace<T>(ObservableCollection<T> target, IEnumerable<T> source) { target.Clear(); foreach (var item in source) target.Add(item); }
}
