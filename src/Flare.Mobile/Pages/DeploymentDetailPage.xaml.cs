using Flare.Contracts;
using Flare.Mobile.Services;

namespace Flare.Mobile.Pages;

public partial class DeploymentDetailPage : BindablePage
{
    private readonly ApiClient _api;
    private DeploymentResponse _deployment;
    public DeploymentResponse Deployment
    {
        get => _deployment;
        private set { if (Set(ref _deployment, value)) OnPropertyChanged(nameof(CanRedeploy)); }
    }
    public bool CanRedeploy => !string.IsNullOrWhiteSpace(Deployment.ResourceUuid);
    public DeploymentDetailPage(ApiClient api, DeploymentResponse deployment)
    {
        InitializeComponent(); _api = api; _deployment = deployment; BindingContext = this;
    }
    protected override async void OnAppearing()
    {
        base.OnAppearing();
        if (string.IsNullOrWhiteSpace(Deployment.Uuid)) return;
        try
        {
            var resourceUuid = Deployment.ResourceUuid;
            var resourceName = Deployment.ResourceName;
            var detail = await _api.GetAsync<DeploymentResponse>($"api/v1/coolify/deployments/{Deployment.Uuid}", CancellationToken.None);
            Deployment = detail with
            {
                ResourceUuid = string.IsNullOrWhiteSpace(detail.ResourceUuid) ? resourceUuid : detail.ResourceUuid,
                ResourceName = detail.ResourceName == "Application" && !string.IsNullOrWhiteSpace(resourceName)
                    ? resourceName
                    : detail.ResourceName
            };
        }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
    }
    private async void RedeployClicked(object? sender, EventArgs eventArgs)
    {
        var selected = await DisplayActionSheetAsync(
            $"Queue a new deployment for {Deployment.ResourceName}?", "Cancel", null, "Redeploy");
        if (selected != "Redeploy") return;
        HapticFeedback.Default.Perform(HapticFeedbackType.LongPress); IsBusy = true; ErrorMessage = null;
        try { _ = await _api.PostAsync<ActionResponse>($"api/v1/coolify/applications/{Deployment.ResourceUuid}/redeploy", null, true, CancellationToken.None); }
        catch (FlareApiException exception) { ErrorMessage = exception.Message; }
        finally { IsBusy = false; }
    }
}
