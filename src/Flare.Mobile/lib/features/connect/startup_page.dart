import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_brand.dart';

final class StartupPage extends ConsumerWidget {
  const StartupPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.listen(startupDestinationProvider, (previous, next) {
      next.whenData((destination) {
        final route = switch (destination) {
          StartupDestination.connect => '/connect',
          StartupDestination.login => '/login',
          StartupDestination.overview => '/overview',
        };
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (context.mounted) context.go(route);
        });
      });
    });
    final state = ref.watch(startupDestinationProvider);
    final palette = context.flare;
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            const FlareMark(size: 64)
                .animate()
                .fadeIn(duration: 240.ms)
                .scaleXY(begin: 0.94, end: 1, curve: Curves.easeOutCubic),
            const SizedBox(height: FlareSpace.md),
            Text(
              'FLARE',
              style: FlareType.label.copyWith(
                color: palette.text,
                letterSpacing: 4,
              ),
            ),
            if (state.hasError) ...<Widget>[
              const SizedBox(height: 20),
              Text(
                'Secure storage could not be opened.',
                style: FlareType.metadata.copyWith(color: palette.danger),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
