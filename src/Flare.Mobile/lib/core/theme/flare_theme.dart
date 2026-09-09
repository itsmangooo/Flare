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
    FlareAccent.blue => 'Ocean Blue',
    FlareAccent.cyan => 'Electric Cyan',
    FlareAccent.violet => 'Violet',
    FlareAccent.emerald => 'Flare Emerald',
    FlareAccent.amber => 'Amber',
    FlareAccent.indigo => 'Indigo',
    FlareAccent.rose => 'Rose',
    FlareAccent.coral => 'Coral',
    FlareAccent.slate => 'Slate',
    FlareAccent.mint => 'Mint',
  };

  Color get color => switch (this) {
    FlareAccent.blue => const Color(0xFF2F81F7),
    FlareAccent.cyan => const Color(0xFF16B8D4),
    FlareAccent.violet => const Color(0xFF8B6FF2),
    FlareAccent.emerald => const Color(0xFF26A878),
    FlareAccent.amber => const Color(0xFFD49732),
    FlareAccent.indigo => const Color(0xFF6967E8),
    FlareAccent.rose => const Color(0xFFE0668C),
    FlareAccent.coral => const Color(0xFFE47865),
    FlareAccent.slate => const Color(0xFF748CA3),
    FlareAccent.mint => const Color(0xFF4AC7A3),
  };
}

extension FlareAtmospherePresentation on FlareAtmosphere {
  String get label => switch (this) {
    FlareAtmosphere.none => 'None',
    FlareAtmosphere.softGradient => 'Soft Gradient',
    FlareAtmosphere.aurora => 'Aurora',
    FlareAtmosphere.midnight => 'Midnight',
    FlareAtmosphere.graphite => 'Graphite',
  };
}

extension FlareThemePresetPresentation on FlareThemePreset {
  String get label => switch (this) {
    FlareThemePreset.flareClassic => 'Flare Classic',
    FlareThemePreset.midnightBlue => 'Midnight Blue',
    FlareThemePreset.graphite => 'Graphite',
    FlareThemePreset.aurora => 'Aurora',
    FlareThemePreset.nord => 'Nord',
    FlareThemePreset.ember => 'Ember',
    FlareThemePreset.cloud => 'Cloud',
    FlareThemePreset.lavender => 'Lavender',
    FlareThemePreset.discordInspired => 'Discord-inspired',
  };

  FlareThemeSettings applyTo(FlareThemeSettings current) {
    final (mode, accent, atmosphere) = switch (this) {
      FlareThemePreset.flareClassic => (
        FlareThemeMode.dark,
        FlareAccent.emerald,
        FlareAtmosphere.none,
      ),
      FlareThemePreset.midnightBlue => (
        FlareThemeMode.dark,
        FlareAccent.blue,
        FlareAtmosphere.midnight,
      ),
      FlareThemePreset.graphite => (
        FlareThemeMode.dark,
        FlareAccent.slate,
        FlareAtmosphere.graphite,
      ),
      FlareThemePreset.aurora => (
        FlareThemeMode.dark,
        FlareAccent.cyan,
        FlareAtmosphere.aurora,
      ),
      FlareThemePreset.nord => (
        FlareThemeMode.dark,
        FlareAccent.cyan,
        FlareAtmosphere.midnight,
      ),
      FlareThemePreset.ember => (
        FlareThemeMode.dark,
        FlareAccent.coral,
        FlareAtmosphere.softGradient,
      ),
      FlareThemePreset.cloud => (
        FlareThemeMode.light,
        FlareAccent.blue,
        FlareAtmosphere.none,
      ),
      FlareThemePreset.lavender => (
        FlareThemeMode.dark,
        FlareAccent.violet,
        FlareAtmosphere.softGradient,
      ),
      FlareThemePreset.discordInspired => (
        FlareThemeMode.dark,
        FlareAccent.indigo,
        FlareAtmosphere.graphite,
      ),
    };
    return current.copyWith(mode: mode, accent: accent, atmosphere: atmosphere);
  }
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
    required this.ambientStart,
    required this.ambientEnd,
    required this.glassTint,
  });

  factory FlarePalette.dark(
    FlareAccent accent, {
    bool oled = false,
    FlareAtmosphere atmosphere = FlareAtmosphere.none,
  }) => FlarePalette(
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
    ambientStart: _ambientStart(atmosphere, accent.color, false),
    ambientEnd: _ambientEnd(atmosphere, accent.color, false),
    glassTint: _glassTint(atmosphere, accent.color),
  );

  factory FlarePalette.light(
    FlareAccent accent, {
    FlareAtmosphere atmosphere = FlareAtmosphere.none,
  }) => FlarePalette(
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
    ambientStart: _ambientStart(atmosphere, accent.color, true),
    ambientEnd: _ambientEnd(atmosphere, accent.color, true),
    glassTint: _glassTint(atmosphere, accent.color),
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
  final Color ambientStart;
  final Color ambientEnd;
  final Color glassTint;

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
    Color? ambientStart,
    Color? ambientEnd,
    Color? glassTint,
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
    ambientStart: ambientStart ?? this.ambientStart,
    ambientEnd: ambientEnd ?? this.ambientEnd,
    glassTint: glassTint ?? this.glassTint,
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
      ambientStart: Color.lerp(ambientStart, other.ambientStart, t)!,
      ambientEnd: Color.lerp(ambientEnd, other.ambientEnd, t)!,
      glassTint: Color.lerp(glassTint, other.glassTint, t)!,
    );
  }
}

Color _ambientStart(FlareAtmosphere atmosphere, Color accent, bool light) =>
    switch (atmosphere) {
      FlareAtmosphere.none => Colors.transparent,
      FlareAtmosphere.softGradient => accent.withValues(
        alpha: light ? 0.07 : 0.1,
      ),
      FlareAtmosphere.aurora => const Color(0x2421C990),
      FlareAtmosphere.midnight => const Color(0x263B5FD4),
      FlareAtmosphere.graphite => const Color(0x20788592),
    };

Color _ambientEnd(FlareAtmosphere atmosphere, Color accent, bool light) =>
    switch (atmosphere) {
      FlareAtmosphere.none => Colors.transparent,
      FlareAtmosphere.softGradient => accent.withValues(
        alpha: light ? 0.025 : 0.035,
      ),
      FlareAtmosphere.aurora => const Color(0x1F5A4AE3),
      FlareAtmosphere.midnight => const Color(0x120A2B63),
      FlareAtmosphere.graphite => const Color(0x0F88909A),
    };

Color _glassTint(FlareAtmosphere atmosphere, Color accent) =>
    switch (atmosphere) {
      FlareAtmosphere.none => Colors.transparent,
      FlareAtmosphere.softGradient => accent.withValues(alpha: 0.025),
      FlareAtmosphere.aurora => const Color(0x0D38D9A2),
      FlareAtmosphere.midnight => const Color(0x0F5378E5),
      FlareAtmosphere.graphite => const Color(0x0D89939E),
    };

extension FlareThemeContext on BuildContext {
  FlarePalette get flare => FlarePalette.of(this);
}

final class FlareScrollBehavior extends MaterialScrollBehavior {
  const FlareScrollBehavior();

  @override
  ScrollPhysics getScrollPhysics(BuildContext context) =>
      const BouncingScrollPhysics(parent: AlwaysScrollableScrollPhysics());

  @override
  Widget buildOverscrollIndicator(
    BuildContext context,
    Widget child,
    ScrollableDetails details,
  ) => child;
}

abstract final class FlareSpace {
  static const xxs = 4.0;
  static const xs = 8.0;
  static const sm = 12.0;
  static const md = 16.0;
  static const lg = 20.0;
  static const xl = 24.0;
  static const xxl = 32.0;
  static const huge = 40.0;
}

abstract final class FlareRadii {
  static const small = 16.0;
  static const normal = 20.0;
  static const large = 26.0;
  static const dock = 34.0;
  static const input = 22.0;
  static const sheet = 34.0;
}

abstract final class FlareType {
  static const display = TextStyle(
    fontSize: 34,
    height: 1.08,
    fontWeight: FontWeight.w700,
    letterSpacing: -1.15,
  );
  static const title = TextStyle(
    fontSize: 20,
    height: 1.25,
    fontWeight: FontWeight.w600,
    letterSpacing: -0.35,
  );
  static const metric = TextStyle(
    fontSize: 27,
    height: 1,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.8,
    fontFeatures: [FontFeature.tabularFigures()],
  );
  static const metricCompact = TextStyle(
    fontSize: 22,
    height: 1.05,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.55,
    fontFeatures: [FontFeature.tabularFigures()],
  );
  static const body = TextStyle(
    fontSize: 14.5,
    height: 1.48,
    fontWeight: FontWeight.w400,
  );
  static const metadata = TextStyle(
    fontSize: 12,
    height: 1.35,
    fontWeight: FontWeight.w500,
  );
  static const caption = TextStyle(
    fontSize: 10.5,
    height: 1.3,
    fontWeight: FontWeight.w500,
  );
  static const navigation = TextStyle(
    fontSize: 10,
    height: 1.2,
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
  FlareAtmosphere atmosphere = FlareAtmosphere.none,
  bool oled = false,
}) {
  final palette = brightness == Brightness.light
      ? FlarePalette.light(accent, atmosphere: atmosphere)
      : FlarePalette.dark(accent, oled: oled, atmosphere: atmosphere);
  return ThemeData(
    brightness: brightness,
    fontFamily: 'Poppins',
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
    chipTheme: ChipThemeData(
      backgroundColor: palette.surface.withValues(alpha: 0.68),
      selectedColor: palette.accent.withValues(alpha: 0.14),
      disabledColor: palette.surface.withValues(alpha: 0.36),
      side: BorderSide(color: palette.borderStrong),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(FlareRadii.normal),
      ),
      labelStyle: FlareType.metadata.copyWith(color: palette.textSecondary),
      secondaryLabelStyle: FlareType.metadata.copyWith(
        color: palette.accent,
        fontWeight: FontWeight.w600,
      ),
      showCheckmark: false,
      pressElevation: 0,
      elevation: 0,
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        backgroundColor: WidgetStateProperty.resolveWith((states) {
          if (states.contains(WidgetState.selected)) {
            return palette.accent.withValues(alpha: 0.14);
          }
          return palette.surface.withValues(alpha: 0.62);
        }),
        foregroundColor: WidgetStateProperty.resolveWith((states) {
          return states.contains(WidgetState.selected)
              ? palette.accent
              : palette.textSecondary;
        }),
        side: WidgetStatePropertyAll(BorderSide(color: palette.borderStrong)),
        shape: WidgetStatePropertyAll(
          RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(FlareRadii.normal),
          ),
        ),
        textStyle: WidgetStatePropertyAll(
          FlareType.metadata.copyWith(fontWeight: FontWeight.w600),
        ),
        elevation: const WidgetStatePropertyAll(0),
        overlayColor: const WidgetStatePropertyAll(Colors.transparent),
      ),
    ),
    pageTransitionsTheme: const PageTransitionsTheme(
      builders: <TargetPlatform, PageTransitionsBuilder>{
        TargetPlatform.android: FlarePageTransitionsBuilder(),
      },
    ),
  );
}

final class FlarePageTransitionsBuilder extends PageTransitionsBuilder {
  const FlarePageTransitionsBuilder();

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    if (MediaQuery.disableAnimationsOf(context)) return child;
    final curved = CurvedAnimation(
      parent: animation,
      curve: Curves.easeOutCubic,
      reverseCurve: Curves.easeInCubic,
    );
    return FadeTransition(
      opacity: curved,
      child: SlideTransition(
        position: Tween<Offset>(
          begin: const Offset(0.025, 0),
          end: Offset.zero,
        ).animate(curved),
        child: child,
      ),
    );
  }

  @override
  Duration get transitionDuration => const Duration(milliseconds: 220);

  @override
  Duration get reverseTransitionDuration => const Duration(milliseconds: 190);
}
