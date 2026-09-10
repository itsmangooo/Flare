import 'package:flutter/material.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/theme/flare_theme.dart';
import 'flare_controls.dart';

enum FlareToastTone { info, success, error }

abstract final class FlareToast {
  static OverlayEntry? _active;

  static void show(
    BuildContext context,
    String message, {
    FlareToastTone tone = FlareToastTone.info,
  }) {
    _active?.remove();
    final overlay = Overlay.of(context);
    final palette = context.flare;
    final (icon, color) = switch (tone) {
      FlareToastTone.success => (
        PhosphorIconsRegular.checkCircle,
        palette.success,
      ),
      FlareToastTone.error => (
        PhosphorIconsRegular.warningCircle,
        palette.danger,
      ),
      FlareToastTone.info => (PhosphorIconsRegular.info, palette.info),
    };
    late final OverlayEntry entry;
    entry = OverlayEntry(
      builder: (context) => Positioned(
        left: 16,
        right: 16,
        bottom: MediaQuery.paddingOf(context).bottom + 92,
        child: _ToastSurface(icon: icon, color: color, message: message),
      ),
    );
    _active = entry;
    overlay.insert(entry);
    Future<void>.delayed(const Duration(seconds: 4)).then((_) {
      if (_active == entry && entry.mounted) {
        entry.remove();
        _active = null;
      }
    });
  }
}

final class _ToastSurface extends StatelessWidget {
  const _ToastSurface({
    required this.icon,
    required this.color,
    required this.message,
  });
  final PhosphorIconData icon;
  final Color color;
  final String message;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return TweenAnimationBuilder<double>(
      duration: const Duration(milliseconds: 220),
      curve: Curves.easeOutCubic,
      tween: Tween<double>(begin: 0, end: 1),
      builder: (context, value, child) => Transform.translate(
        offset: Offset(0, 12 * (1 - value)),
        child: Opacity(opacity: value, child: child),
      ),
      child: Material(
        color: Colors.transparent,
        child: FlareGlassSurface(
          borderRadius: BorderRadius.circular(FlareRadii.small),
          blurSigma: AppGlassTokens.blur,
          backgroundColor: palette.surfaceHigh.withValues(alpha: 0.86),
          borderColor: color.withValues(alpha: 0.18),
          child: Padding(
            padding: const EdgeInsets.symmetric(
              horizontal: FlareSpace.md,
              vertical: FlareSpace.sm,
            ),
            child: Row(
              children: <Widget>[
                PhosphorIcon(icon, size: 19, color: color),
                const SizedBox(width: FlareSpace.sm),
                Expanded(
                  child: Text(
                    message,
                    style: FlareType.body.copyWith(color: palette.text),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class FlareConfirmSheet {
  const FlareConfirmSheet._();

  static Future<bool> show(
    BuildContext context, {
    required String title,
    required String subject,
    required String message,
    required String confirmLabel,
    bool destructive = false,
  }) async =>
      await FlareBottomSheet.show<bool>(
        context,
        barrierLabel: 'Dismiss confirmation',
        child: _ConfirmSheetBody(
          title: title,
          subject: subject,
          message: message,
          confirmLabel: confirmLabel,
          destructive: destructive,
        ),
      ) ??
      false;
}

final class FlareBottomSheet {
  const FlareBottomSheet._();

  static Future<T?> show<T>(
    BuildContext context, {
    required Widget child,
    String barrierLabel = 'Dismiss sheet',
  }) => showGeneralDialog<T>(
    context: context,
    barrierDismissible: true,
    barrierLabel: barrierLabel,
    barrierColor: const Color(0x99000000),
    transitionDuration: const Duration(milliseconds: 250),
    pageBuilder: (context, primary, secondary) {
      final palette = context.flare;
      return SafeArea(
        child: Align(
          alignment: Alignment.bottomCenter,
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Material(
              color: Colors.transparent,
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 560),
                child: AppGlassSurface(
                  borderRadius: BorderRadius.circular(FlareRadii.sheet),
                  level: AppGlassLevel.sheet,
                  blurSigma: AppGlassTokens.sheetBlur,
                  opacity: AppGlassTokens.sheetOpacity,
                  backgroundColor: palette.surfaceHigh,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(
                      FlareSpace.lg,
                      FlareSpace.sm,
                      FlareSpace.lg,
                      FlareSpace.lg,
                    ),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        Container(
                          width: 34,
                          height: 3,
                          decoration: BoxDecoration(
                            color: palette.textSecondary.withValues(alpha: 0.5),
                            borderRadius: BorderRadius.circular(2),
                          ),
                        ),
                        const SizedBox(height: FlareSpace.md),
                        child,
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      );
    },
    transitionBuilder: (context, animation, secondary, child) {
      final curved = CurvedAnimation(
        parent: animation,
        curve: Curves.easeOutCubic,
        reverseCurve: Curves.easeInCubic,
      );
      return FadeTransition(
        opacity: curved,
        child: SlideTransition(
          position: Tween<Offset>(
            begin: const Offset(0, 0.12),
            end: Offset.zero,
          ).animate(curved),
          child: child,
        ),
      );
    },
  );
}

final class _ConfirmSheetBody extends StatelessWidget {
  const _ConfirmSheetBody({
    required this.title,
    required this.subject,
    required this.message,
    required this.confirmLabel,
    required this.destructive,
  });
  final String title;
  final String subject;
  final String message;
  final String confirmLabel;
  final bool destructive;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(title, style: FlareType.title.copyWith(color: palette.text)),
        const SizedBox(height: FlareSpace.xs),
        Text(
          subject,
          style: FlareType.body.copyWith(
            fontWeight: FontWeight.w600,
            color: palette.text,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          message,
          style: FlareType.body.copyWith(color: palette.textSecondary),
        ),
        const SizedBox(height: FlareSpace.lg),
        Row(
          children: <Widget>[
            Expanded(
              child: FlareButton(
                label: 'Cancel',
                expand: true,
                onPressed: () => Navigator.pop(context, false),
              ),
            ),
            const SizedBox(width: FlareSpace.sm),
            Expanded(
              child: FlareButton(
                label: confirmLabel,
                expand: true,
                tone: destructive
                    ? FlareButtonTone.danger
                    : FlareButtonTone.primary,
                onPressed: () {
                  if (destructive) safeHaptic();
                  Navigator.pop(context, true);
                },
              ),
            ),
          ],
        ),
      ],
    );
  }
}
