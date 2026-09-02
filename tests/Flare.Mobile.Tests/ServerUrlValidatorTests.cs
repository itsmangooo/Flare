using Flare.Mobile.Services;

namespace Flare.Mobile.Tests;

public sealed class ServerUrlValidatorTests
{
    [Theory]
    [InlineData("https://flare.example.com", "https://flare.example.com")]
    [InlineData(" https://flare.example.com/path ", "https://flare.example.com")]
    [InlineData("https://flare.example.com:8443", "https://flare.example.com:8443")]
    public void AcceptsAndNormalizesHttpsOrigins(string input, string expected)
    {
        Assert.True(ServerUrlValidator.TryNormalize(input, false, out var normalized, out _));
        Assert.Equal(expected, normalized);
    }

    [Theory]
    [InlineData("http://flare.example.com")]
    [InlineData("https://user:password@flare.example.com")]
    [InlineData("https://flare.example.com?token=secret")]
    [InlineData("not a url")]
    public void RejectsInsecureOrCredentialBearingUrls(string input)
    {
        Assert.False(ServerUrlValidator.TryNormalize(input, false, out _, out var error));
        Assert.NotEmpty(error);
    }
}
