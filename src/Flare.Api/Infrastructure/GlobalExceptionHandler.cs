using Microsoft.AspNetCore.Diagnostics;
using Microsoft.AspNetCore.Mvc;
using Microsoft.EntityFrameworkCore;

namespace Flare.Api.Infrastructure;

public sealed partial class GlobalExceptionHandler(
    IProblemDetailsService problemDetails,
    IHostEnvironment environment,
    ILogger<GlobalExceptionHandler> logger) : IExceptionHandler
{
    public async ValueTask<bool> TryHandleAsync(
        HttpContext httpContext, Exception exception, CancellationToken cancellationToken)
    {
        var (status, title) = exception switch
        {
            ArgumentException => (StatusCodes.Status400BadRequest, "The request is invalid."),
            InfrastructureUnavailableException => (StatusCodes.Status503ServiceUnavailable, "Infrastructure is unavailable."),
            DbUpdateException => (StatusCodes.Status503ServiceUnavailable, "The database operation failed."),
            _ => (StatusCodes.Status500InternalServerError, "An unexpected error occurred.")
        };
        if (status >= StatusCodes.Status500InternalServerError) LogRequestFailure(logger, status, exception);
        else LogRequestRejected(logger, status);
        httpContext.Response.StatusCode = status;
        return await problemDetails.TryWriteAsync(new ProblemDetailsContext
        {
            HttpContext = httpContext,
            Exception = exception,
            ProblemDetails = new ProblemDetails
            {
                Status = status,
                Title = title,
                Detail = environment.IsDevelopment() ? exception.Message : null,
                Extensions = { ["correlationId"] = httpContext.TraceIdentifier }
            }
        });
    }

    [LoggerMessage(LogLevel.Error, "Request failed with status {StatusCode}.")]
    private static partial void LogRequestFailure(ILogger logger, int statusCode, Exception exception);

    [LoggerMessage(LogLevel.Information, "Request was rejected with status {StatusCode}.")]
    private static partial void LogRequestRejected(ILogger logger, int statusCode);
}
