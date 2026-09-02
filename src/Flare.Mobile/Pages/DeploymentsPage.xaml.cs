using System.Collections.ObjectModel;
using System.Windows.Input;
using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class DeploymentsPage : BindablePage
{
    private readonly ApiClient _api;
    private readonly AppNavigator _navigator;
    private bool _loaded;
    public ObservableCollection<DeploymentResponse> Deployments { get; } = [];
    public ICommand RefreshCommand { get; }

    public DeploymentsPage() : this(AppServices.Get<ApiClient>(), AppServices.Get<AppNavigator>()) { }
    public DeploymentsPage(ApiClient api, AppNavigator navigator)
    {
        InitializeComponent(); _api = api; _navigator = navigator;
        RefreshCommand = new Command(async () => await LoadAsync()); BindingContext = this;
    }
    protected override async void OnAppearing() { base.OnAppearing(); if (!_loaded) { _loaded = true; await LoadAsync(); } }
    private async Task LoadAsync()
    {
        if (IsBusy) return; IsBusy = true; ErrorMessage = null;
        try
        {
            var page = await _api.GetAsync<PagedResponse<DeploymentResponse>>("api/v1/coolify/deployments?page=1&pageSize=50", CancellationToken.None);
            Deployments.Clear(); foreach (var item in page.Items) Deployments.Add(item);
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }
    private async void DeploymentSelected(object? sender, SelectionChangedEventArgs eventArgs)
    {
        if (eventArgs.CurrentSelection.Count == 0 || eventArgs.CurrentSelection[0] is not DeploymentResponse deployment) return;
        ((CollectionView)sender!).SelectedItem = null;
        await Navigation.PushAsync(new DeploymentDetailPage(_api, deployment));
    }
    private async void SettingsClicked(object? sender, EventArgs eventArgs) => await _navigator.ShowSettingsAsync();
    private async void ResourcesClicked(object? sender, EventArgs eventArgs) => await Navigation.PushAsync(new CoolifyResourcesPage(_api));
}
