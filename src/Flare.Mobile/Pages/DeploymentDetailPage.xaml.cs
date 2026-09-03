using Flare.Contracts;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public partial class DeploymentDetailPage : BindablePage
{
    private readonly ApiClient _api;
    private readonly IActionFeedback _actionFeedback;
    private readonly ILogger<DeploymentDetailPage> _logger;
    private DeploymentResponse _deployment;
    private int _actionInProgress;
    public DeploymentResponse Deployment
    {
        get => _deployment;
        private set { if (Set(ref _deployment, value)) OnPropertyChanged(nameof(CanRedeploy)); }
    }
    public bool CanRedeploy => !string.IsNullOrWhiteSpace(Deployment.ResourceUuid);
    public DeploymentDetailPage(ApiClient api, DeploymentResponse deployment)
        : this(api, deployment, AppServices.Get<IActionFeedback>(), AppServices.Get<ILogger<DeploymentDetailPage>>())
    {
    }

    internal DeploymentDetailPage(
        ApiClient api,
        DeploymentResponse deployment,
        IActionFeedback actionFeedback,
        ILogger<DeploymentDetailPage> logger)
    {
        InitializeComponent();
        _api = api;
        _deployment = deployment;
        _actionFeedback = actionFeedback;
        _logger = logger;
        BindingContext = this;
    }
    protected override void OnAppearing()
    {
        base.OnAppearing();
        _ = LoadDeploymentAsync();
    }

    private async Task LoadDeploymentAsync()
    {
        if (string.IsNullOrWhiteSpace(Deployment.Uuid)) return;
        await RunUiActionSafelyAsync(_logger, "coolify.deployment.load", async () =>
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
        });
    }
    private void RedeployClicked(object? sender, EventArgs eventArgs) => _ = RedeployAsync();

    private async Task RedeployAsync()
    {
        const string operation = "coolify.application.redeploy";
        if (IsBusy || Interlocked.CompareExchange(ref _actionInProgress, 1, 0) != 0)
        {
            MobileLog.RepeatedActionIgnored(_logger, operation);
            return;
        }

        try
        {
            await RunUiActionSafelyAsync(_logger, operation, async () =>
            {
                var selected = await DisplayActionSheetAsync(
                    $"Queue a new deployment for {Deployment.ResourceName}?", "Cancel", null, "Redeploy");
                if (selected != "Redeploy") return;

                IsBusy = true;
                ErrorMessage = null;
                try
                {
                    _actionFeedback.TryPerformLongPress(operation);
                    await _api.PostAsync<ActionResponse>(
                        $"api/v1/coolify/applications/{Deployment.ResourceUuid}/redeploy",
                        null,
                        true,
                        CancellationToken.None);
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
}
