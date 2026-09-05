import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

enum FlareThemeMode { system, light, dark, oled }

enum FlareAccent { blue, cyan, violet, emerald, amber }

final class FlareThemeSettings {
  const FlareThemeSettings({
    this.mode = FlareThemeMode.dark,
    this.accent = FlareAccent.blue,
  });

  final FlareThemeMode mode;
  final FlareAccent accent;

  FlareThemeSettings copyWith({FlareThemeMode? mode, FlareAccent? accent}) =>
      FlareThemeSettings(
        mode: mode ?? this.mode,
        accent: accent ?? this.accent,
      );
}

final themeSettingsProvider =
    AsyncNotifierProvider<ThemeSettingsController, FlareThemeSettings>(
      ThemeSettingsController.new,
    );

final class ThemeSettingsController extends AsyncNotifier<FlareThemeSettings> {
  static const _modeKey = 'appearance.theme_mode';
  static const _accentKey = 'appearance.accent';

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
    );
  }

  Future<void> setMode(FlareThemeMode mode) async {
    final current = state.value ?? await future;
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_modeKey, mode.name);
    state = AsyncData(current.copyWith(mode: mode));
  }

  Future<void> setAccent(FlareAccent accent) async {
    final current = state.value ?? await future;
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_accentKey, accent.name);
    state = AsyncData(current.copyWith(accent: accent));
  }
}

T _enumValue<T extends Enum>(List<T> values, String? name, T fallback) {
  for (final value in values) {
    if (value.name == name) return value;
  }
  return fallback;
}
