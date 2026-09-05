import 'package:flutter/material.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/theme/flare_theme.dart';
import 'flare_controls.dart';

final class FlareLoading extends StatelessWidget {
  const FlareLoading({this.label = 'Loading', super.key});
  final String label;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          SizedBox(
            width: 22,
            height: 22,
            child: CircularProgressIndicator(
              strokeWidth: 2,
              color: palette.accent,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            label,
            style: FlareType.metadata.copyWith(color: palette.textSecondary),
          ),
        ],
      ),
    );
  }
}

final class FlareEmptyState extends StatelessWidget {
  const FlareEmptyState({
    required this.title,
    required this.message,
    this.icon = PhosphorIconsRegular.tray,
    super.key,
  });
  final String title;
  final String message;
  final PhosphorIconData icon;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(FlareSpace.xl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            PhosphorIcon(icon, size: 30, color: palette.muted),
            const SizedBox(height: 13),
            Text(
              title,
              style: FlareType.title.copyWith(color: palette.text),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 6),
            Text(
              message,
              style: FlareType.body.copyWith(color: palette.textSecondary),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }
}

final class FlareErrorState extends StatelessWidget {
  const FlareErrorState({
    required this.error,
    required this.onRetry,
    super.key,
  });
  final Object error;
  final VoidCallback onRetry;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final message = error is FlareApiException
        ? (error as FlareApiException).message
        : 'Flare could not load this data.';
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(FlareSpace.xl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            PhosphorIcon(
              PhosphorIconsRegular.cloudSlash,
              size: 32,
              color: palette.danger,
            ),
            const SizedBox(height: 13),
            Text(
              'Data unavailable',
              style: FlareType.title.copyWith(color: palette.text),
            ),
            const SizedBox(height: 6),
            Text(
              message,
              style: FlareType.body.copyWith(color: palette.textSecondary),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 18),
            FlareButton(
              label: 'Try again',
              icon: PhosphorIconsRegular.arrowClockwise,
              onPressed: onRetry,
            ),
          ],
        ),
      ),
    );
  }
}
