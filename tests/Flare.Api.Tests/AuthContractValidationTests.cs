using System.ComponentModel.DataAnnotations;
using Flare.Contracts;

namespace Flare.Api.Tests;

public sealed class AuthContractValidationTests
{
    [Theory]
    [InlineData(typeof(BootstrapRequest))]
    [InlineData(typeof(LoginRequest))]
    [InlineData(typeof(RefreshRequest))]
    [InlineData(typeof(LogoutRequest))]
    public void ValidationMetadataIsAttachedToRecordConstructorParameters(Type contractType)
    {
        var constructor = Assert.Single(contractType.GetConstructors());
        Assert.All(constructor.GetParameters(), parameter =>
            Assert.Contains(parameter.GetCustomAttributes(inherit: true), attribute => attribute is ValidationAttribute));
        Assert.All(contractType.GetProperties(), property =>
            Assert.DoesNotContain(property.GetCustomAttributes(inherit: true), attribute => attribute is ValidationAttribute));
    }
}
