import 'package:flutter/material.dart';

import 'theme_settings.dart';

abstract final class FlareColors {
  static const background = Color(0xFF07090C);
  static const backgroundSecondary = Color(0xFF0C0F14);
  static const surface = Color(0xFF11151B);
  static const surfaceHigh = Color(0xFF171C23);
  static const border = Color(0x12FFFFFF);
  static const borderStrong = Color(0x22FFFFFF);
  static const text = Color(0xFFF5F7FA);
  static const textSecondary = Color(0xFFA1A9B5);
  static const muted = Color(0xFF67707C);
  static const brandDeep = Color(0xFF135DDA);
  static const accent = Color(0xFF2F81F7);
  static const brandCyan = Color(0xFF55D6FF);
  static const accentSoft = Color(0x262F81F7);
  static const success = Color(0xFF55C991);
  static const successSoft = Color(0x2455C991);
  static const warning = Color(0xFFE5AE5B);
  static const warningSoft = Color(0x24E5AE5B);
  static const danger = Color(0xFFF16F75);
  static const dangerSoft = Color(0x24F16F75);
  static const info = Color(0xFF70A8E8);
  static const infoSoft = Color(0x2470A8E8);
}

extension FlareAccentPresentation on FlareAccent {
  String get label => switch (this) {
    FlareAccent.blue => 'Flare Blue',
    FlareAccent.cyan => 'Cyan',
    FlareAccent.violet => 'Violet',
    FlareAccent.emerald => 'Emerald',
    FlareAccent.amber => 'Amber',
  };

  Color get color => switch (this) {
    FlareAccent.blue => const Color(0xFF2F81F7),
    FlareAccent.cyan => const Color(0xFF16B8D4),
    FlareAccent.violet => const Color(0xFF8B6FF2),
    FlareAccent.emerald => const Color(0xFF26A878),
    FlareAccent.amber => const Color(0xFFD49732),
  };
}

@immutable
final class FlarePalette extends ThemeExtension<FlarePalette> {
  const FlarePalette({
    required this.background,
    required this.backgroundSecondary,
    required this.surface,
    required this.surfaceHigh,
    required this.border,
    required this.borderStrong,
    required this.text,
    required this.textSecondary,
    required this.muted,
    required this.accent,
    required this.accentSoft,
    required this.success,
    required this.successSoft,
    required this.warning,
    required this.warningSoft,
    required this.danger,
    required this.dangerSoft,
    required this.info,
    required this.infoSoft,
  });

  factory FlarePalette.dark(FlareAccent accent, {bool oled = false}) =>
      FlarePalette(
        background: oled ? Colors.black : FlareColors.background,
        backgroundSecondary: oled
            ? const Color(0xFF050505)
            : FlareColors.backgroundSecondary,
        surface: oled ? const Color(0xFF0A0A0A) : FlareColors.surface,
        surfaceHigh: oled ? const Color(0xFF121212) : FlareColors.surfaceHigh,
        border: FlareColors.border,
        borderStrong: FlareColors.borderStrong,
        text: FlareColors.text,
        textSecondary: FlareColors.textSecondary,
        muted: FlareColors.muted,
        accent: accent.color,
        accentSoft: accent.color.withAlpha(38),
        success: FlareColors.success,
        successSoft: FlareColors.successSoft,
        warning: FlareColors.warning,
        warningSoft: FlareColors.warningSoft,
        danger: FlareColors.danger,
        dangerSoft: FlareColors.dangerSoft,
        info: FlareColors.info,
        infoSoft: FlareColors.infoSoft,
      );

  factory FlarePalette.light(FlareAccent accent) => FlarePalette(
    background: const Color(0xFFF7F8FA),
    backgroundSecondary: const Color(0xFFF2F4F7),
    surface: Colors.white,
    surfaceHigh: const Color(0xFFF2F4F7),
    border: const Color(0x0F000000),
    borderStrong: const Color(0x1A000000),
    text: const Color(0xFF17191D),
    textSecondary: const Color(0xFF6E7681),
    muted: const Color(0xFF87909C),
    accent: accent.color,
    accentSoft: accent.color.withAlpha(28),
    success: const Color(0xFF197A55),
    successSoft: const Color(0x18197A55),
    warning: const Color(0xFF94600C),
    warningSoft: const Color(0x1894600C),
    danger: const Color(0xFFC43D49),
    dangerSoft: const Color(0x18C43D49),
    info: const Color(0xFF286FA8),
    infoSoft: const Color(0x18286FA8),
  );

  final Color background;
  final Color backgroundSecondary;
  final Color surface;
  final Color surfaceHigh;
  final Color border;
  final Color borderStrong;
  final Color text;
  final Color textSecondary;
  final Color muted;
  final Color accent;
  final Color accentSoft;
  final Color success;
  final Color successSoft;
  final Color warning;
  final Color warningSoft;
  final Color danger;
  final Color dangerSoft;
  final Color info;
  final Color infoSoft;

  static FlarePalette of(BuildContext context) =>
      Theme.of(context).extension<FlarePalette>()!;

  @override
  FlarePalette copyWith({
    Color? background,
    Color? backgroundSecondary,
    Color? surface,
    Color? surfaceHigh,
    Color? border,
    Color? borderStrong,
    Color? text,
    Color? textSecondary,
    Color? muted,
    Color? accent,
    Color? accentSoft,
    Color? success,
    Color? successSoft,
    Color? warning,
    Color? warningSoft,
    Color? danger,
    Color? dangerSoft,
    Color? info,
    Color? infoSoft,
  }) => FlarePalette(
    background: background ?? this.background,
    backgroundSecondary: backgroundSecondary ?? this.backgroundSecondary,
    surface: surface ?? this.surface,
    surfaceHigh: surfaceHigh ?? this.surfaceHigh,
    border: border ?? this.border,
    borderStrong: borderStrong ?? this.borderStrong,
    text: text ?? this.text,
    textSecondary: textSecondary ?? this.textSecondary,
    muted: muted ?? this.muted,
    accent: accent ?? this.accent,
    accentSoft: accentSoft ?? this.accentSoft,
    success: success ?? this.success,
    successSoft: successSoft ?? this.successSoft,
    warning: warning ?? this.warning,
    warningSoft: warningSoft ?? this.warningSoft,
    danger: danger ?? this.danger,
    dangerSoft: dangerSoft ?? this.dangerSoft,
    info: info ?? this.info,
    infoSoft: infoSoft ?? this.infoSoft,
  );

  @override
  FlarePalette lerp(covariant FlarePalette? other, double t) {
    if (other == null) return this;
    return FlarePalette(
      background: Color.lerp(background, other.background, t)!,
      backgroundSecondary: Color.lerp(
        backgroundSecondary,
        other.backgroundSecondary,
        t,
      )!,
      surface: Color.lerp(surface, other.surface, t)!,
      surfaceHigh: Color.lerp(surfaceHigh, other.surfaceHigh, t)!,
      border: Color.lerp(border, other.border, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
      text: Color.lerp(text, other.text, t)!,
      textSecondary: Color.lerp(textSecondary, other.textSecondary, t)!,
      muted: Color.lerp(muted, other.muted, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      accentSoft: Color.lerp(accentSoft, other.accentSoft, t)!,
      success: Color.lerp(success, other.success, t)!,
      successSoft: Color.lerp(successSoft, other.successSoft, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      warningSoft: Color.lerp(warningSoft, other.warningSoft, t)!,
      danger: Color.lerp(danger, other.danger, t)!,
      dangerSoft: Color.lerp(dangerSoft, other.dangerSoft, t)!,
      info: Color.lerp(info, other.info, t)!,
      infoSoft: Color.lerp(infoSoft, other.infoSoft, t)!,
    );
  }
}

extension FlareThemeContext on BuildContext {
  FlarePalette get flare => FlarePalette.of(this);
}

abstract final class FlareSpace {
  static const xxs = 4.0;
  static const xs = 8.0;
  static const sm = 12.0;
  static const md = 16.0;
  static const lg = 24.0;
  static const xl = 32.0;
  static const xxl = 48.0;
}

abstract final class FlareRadii {
  static const small = 7.0;
  static const normal = 11.0;
  static const large = 18.0;
  static const dock = 28.0;
}

abstract final class FlareType {
  static const display = TextStyle(
    fontSize: 30,
    height: 1.1,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.9,
  );
  static const title = TextStyle(
    fontSize: 19,
    height: 1.25,
    fontWeight: FontWeight.w600,
    letterSpacing: -0.25,
  );
  static const metric = TextStyle(
    fontSize: 27,
    height: 1,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.8,
    fontFeatures: [FontFeature.tabularFigures()],
  );
  static const body = TextStyle(
    fontSize: 14,
    height: 1.45,
    fontWeight: FontWeight.w400,
  );
  static const metadata = TextStyle(
    fontSize: 12,
    height: 1.35,
    fontWeight: FontWeight.w500,
  );
  static const label = TextStyle(
    fontSize: 11,
    height: 1.2,
    fontWeight: FontWeight.w600,
    letterSpacing: 0.8,
  );
  static const mono = TextStyle(
    fontFamily: 'monospace',
    fontSize: 12,
    height: 1.5,
  );
}

ThemeData buildFlareTheme({
  Brightness brightness = Brightness.dark,
  FlareAccent accent = FlareAccent.blue,
  bool oled = false,
}) {
  final palette = brightness == Brightness.light
      ? FlarePalette.light(accent)
      : FlarePalette.dark(accent, oled: oled);
  return ThemeData(
    brightness: brightness,
    fontFamily: 'Inter',
    scaffoldBackgroundColor: palette.background,
    colorScheme: ColorScheme(
      brightness: brightness,
      primary: palette.accent,
      onPrimary: brightness == Brightness.light ? Colors.white : palette.text,
      secondary: palette.accent,
      onSecondary: brightness == Brightness.light ? Colors.white : palette.text,
      error: palette.danger,
      onError: Colors.white,
      surface: palette.surface,
      onSurface: palette.text,
    ),
    extensions: <ThemeExtension<dynamic>>[palette],
    splashFactory: NoSplash.splashFactory,
    highlightColor: Colors.transparent,
    hoverColor: Colors.transparent,
    focusColor: Colors.transparent,
    textSelectionTheme: TextSelectionThemeData(
      cursorColor: palette.accent,
      selectionColor: palette.accentSoft,
      selectionHandleColor: palette.accent,
    ),
    pageTransitionsTheme: const PageTransitionsTheme(
      builders: <TargetPlatform, PageTransitionsBuilder>{
        TargetPlatform.android: FadeForwardsPageTransitionsBuilder(),
      },
    ),
  );
}
