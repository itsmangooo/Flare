using System.Data;
using System.Security.Claims;
using System.Security.Cryptography;
using System.Text;
using Flare.Api.Data;
using Flare.Api.Services;
using Flare.Contracts;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Identity;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.RateLimiting;
using Microsoft.EntityFrameworkCore;
using Microsoft.IdentityModel.Tokens;

namespace Flare.Api.Controllers;

[ApiController]
[Route("api/v1/auth")]
[EnableRateLimiting("auth")]
public sealed class AuthController(
    FlareDbContext database,
    UserManager<ApplicationUser> userManager,
    RoleManager<IdentityRole<Guid>> roleManager,
    SignInManager<ApplicationUser> signInManager,
    ITokenService tokenService,
    IAuditService auditService,
    IConfiguration configuration) : ControllerBase
{
    public const string AdministratorRole = "Administrator";

    [HttpGet("bootstrap/status")]
    [AllowAnonymous]
    public async Task<ActionResult<BootstrapStatusResponse>> BootstrapStatus(CancellationToken cancellationToken) =>
        new BootstrapStatusResponse(!await userManager.Users.AnyAsync(cancellationToken));

    [HttpPost("bootstrap")]
    [AllowAnonymous]
    public async Task<ActionResult<TokenResponse>> Bootstrap(
        BootstrapRequest request, CancellationToken cancellationToken)
    {
        var expectedToken = configuration["FLARE_BOOTSTRAP_TOKEN"];
        if (string.IsNullOrWhiteSpace(expectedToken) || !FixedTimeEquals(expectedToken, request.BootstrapToken))
        {
            await auditService.RecordAsync(null, request.Email, "auth.bootstrap", "first-admin",
                OperationResult.Failed, HttpContext.TraceIdentifier, "Invalid bootstrap token.", cancellationToken);
            return Unauthorized(Problem(title: "Bootstrap authorization failed.", statusCode: StatusCodes.Status401Unauthorized));
        }

        await using var transaction = await database.Database.BeginTransactionAsync(
            IsolationLevel.Serializable, cancellationToken);
        if (await userManager.Users.AnyAsync(cancellationToken))
        {
            return Conflict(Problem(
                title: "Bootstrap is permanently disabled.",
                detail: "An administrator account already exists.",
                statusCode: StatusCodes.Status409Conflict));
        }

        if (!await roleManager.RoleExistsAsync(AdministratorRole))
        {
            var roleResult = await roleManager.CreateAsync(new IdentityRole<Guid>(AdministratorRole));
            if (!roleResult.Succeeded)
            {
                return IdentityProblem(roleResult);
            }
        }

        var email = request.Email.Trim().ToLowerInvariant();
        var user = new ApplicationUser { Id = Guid.NewGuid(), UserName = email, Email = email, EmailConfirmed = true };
        var createResult = await userManager.CreateAsync(user, request.Password);
        if (!createResult.Succeeded)
        {
            return IdentityProblem(createResult);
        }

        var roleAssignment = await userManager.AddToRoleAsync(user, AdministratorRole);
        if (!roleAssignment.Succeeded)
        {
            return IdentityProblem(roleAssignment);
        }

        await auditService.RecordAsync(user.Id, email, "auth.bootstrap", "first-admin",
            OperationResult.Succeeded, HttpContext.TraceIdentifier, null, cancellationToken);
        await transaction.CommitAsync(cancellationToken);
        return Ok(await tokenService.IssueAsync(user, ClientIp(), cancellationToken));
    }

    [HttpPost("login")]
    [AllowAnonymous]
    public async Task<ActionResult<TokenResponse>> Login(LoginRequest request, CancellationToken cancellationToken)
    {
        var email = request.Email.Trim().ToLowerInvariant();
        var user = await userManager.FindByEmailAsync(email);
        if (user is null)
        {
            await auditService.RecordAsync(null, email, "auth.login", email, OperationResult.Failed,
                HttpContext.TraceIdentifier, "Unknown account.", cancellationToken);
            return InvalidLogin();
        }

        var result = await signInManager.CheckPasswordSignInAsync(user, request.Password, lockoutOnFailure: true);
        if (!result.Succeeded)
        {
            var detail = result.IsLockedOut ? "Account locked." : "Invalid credentials.";
            await auditService.RecordAsync(user.Id, email, "auth.login", email, OperationResult.Failed,
                HttpContext.TraceIdentifier, detail, cancellationToken);
            return result.IsLockedOut
                ? StatusCode(StatusCodes.Status423Locked, Problem(title: "Account temporarily locked.",
                    detail: "Try again later.", statusCode: StatusCodes.Status423Locked))
                : InvalidLogin();
        }

        await auditService.RecordAsync(user.Id, email, "auth.login", email, OperationResult.Succeeded,
            HttpContext.TraceIdentifier, null, cancellationToken);
        return Ok(await tokenService.IssueAsync(user, ClientIp(), cancellationToken));
    }

    [HttpPost("refresh")]
    [AllowAnonymous]
    public async Task<ActionResult<TokenResponse>> Refresh(
        RefreshRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var response = await tokenService.RotateAsync(request.RefreshToken, ClientIp(), cancellationToken);
            await auditService.RecordAsync(response.User.Id, response.User.Email, "auth.refresh", "session",
                OperationResult.Succeeded, HttpContext.TraceIdentifier, null, cancellationToken);
            return Ok(response);
        }
        catch (SecurityTokenException)
        {
            await auditService.RecordAsync(null, null, "auth.refresh", "session", OperationResult.Failed,
                HttpContext.TraceIdentifier, "Refresh token was invalid, expired, or reused.", cancellationToken);
            return Unauthorized(Problem(title: "Session expired.",
                detail: "Sign in again to continue.", statusCode: StatusCodes.Status401Unauthorized));
        }
    }

    [HttpPost("logout")]
    [Authorize]
    public async Task<IActionResult> Logout(LogoutRequest request, CancellationToken cancellationToken)
    {
        var revoked = await tokenService.RevokeAsync(request.RefreshToken, ClientIp(), cancellationToken);
        var userId = UserId();
        await auditService.RecordAsync(userId, User.FindFirstValue(ClaimTypes.Email), "auth.logout", "session",
            revoked ? OperationResult.Succeeded : OperationResult.Failed, HttpContext.TraceIdentifier,
            revoked ? null : "Token was already inactive.", cancellationToken);
        return NoContent();
    }

    [HttpGet("me")]
    [Authorize]
    public async Task<ActionResult<UserResponse>> Me(CancellationToken cancellationToken)
    {
        var user = await userManager.FindByIdAsync(UserId().ToString());
        if (user is null)
        {
            return Unauthorized();
        }

        var roles = await userManager.GetRolesAsync(user);
        return new UserResponse(user.Id, user.Email!, roles.ToArray());
    }

    private ActionResult<TokenResponse> InvalidLogin() => Unauthorized(Problem(
        title: "Sign-in failed.", detail: "Email or password is incorrect.",
        statusCode: StatusCodes.Status401Unauthorized));

    private ObjectResult IdentityProblem(IdentityResult result) => Problem(
        title: "Account could not be created.",
        detail: string.Join(' ', result.Errors.Select(error => error.Description)),
        statusCode: StatusCodes.Status400BadRequest);

    private string? ClientIp() => HttpContext.Connection.RemoteIpAddress?.ToString();

    private Guid UserId() => Guid.Parse(User.FindFirstValue(ClaimTypes.NameIdentifier)!);

    private static bool FixedTimeEquals(string expected, string supplied)
    {
        var expectedBytes = Encoding.UTF8.GetBytes(expected);
        var suppliedBytes = Encoding.UTF8.GetBytes(supplied);
        return expectedBytes.Length == suppliedBytes.Length
            && CryptographicOperations.FixedTimeEquals(expectedBytes, suppliedBytes);
    }
}
