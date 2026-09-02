using System.Collections.ObjectModel;
using System.Windows.Input;
using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class ActivityPage : BindablePage
{
    private readonly ApiClient _api;
    private readonly AppNavigator _navigator;
    private bool _loaded;
    public ObservableCollection<ActivityEventResponse> Events { get; } = [];
    public ICommand RefreshCommand { get; }
    public ActivityPage() : this(AppServices.Get<ApiClient>(), AppServices.Get<AppNavigator>()) { }
    public ActivityPage(ApiClient api, AppNavigator navigator)
    {
        InitializeComponent(); _api = api; _navigator = navigator;
        RefreshCommand = new Command(async () => await LoadAsync()); BindingContext = this;
    }
    protected override async void OnAppearing() { base.OnAppearing(); if (!_loaded) { _loaded = true; await LoadAsync(); } }
    private async Task LoadAsync()
    {
        if (IsBusy) return; IsBusy = true; ErrorMessage = null;
        try { var page = await _api.GetAsync<PagedResponse<ActivityEventResponse>>("api/v1/activity?page=1&pageSize=60", CancellationToken.None); Events.Clear(); foreach (var item in page.Items) Events.Add(item); }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }
    private async void SettingsClicked(object? sender, EventArgs eventArgs) => await _navigator.ShowSettingsAsync();
}
