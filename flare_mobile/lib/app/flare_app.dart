import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/providers.dart';
import '../core/routing/app_router.dart';
import '../core/theme/flare_theme.dart';
import '../core/theme/theme_settings.dart';
import '../design/components/flare_background.dart';
import '../design/components/flare_controls.dart';

final class FlareApp extends ConsumerStatefulWidget {
  const FlareApp({super.key});

  @override
  ConsumerState<FlareApp> createState() => _FlareAppState();
}

final class _FlareAppState extends ConsumerState<FlareApp> {
  StreamSubscription<void>? _sessionSubscription;

  @override
  void initState() {
    super.initState();
    _sessionSubscription = ref.read(sessionStoreProvider).changes.listen((_) {
      unawaited(_enforceSessionBoundary());
    });
  }

  Future<void> _enforceSessionBoundary() async {
    final session = ref.read(sessionStoreProvider);
    final serverUrl = await session.serverUrl;
    final hasSession = serverUrl != null && await session.hasUsableSession();
    final location = flareRouter.state.uri.path;
    if (!mounted ||
        location == '/' ||
        location == '/connect' ||
        location == '/login') {
      return;
    }
    if (serverUrl == null) {
      flareRouter.go('/connect');
    } else if (!hasSession) {
      flareRouter.go('/login');
    }
  }

  @override
  void dispose() {
    _sessionSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final settings =
        ref.watch(themeSettingsProvider).value ?? const FlareThemeSettings();
    final mode = switch (settings.mode) {
      FlareThemeMode.system => ThemeMode.system,
      FlareThemeMode.light => ThemeMode.light,
      FlareThemeMode.dark || FlareThemeMode.oled => ThemeMode.dark,
    };
    FlareHaptics.enabled = settings.hapticsEnabled;
    return MaterialApp.router(
      title: 'Flare',
      debugShowCheckedModeBanner: false,
      scrollBehavior: const FlareScrollBehavior(),
      theme: buildFlareTheme(
        brightness: Brightness.light,
        accent: settings.accent,
        atmosphere: settings.atmosphere,
      ),
      darkTheme: buildFlareTheme(
        accent: settings.accent,
        atmosphere: settings.atmosphere,
        oled: settings.mode == FlareThemeMode.oled,
      ),
      themeMode: mode,
      routerConfig: flareRouter,
      builder: (context, child) {
        final dark = Theme.of(context).brightness == Brightness.dark;
        final iconBrightness = dark ? Brightness.light : Brightness.dark;
        return AnnotatedRegion<SystemUiOverlayStyle>(
          value: SystemUiOverlayStyle(
            statusBarColor: Colors.transparent,
            statusBarIconBrightness: iconBrightness,
            statusBarBrightness: dark ? Brightness.dark : Brightness.light,
            systemNavigationBarColor: Colors.transparent,
            systemNavigationBarDividerColor: Colors.transparent,
            systemNavigationBarIconBrightness: iconBrightness,
            systemNavigationBarContrastEnforced: false,
            systemStatusBarContrastEnforced: false,
          ),
          child: FlareAppBackground(
            imagePath: settings.customBackgroundPath,
            child: child ?? const SizedBox.shrink(),
          ),
        );
      },
    );
  }
}
