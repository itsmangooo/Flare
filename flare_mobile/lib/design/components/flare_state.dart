import 'package:flutter/material.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/theme/flare_theme.dart';
import 'flare_controls.dart';

final class FlareLoading extends StatefulWidget {
  const FlareLoading({this.label = 'Loading', super.key});
  final String label;

  @override
  State<FlareLoading> createState() => _FlareLoadingState();
}

final class _FlareLoadingState extends State<FlareLoading>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1100),
    )..repeat(reverse: true);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final reduceMotion = MediaQuery.disableAnimationsOf(context);
    return Center(
      child: Semantics(
        liveRegion: true,
        label: widget.label,
        child: ExcludeSemantics(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              AnimatedBuilder(
                animation: _controller,
                builder: (context, child) {
                  final alpha = reduceMotion
                      ? 0.5
                      : 0.28 + _controller.value * 0.32;
                  return Column(
                    children: <Widget>[
                      for (final width in <double>[176, 136, 96]) ...<Widget>[
                        Container(
                          width: width,
                          height: 8,
                          decoration: BoxDecoration(
                            color: palette.accent.withValues(alpha: alpha),
                            borderRadius: BorderRadius.circular(
                              FlareRadii.small,
                            ),
                          ),
                        ),
                        const SizedBox(height: FlareSpace.xs),
                      ],
                    ],
                  );
                },
              ),
              const SizedBox(height: FlareSpace.xxs),
              Text(
                widget.label,
                style: FlareType.metadata.copyWith(
                  color: palette.textSecondary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class FlareEmptyState extends StatelessWidget {
  const FlareEmptyState({
    required this.title,
    required this.message,
    this.icon = PhosphorIconsRegular.tray,
    this.actionLabel,
    this.onAction,
    super.key,
  });
  final String title;
  final String message;
  final PhosphorIconData icon;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(FlareSpace.xxl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            PhosphorIcon(icon, size: 32, color: palette.muted),
            const SizedBox(height: FlareSpace.sm),
            Text(
              title,
              style: FlareType.title.copyWith(color: palette.text),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: FlareSpace.xs),
            Text(
              message,
              style: FlareType.body.copyWith(color: palette.textSecondary),
              textAlign: TextAlign.center,
            ),
            if (actionLabel != null && onAction != null) ...<Widget>[
              const SizedBox(height: FlareSpace.lg),
              FlareButton(label: actionLabel!, onPressed: onAction),
            ],
          ],
        ),
      ),
    );
  }
}

final class FlareErrorState extends StatefulWidget {
  const FlareErrorState({
    required this.error,
    required this.onRetry,
    super.key,
  });
  final Object error;
  final VoidCallback onRetry;

  @override
  State<FlareErrorState> createState() => _FlareErrorStateState();
}

final class _FlareErrorStateState extends State<FlareErrorState> {
  bool _showDetails = false;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final message = widget.error is FlareApiException
        ? (widget.error as FlareApiException).message
        : 'Flare could not load this data.';
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(FlareSpace.xxl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            PhosphorIcon(
              PhosphorIconsRegular.cloudSlash,
              size: 32,
              color: palette.danger,
            ),
            const SizedBox(height: FlareSpace.sm),
            Text(
              'Data unavailable',
              style: FlareType.title.copyWith(color: palette.text),
            ),
            const SizedBox(height: FlareSpace.xs),
            Text(
              message,
              style: FlareType.body.copyWith(color: palette.textSecondary),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: FlareSpace.md),
            FlareButton(
              label: 'Try again',
              icon: PhosphorIconsRegular.arrowClockwise,
              onPressed: widget.onRetry,
            ),
            const SizedBox(height: FlareSpace.xs),
            Semantics(
              button: true,
              expanded: _showDetails,
              child: InkWell(
                borderRadius: BorderRadius.circular(FlareRadii.normal),
                onTap: () => setState(() => _showDetails = !_showDetails),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 44),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: FlareSpace.sm,
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        Text(
                          _showDetails ? 'Hide details' : 'Technical details',
                          style: FlareType.metadata.copyWith(
                            color: palette.muted,
                          ),
                        ),
                        const SizedBox(width: FlareSpace.xs),
                        PhosphorIcon(
                          _showDetails
                              ? PhosphorIconsRegular.caretUp
                              : PhosphorIconsRegular.caretDown,
                          size: 14,
                          color: palette.muted,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
            AnimatedSize(
              duration: const Duration(milliseconds: 180),
              child: _showDetails
                  ? Padding(
                      padding: const EdgeInsets.only(top: FlareSpace.xs),
                      child: AppGlassSurface(
                        borderRadius: BorderRadius.circular(FlareRadii.normal),
                        level: AppGlassLevel.compact,
                        padding: const EdgeInsets.all(FlareSpace.sm),
                        child: SizedBox(
                          width: double.infinity,
                          child: SelectableText(
                            widget.error.toString(),
                            style: FlareType.mono.copyWith(
                              color: palette.textSecondary,
                            ),
                          ),
                        ),
                      ),
                    )
                  : const SizedBox.shrink(),
            ),
          ],
        ),
      ),
    );
  }
}
