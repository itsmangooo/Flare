using System.Collections.ObjectModel;
using System.Windows.Input;
using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class CoolifyResourcesPage : BindablePage
{
    private readonly ApiClient _api;
    public ObservableCollection<CoolifyServerResponse> Servers { get; } = [];
    public ObservableCollection<CoolifyResourceResponse> ServerResources { get; } = [];
    public ObservableCollection<CoolifyApplicationResponse> Applications { get; } = [];
    public ObservableCollection<CoolifyServiceResponse> Services { get; } = [];
    public ICommand RefreshCommand { get; }
    public CoolifyResourcesPage(ApiClient api)
    {
        InitializeComponent(); _api = api; RefreshCommand = new Command(async () => await LoadAsync()); BindingContext = this;
    }
    protected override async void OnAppearing() { base.OnAppearing(); await LoadAsync(); }
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
        finally { IsBusy = false; }
    }
    private async Task ApplicationActionAsync(CoolifyApplicationResponse application, string action, string label)
    {
        if (await DisplayActionSheetAsync($"{label} {application.Name}?", "Cancel", null, label) != label) return;
        await RunAsync($"api/v1/coolify/applications/{application.Uuid}/{action}");
    }
    private async Task RunAsync(string path)
    {
        HapticFeedback.Default.Perform(HapticFeedbackType.LongPress); IsBusy = true; ErrorMessage = null;
        try { _ = await _api.PostAsync<ActionResponse>(path, null, true, CancellationToken.None); await Task.Delay(200); }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }
    private async void StartApplicationClicked(object? sender, EventArgs eventArgs) { if (((Button)sender!).CommandParameter is CoolifyApplicationResponse app) await ApplicationActionAsync(app, "start", "Start"); }
    private async void StopApplicationClicked(object? sender, EventArgs eventArgs) { if (((Button)sender!).CommandParameter is CoolifyApplicationResponse app) await ApplicationActionAsync(app, "stop", "Stop"); }
    private async void RestartApplicationClicked(object? sender, EventArgs eventArgs) { if (((Button)sender!).CommandParameter is CoolifyApplicationResponse app) await ApplicationActionAsync(app, "restart", "Restart"); }
    private async void RedeployApplicationClicked(object? sender, EventArgs eventArgs) { if (((Button)sender!).CommandParameter is CoolifyApplicationResponse app) await ApplicationActionAsync(app, "redeploy", "Redeploy"); }
    private async void RestartServiceClicked(object? sender, EventArgs eventArgs)
    {
        if (((Button)sender!).CommandParameter is not CoolifyServiceResponse service) return;
        if (await DisplayActionSheetAsync($"Restart {service.Name}?", "Cancel", null, "Restart") == "Restart")
            await RunAsync($"api/v1/coolify/services/{service.Uuid}/restart");
    }
    private static void Replace<T>(ObservableCollection<T> target, IEnumerable<T> source) { target.Clear(); foreach (var item in source) target.Add(item); }
}
