using Flare.Mobile.Pages;
using Flare.Mobile.Platforms.Android;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile;

public static class MauiProgram
{
    public static MauiApp CreateMauiApp()
    {
        var builder = MauiApp.CreateBuilder();
        builder.UseMauiApp<App>();
        builder.Logging.AddDebug();
        builder.Logging.AddProvider(new AndroidLogProvider());
        builder.Services.AddSingleton(new HttpClient { Timeout = TimeSpan.FromSeconds(15) });
        builder.Services.AddSingleton<SessionStore>();
        builder.Services.AddSingleton<IServerUrlStore>(services => services.GetRequiredService<SessionStore>());
        builder.Services.AddSingleton<ApiClient>();
        builder.Services.AddSingleton<IActionFeedback, SafeActionFeedback>();
        builder.Services.AddSingleton<AuthService>();
        builder.Services.AddSingleton<ConnectionService>();
        builder.Services.AddSingleton<LiveTelemetryService>();
        builder.Services.AddSingleton<AppNavigator>();
        builder.Services.AddSingleton<StartupPage>();
        builder.Services.AddTransient<ConnectPage>();
        builder.Services.AddTransient<LoginPage>();
        builder.Services.AddTransient<AppShell>();
        builder.Services.AddTransient<OverviewPage>();
        builder.Services.AddTransient<ContainersPage>();
        builder.Services.AddTransient<DeploymentsPage>();
        builder.Services.AddTransient<ActivityPage>();
        builder.Services.AddTransient<SettingsPage>();
        var app = builder.Build();
        AppServices.Initialize(app.Services);
        return app;
    }
}
