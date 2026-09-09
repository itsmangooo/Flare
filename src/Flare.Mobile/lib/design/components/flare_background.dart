import 'dart:io';
import 'dart:ui';

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
            ClipRect(
              child: ImageFiltered(
                imageFilter: ImageFilter.blur(sigmaX: 3.5, sigmaY: 3.5),
                child: Transform.scale(
                  scale: 1.025,
                  child: Image.file(
                    File(path!),
                    fit: BoxFit.cover,
                    cacheWidth: imageWidth,
                    filterQuality: FilterQuality.medium,
                    gaplessPlayback: true,
                    excludeFromSemantics: true,
                    errorBuilder: (context, error, stackTrace) =>
                        const SizedBox.expand(),
                  ),
                ),
              ),
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
                            Colors.black.withValues(alpha: 0.42),
                          ),
                          palette.background.withValues(alpha: 0.58),
                        ]
                      : <Color>[
                          Color.alphaBlend(
                            palette.accent.withValues(alpha: 0.05),
                            Colors.white.withValues(alpha: 0.58),
                          ),
                          palette.background.withValues(alpha: 0.68),
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
