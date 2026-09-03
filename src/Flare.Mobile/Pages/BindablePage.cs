using System.Runtime.CompilerServices;
using Flare.Mobile.Services;
using Microsoft.Extensions.Logging;

namespace Flare.Mobile.Pages;

public abstract class BindablePage : ContentPage
{
    private bool _isBusy;
    private string? _errorMessage;

    public new bool IsBusy { get => _isBusy; protected set => Set(ref _isBusy, value); }
    public string? ErrorMessage { get => _errorMessage; protected set { if (Set(ref _errorMessage, value)) OnPropertyChanged(nameof(HasError)); } }
    public bool HasError => !string.IsNullOrWhiteSpace(ErrorMessage);

    protected bool Set<T>(ref T field, T value, [CallerMemberName] string? propertyName = null)
    {
        if (EqualityComparer<T>.Default.Equals(field, value)) return false;
        field = value;
        OnPropertyChanged(propertyName);
        return true;
    }

    protected async Task RunUiActionSafelyAsync(
        ILogger logger,
        string operation,
        Func<Task> action)
    {
        try
        {
            await action();
        }
        catch (FlareApiException exception)
        {
            ErrorMessage = exception.Message;
        }
        catch (OperationCanceledException)
        {
            // Page/lifecycle cancellation is expected and does not need a user-facing error.
        }
        catch (Exception exception)
        {
            MobileLog.UiActionFailed(logger, operation, exception);
            ErrorMessage = "Flare could not complete this action. Try again.";
        }
    }

}
