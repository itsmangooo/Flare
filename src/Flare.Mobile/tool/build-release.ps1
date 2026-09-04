[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$required = @(
    'FLARE_ANDROID_KEYSTORE',
    'FLARE_ANDROID_KEY_ALIAS',
    'FLARE_ANDROID_KEYSTORE_PASSWORD',
    'FLARE_ANDROID_KEY_PASSWORD'
)

$missing = @($required | Where-Object { [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($_)) })
if ($missing.Count -gt 0) {
    throw "Flare Release signing is not configured. Missing: $($missing -join ', ')."
}

$keystore = [Environment]::GetEnvironmentVariable('FLARE_ANDROID_KEYSTORE')
if (-not (Test-Path -LiteralPath $keystore -PathType Leaf)) {
    throw "FLARE_ANDROID_KEYSTORE does not point to a readable file."
}

$projectRoot = Split-Path -Parent $PSScriptRoot
$pubspec = Get-Content -LiteralPath (Join-Path $projectRoot 'pubspec.yaml') -Raw
if ($pubspec -notmatch '(?m)^version:\s*([0-9]+\.[0-9]+\.[0-9]+)\+([0-9]+)\s*$') {
    throw 'pubspec.yaml must contain a semantic version and integer build number, for example 1.2.0+6.'
}

$displayVersion = $Matches[1]
$buildNumber = $Matches[2]

Push-Location $projectRoot
try {
    & flutter build apk --release
    if ($LASTEXITCODE -ne 0) {
        throw "Flutter Release build failed with exit code $LASTEXITCODE."
    }

    $source = Join-Path $projectRoot 'build\app\outputs\flutter-apk\app-release.apk'
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
        throw 'Flutter completed without producing app-release.apk.'
    }

    $artifactDirectory = Join-Path (Split-Path -Parent (Split-Path -Parent $projectRoot)) 'artifacts'
    New-Item -ItemType Directory -Path $artifactDirectory -Force | Out-Null
    $destination = Join-Path $artifactDirectory "Flare-v$displayVersion-build$buildNumber-android.apk"
    Copy-Item -LiteralPath $source -Destination $destination -Force
    Write-Host "Signed Flare APK: $destination"
}
finally {
    Pop-Location
}
