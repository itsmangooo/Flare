using Flare.Contracts;

namespace Flare.Mobile.Services;

public sealed class SessionStore
{
    private const string ServerUrlKey = "flare.server_url";
    private const string AccessTokenKey = "flare.access_token";
    private const string AccessExpiryKey = "flare.access_expiry";
    private const string RefreshTokenKey = "flare.refresh_token";
    private const string RefreshExpiryKey = "flare.refresh_expiry";
    private const string UserEmailKey = "flare.user_email";
    private readonly IPreferences _preferences = Preferences.Default;
    private readonly ISecureStorage _secureStorage = SecureStorage.Default;

    public string? ServerUrl => _preferences.Get<string?>(ServerUrlKey, null);
    public string? UserEmail => _preferences.Get<string?>(UserEmailKey, null);

    public void SaveServerUrl(string url) => _preferences.Set(ServerUrlKey, url.TrimEnd('/'));

    public async Task SaveTokensAsync(TokenResponse response)
    {
        await _secureStorage.SetAsync(AccessTokenKey, response.AccessToken);
        await _secureStorage.SetAsync(AccessExpiryKey, response.AccessTokenExpiresAt.ToString("O"));
        await _secureStorage.SetAsync(RefreshTokenKey, response.RefreshToken);
        await _secureStorage.SetAsync(RefreshExpiryKey, response.RefreshTokenExpiresAt.ToString("O"));
        _preferences.Set(UserEmailKey, response.User.Email);
    }

    public Task<string?> GetAccessTokenAsync() => _secureStorage.GetAsync(AccessTokenKey);
    public Task<string?> GetRefreshTokenAsync() => _secureStorage.GetAsync(RefreshTokenKey);

    public async Task<DateTimeOffset?> GetAccessExpiryAsync() => Parse(await _secureStorage.GetAsync(AccessExpiryKey));

    public async Task<bool> HasSessionAsync()
    {
        try
        {
            var refreshToken = await GetRefreshTokenAsync();
            var refreshExpiry = Parse(await _secureStorage.GetAsync(RefreshExpiryKey));
            return !string.IsNullOrWhiteSpace(refreshToken) && refreshExpiry > DateTimeOffset.UtcNow;
        }
        catch (Exception)
        {
            // Android can invalidate encrypted preferences after a keystore or backup change.
            // Treat unreadable credentials as signed out instead of crashing at startup.
            ClearAuthentication();
            return false;
        }
    }

    public void ClearServer()
    {
        _preferences.Remove(ServerUrlKey);
        ClearAuthentication();
    }

    public void ClearAuthentication()
    {
        _secureStorage.Remove(AccessTokenKey);
        _secureStorage.Remove(AccessExpiryKey);
        _secureStorage.Remove(RefreshTokenKey);
        _secureStorage.Remove(RefreshExpiryKey);
        _preferences.Remove(UserEmailKey);
    }

    private static DateTimeOffset? Parse(string? value) => DateTimeOffset.TryParse(value, out var parsed) ? parsed : null;
}
