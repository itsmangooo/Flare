import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/providers.dart';
import '../core/routing/app_router.dart';
import '../core/theme/flare_theme.dart';

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
  Widget build(BuildContext context) => MaterialApp.router(
    title: 'Flare',
    debugShowCheckedModeBanner: false,
    theme: buildFlareTheme(),
    darkTheme: buildFlareTheme(),
    themeMode: ThemeMode.dark,
    routerConfig: flareRouter,
  );
}
