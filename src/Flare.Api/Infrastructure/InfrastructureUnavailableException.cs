namespace Flare.Api.Infrastructure;

public sealed class InfrastructureUnavailableException(string message, Exception? innerException = null)
    : Exception(message, innerException);
