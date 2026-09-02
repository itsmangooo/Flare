namespace Flare.Mobile.Services;

public static class ServerUrlValidator
{
    public static bool TryNormalize(string? input, bool allowHttpLoopback, out string normalized, out string error)
    {
        normalized = string.Empty;
        error = string.Empty;
        if (!Uri.TryCreate(input?.Trim(), UriKind.Absolute, out var uri) || string.IsNullOrWhiteSpace(uri.Host))
        {
            error = "Enter a complete Flare server URL.";
            return false;
        }
        var loopbackHttp = allowHttpLoopback && uri.Scheme == Uri.UriSchemeHttp && uri.IsLoopback;
        if (uri.Scheme != Uri.UriSchemeHttps && !loopbackHttp)
        {
            error = "Flare requires an HTTPS server.";
            return false;
        }
        if (!string.IsNullOrEmpty(uri.UserInfo) || !string.IsNullOrEmpty(uri.Query) || !string.IsNullOrEmpty(uri.Fragment))
        {
            error = "Use the server base URL without credentials, a query, or a fragment.";
            return false;
        }
        normalized = uri.GetLeftPart(UriPartial.Authority).TrimEnd('/');
        return true;
    }
}
