using System.Runtime.CompilerServices;

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

}
