using Flare.Contracts;

namespace Flare.Mobile.Services;

public sealed class AuthService(ApiClient api, SessionStore sessionStore)
{
    public async Task LoginAsync(string email, string password, CancellationToken cancellationToken)
    {
        var response = await api.PostAsync<TokenResponse>("api/v1/auth/login",
            new LoginRequest(email, password), false, cancellationToken);
        await sessionStore.SaveTokensAsync(response);
    }

    public async Task LogoutAsync(CancellationToken cancellationToken)
    {
        var refresh = await sessionStore.GetRefreshTokenAsync();
        try
        {
            if (!string.IsNullOrWhiteSpace(refresh))
                await api.PostAsync("api/v1/auth/logout", new LogoutRequest(refresh), cancellationToken);
        }
        finally
        {
            sessionStore.ClearAuthentication();
        }
    }
}
