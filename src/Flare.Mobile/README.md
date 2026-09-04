# Flare Mobile

Native Flutter/Dart Android client for Flare. The app talks only to the public Flare.Api HTTPS origin and never receives Docker, PostgreSQL, or Coolify credentials.

## Development

```powershell
flutter pub get
flutter analyze --fatal-infos
flutter test
flutter build apk --debug
```

Release signing and versioning are documented in the repository's `docs/DEPLOYMENT.md`. Release builds require all four `FLARE_ANDROID_*` environment variables and never fall back to debug signing. After setting them, `./tool/build-release.ps1` creates a signed, version-named APK under the repository's `artifacts/` directory.
