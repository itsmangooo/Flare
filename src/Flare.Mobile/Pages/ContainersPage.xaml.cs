using System.Collections.ObjectModel;
using System.Windows.Input;
using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class ContainersPage : BindablePage, IDisposable
{
    private readonly ApiClient _api;
    private readonly AppNavigator _navigator;
    private IReadOnlyList<ContainerSummaryResponse> _all = [];
    private bool _loaded;
    private CancellationTokenSource? _updates;
    public ObservableCollection<ContainerSummaryResponse> Containers { get; } = [];
    public ICommand RefreshCommand { get; }

    public ContainersPage() : this(AppServices.Get<ApiClient>(), AppServices.Get<AppNavigator>()) { }
    public ContainersPage(ApiClient api, AppNavigator navigator)
    {
        InitializeComponent();
        _api = api;
        _navigator = navigator;
        RefreshCommand = new Command(async () => await LoadAsync());
        BindingContext = this;
    }

    protected override async void OnAppearing()
    {
        base.OnAppearing();
        if (!_loaded) { _loaded = true; await LoadAsync(); }
        _updates?.Cancel();
        _updates = new CancellationTokenSource();
        _ = UpdateLoopAsync(_updates.Token);
    }

    protected override void OnDisappearing() { _updates?.Cancel(); base.OnDisappearing(); }

    private async Task LoadAsync()
    {
        if (IsBusy) return;
        IsBusy = true;
        ErrorMessage = null;
        try { _all = await _api.GetAsync<IReadOnlyList<ContainerSummaryResponse>>("api/v1/containers", CancellationToken.None); ApplyFilter(); }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }

    private void ApplyFilter()
    {
        var query = _all.AsEnumerable();
        var search = SearchText;
        if (!string.IsNullOrWhiteSpace(search)) query = query.Where(item => item.Name.Contains(search, StringComparison.OrdinalIgnoreCase) || item.Image.Contains(search, StringComparison.OrdinalIgnoreCase));
        query = FilterPicker.SelectedItem?.ToString() switch
        {
            "Running" => query.Where(item => item.State == ContainerState.Running),
            "Stopped" => query.Where(item => item.State != ContainerState.Running),
            "Unhealthy" => query.Where(item => item.Health == HealthState.Unhealthy),
            _ => query
        };
        Sync(query.ToArray());
    }

    private void Sync(ContainerSummaryResponse[] desired)
    {
        for (var index = 0; index < desired.Length; index++)
        {
            var currentIndex = -1;
            for (var searchIndex = index; searchIndex < Containers.Count; searchIndex++)
            {
                if (Containers[searchIndex].Id == desired[index].Id) { currentIndex = searchIndex; break; }
            }
            if (currentIndex < 0) Containers.Insert(index, desired[index]);
            else
            {
                if (currentIndex != index) Containers.Move(currentIndex, index);
                Containers[index] = desired[index];
            }
        }
        while (Containers.Count > desired.Length) Containers.RemoveAt(Containers.Count - 1);
    }

    private async Task UpdateLoopAsync(CancellationToken cancellationToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(5));
        try
        {
            while (await timer.WaitForNextTickAsync(cancellationToken))
            {
                if (IsBusy) continue;
                try
                {
                    _all = await _api.GetAsync<IReadOnlyList<ContainerSummaryResponse>>("api/v1/containers", cancellationToken);
                    ErrorMessage = null;
                    ApplyFilter();
                }
                catch (FlareApiException exception) { ErrorMessage = exception.Message; }
            }
        }
        catch (OperationCanceledException) { }
    }

    private string SearchText { get; set; } = string.Empty;
    private void SearchChanged(object? sender, TextChangedEventArgs eventArgs) { SearchText = eventArgs.NewTextValue ?? string.Empty; ApplyFilter(); }
    private void FilterChanged(object? sender, EventArgs eventArgs) => ApplyFilter();
    private async void ContainerSelected(object? sender, SelectionChangedEventArgs eventArgs)
    {
        if (eventArgs.CurrentSelection.Count == 0 || eventArgs.CurrentSelection[0] is not ContainerSummaryResponse container) return;
        ((CollectionView)sender!).SelectedItem = null;
        await Navigation.PushAsync(new ContainerDetailPage(_api, container.Id));
    }
    private async void SettingsClicked(object? sender, EventArgs eventArgs) => await _navigator.ShowSettingsAsync();
    public void Dispose()
    {
        _updates?.Cancel();
        _updates?.Dispose();
        _updates = null;
        GC.SuppressFinalize(this);
    }
}
