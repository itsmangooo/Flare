import 'package:flutter/material.dart';

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
  static const accent = Color(0xFFFF6847);
  static const accentSoft = Color(0x26FF6847);
  static const success = Color(0xFF55C991);
  static const successSoft = Color(0x2455C991);
  static const warning = Color(0xFFE5AE5B);
  static const warningSoft = Color(0x24E5AE5B);
  static const danger = Color(0xFFF16F75);
  static const dangerSoft = Color(0x24F16F75);
  static const info = Color(0xFF70A8E8);
  static const infoSoft = Color(0x2470A8E8);
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
    color: FlareColors.text,
  );
  static const title = TextStyle(
    fontSize: 19,
    height: 1.25,
    fontWeight: FontWeight.w600,
    letterSpacing: -0.25,
    color: FlareColors.text,
  );
  static const metric = TextStyle(
    fontSize: 27,
    height: 1,
    fontWeight: FontWeight.w700,
    letterSpacing: -0.8,
    color: FlareColors.text,
    fontFeatures: [FontFeature.tabularFigures()],
  );
  static const body = TextStyle(
    fontSize: 14,
    height: 1.45,
    fontWeight: FontWeight.w400,
    color: FlareColors.text,
  );
  static const metadata = TextStyle(
    fontSize: 12,
    height: 1.35,
    fontWeight: FontWeight.w500,
    color: FlareColors.textSecondary,
  );
  static const label = TextStyle(
    fontSize: 11,
    height: 1.2,
    fontWeight: FontWeight.w600,
    letterSpacing: 0.8,
    color: FlareColors.muted,
  );
  static const mono = TextStyle(
    fontFamily: 'monospace',
    fontSize: 12,
    height: 1.5,
    color: FlareColors.textSecondary,
  );
}

ThemeData buildFlareTheme() => ThemeData(
  brightness: Brightness.dark,
  fontFamily: 'Inter',
  scaffoldBackgroundColor: FlareColors.background,
  colorScheme: const ColorScheme.dark(
    primary: FlareColors.accent,
    surface: FlareColors.surface,
    error: FlareColors.danger,
  ),
  splashFactory: NoSplash.splashFactory,
  highlightColor: Colors.transparent,
  hoverColor: Colors.transparent,
  focusColor: Colors.transparent,
  textSelectionTheme: const TextSelectionThemeData(
    cursorColor: FlareColors.accent,
    selectionColor: FlareColors.accentSoft,
    selectionHandleColor: FlareColors.accent,
  ),
  pageTransitionsTheme: const PageTransitionsTheme(
    builders: <TargetPlatform, PageTransitionsBuilder>{
      TargetPlatform.android: FadeForwardsPageTransitionsBuilder(),
    },
  ),
);
