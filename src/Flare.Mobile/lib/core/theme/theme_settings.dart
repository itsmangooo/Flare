import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

enum FlareThemeMode { system, light, dark, oled }

enum FlareAccent {
  blue,
  cyan,
  violet,
  emerald,
  amber,
  indigo,
  rose,
  coral,
  slate,
  mint,
}

enum FlareAtmosphere { none, softGradient, aurora, midnight, graphite }

enum FlareThemePreset {
  flareClassic,
  midnightBlue,
  graphite,
  aurora,
  nord,
  ember,
  cloud,
  lavender,
  discordInspired,
}

final class FlareThemeSettings {
  const FlareThemeSettings({
    this.mode = FlareThemeMode.dark,
    this.accent = FlareAccent.blue,
    this.atmosphere = FlareAtmosphere.none,
    this.hapticsEnabled = true,
  });

  final FlareThemeMode mode;
  final FlareAccent accent;
  final FlareAtmosphere atmosphere;
  final bool hapticsEnabled;

  FlareThemeSettings copyWith({
    FlareThemeMode? mode,
    FlareAccent? accent,
    FlareAtmosphere? atmosphere,
    bool? hapticsEnabled,
  }) => FlareThemeSettings(
    mode: mode ?? this.mode,
    accent: accent ?? this.accent,
    atmosphere: atmosphere ?? this.atmosphere,
    hapticsEnabled: hapticsEnabled ?? this.hapticsEnabled,
  );
}

final themeSettingsProvider =
    AsyncNotifierProvider<ThemeSettingsController, FlareThemeSettings>(
      ThemeSettingsController.new,
    );

final class ThemeSettingsController extends AsyncNotifier<FlareThemeSettings> {
  static const _modeKey = 'appearance.theme_mode';
  static const _accentKey = 'appearance.accent';
  static const _atmosphereKey = 'appearance.atmosphere';
  static const _hapticsKey = 'interactions.haptics';

  @override
  Future<FlareThemeSettings> build() async {
    final preferences = await SharedPreferences.getInstance();
    return FlareThemeSettings(
      mode: _enumValue(
        FlareThemeMode.values,
        preferences.getString(_modeKey),
        FlareThemeMode.dark,
      ),
      accent: _enumValue(
        FlareAccent.values,
        preferences.getString(_accentKey),
        FlareAccent.blue,
      ),
      atmosphere: _enumValue(
        FlareAtmosphere.values,
        preferences.getString(_atmosphereKey),
        FlareAtmosphere.none,
      ),
      hapticsEnabled: preferences.getBool(_hapticsKey) ?? true,
    );
  }

  Future<void> setMode(FlareThemeMode mode) async {
    await save((state.value ?? await future).copyWith(mode: mode));
  }

  Future<void> setAccent(FlareAccent accent) async {
    await save((state.value ?? await future).copyWith(accent: accent));
  }

  Future<void> setAtmosphere(FlareAtmosphere atmosphere) async {
    await save((state.value ?? await future).copyWith(atmosphere: atmosphere));
  }

  Future<void> setHapticsEnabled(bool enabled) async {
    await save((state.value ?? await future).copyWith(hapticsEnabled: enabled));
  }

  void preview(FlareThemeSettings settings) => state = AsyncData(settings);

  Future<void> save(FlareThemeSettings settings) async {
    final preferences = await SharedPreferences.getInstance();
    await Future.wait(<Future<bool>>[
      preferences.setString(_modeKey, settings.mode.name),
      preferences.setString(_accentKey, settings.accent.name),
      preferences.setString(_atmosphereKey, settings.atmosphere.name),
      preferences.setBool(_hapticsKey, settings.hapticsEnabled),
    ]);
    state = AsyncData(settings);
  }
}

T _enumValue<T extends Enum>(List<T> values, String? name, T fallback) {
  for (final value in values) {
    if (value.name == name) return value;
  }
  return fallback;
}
