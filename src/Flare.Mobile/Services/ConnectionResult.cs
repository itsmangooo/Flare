namespace Flare.Mobile.Services;

public enum ConnectionOutcome
{
    Connected,
    InvalidUrl,
    NotReady,
    Unreachable,
    TlsError,
    Timeout,
    InvalidResponse,
    StorageError
}

public sealed record ConnectionResult(ConnectionOutcome Outcome, string Message)
{
    public bool Success => Outcome == ConnectionOutcome.Connected;

    public string StateLabel => Outcome switch
    {
        ConnectionOutcome.Connected => "Connected",
        ConnectionOutcome.InvalidUrl => "Invalid URL",
        ConnectionOutcome.NotReady => "Not ready",
        ConnectionOutcome.Unreachable => "Unreachable",
        ConnectionOutcome.TlsError => "TLS error",
        ConnectionOutcome.Timeout => "Timeout",
        ConnectionOutcome.InvalidResponse => "Invalid response",
        ConnectionOutcome.StorageError => "Connection error",
        _ => "Connection error"
    };
}
