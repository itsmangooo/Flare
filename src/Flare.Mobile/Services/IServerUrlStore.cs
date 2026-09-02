namespace Flare.Mobile.Services;

public interface IServerUrlStore
{
    string? ServerUrl { get; }
    void SaveServerUrl(string url);
}
