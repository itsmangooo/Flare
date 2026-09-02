namespace Flare.Mobile.Services;

public static class AppServices
{
    public static IServiceProvider Current { get; private set; } = null!;
    public static void Initialize(IServiceProvider provider) => Current = provider;
    public static T Get<T>() where T : notnull => Current.GetRequiredService<T>();
}
