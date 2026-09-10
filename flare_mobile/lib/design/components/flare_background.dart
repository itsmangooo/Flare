import 'dart:io';
import 'package:flutter/material.dart';

import '../../core/theme/flare_theme.dart';

final class FlareBackgroundScope extends InheritedWidget {
  const FlareBackgroundScope({
    required this.hasCustomBackground,
    required super.child,
    super.key,
  });

  final bool hasCustomBackground;

  static bool hasCustomBackgroundOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<FlareBackgroundScope>()
          ?.hasCustomBackground ??
      false;

  @override
  bool updateShouldNotify(FlareBackgroundScope oldWidget) =>
      hasCustomBackground != oldWidget.hasCustomBackground;
}

final class FlareAppBackground extends StatelessWidget {
  const FlareAppBackground({required this.child, this.imagePath, super.key});

  final String? imagePath;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final path = imagePath?.trim();
    final hasCustomBackground = path?.isNotEmpty ?? false;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final pixelRatio = MediaQuery.devicePixelRatioOf(context);
    final imageWidth = (MediaQuery.sizeOf(context).width * pixelRatio)
        .round()
        .clamp(720, 2560);

    return FlareBackgroundScope(
      hasCustomBackground: hasCustomBackground,
      child: Stack(
        fit: StackFit.expand,
        children: <Widget>[
          const _AtmosphereBackground(),
          if (hasCustomBackground) ...<Widget>[
            Image.file(
              File(path!),
              fit: BoxFit.cover,
              cacheWidth: imageWidth,
              filterQuality: FilterQuality.medium,
              gaplessPlayback: true,
              excludeFromSemantics: true,
              errorBuilder: (context, error, stackTrace) =>
                  const SizedBox.expand(),
            ),
            DecoratedBox(
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: dark
                      ? <Color>[
                          Color.alphaBlend(
                            palette.accent.withValues(alpha: 0.07),
                            Colors.black.withValues(alpha: 0.28),
                          ),
                          palette.background.withValues(alpha: 0.38),
                        ]
                      : <Color>[
                          Color.alphaBlend(
                            palette.accent.withValues(alpha: 0.05),
                            Colors.white.withValues(alpha: 0.34),
                          ),
                          palette.background.withValues(alpha: 0.46),
                        ],
                ),
              ),
            ),
          ],
          child,
        ],
      ),
    );
  }
}

final class _AtmosphereBackground extends StatelessWidget {
  const _AtmosphereBackground();

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return DecoratedBox(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: const Alignment(0, -0.15),
          colors: <Color>[
            Color.alphaBlend(palette.ambientStart, palette.backgroundSecondary),
            Color.alphaBlend(palette.ambientEnd, palette.background),
          ],
        ),
      ),
    );
  }
}
