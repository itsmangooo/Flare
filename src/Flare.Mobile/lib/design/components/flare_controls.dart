import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/theme/flare_theme.dart';

final class FlareGlassSurface extends StatelessWidget {
  const FlareGlassSurface({
    required this.child,
    required this.borderRadius,
    this.blurSigma = 10,
    this.backgroundColor,
    this.borderColor,
    super.key,
  });

  final Widget child;
  final BorderRadius borderRadius;
  final double blurSigma;
  final Color? backgroundColor;
  final Color? borderColor;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final dark = Theme.of(context).brightness == Brightness.dark;
    return RepaintBoundary(
      child: ClipRRect(
        borderRadius: borderRadius,
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: blurSigma, sigmaY: blurSigma),
          child: DecoratedBox(
            decoration: BoxDecoration(
              color:
                  backgroundColor ??
                  palette.surfaceHigh.withValues(alpha: dark ? 0.72 : 0.78),
              borderRadius: borderRadius,
              border: Border.all(
                color:
                    borderColor ??
                    (dark
                        ? Colors.white.withValues(alpha: 0.08)
                        : Colors.black.withValues(alpha: 0.07)),
              ),
            ),
            child: child,
          ),
        ),
      ),
    );
  }
}

final class FlareCard extends StatelessWidget {
  const FlareCard({
    required this.child,
    this.padding = const EdgeInsets.all(FlareSpace.md),
    this.onTap,
    super.key,
  });
  final Widget child;
  final EdgeInsets padding;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final content = AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      padding: padding,
      decoration: BoxDecoration(
        color: palette.surface.withValues(alpha: 0.86),
        borderRadius: BorderRadius.circular(FlareRadii.large),
        border: Border.all(color: palette.text.withValues(alpha: 0.045)),
      ),
      child: child,
    );
    return onTap == null
        ? content
        : Semantics(
            button: true,
            child: InkWell(
              borderRadius: BorderRadius.circular(FlareRadii.large),
              onTap: onTap,
              child: content,
            ),
          );
  }
}

enum FlareButtonTone { primary, neutral, danger }

final class FlareButton extends StatefulWidget {
  const FlareButton({
    required this.label,
    required this.onPressed,
    this.icon,
    this.tone = FlareButtonTone.neutral,
    this.loading = false,
    this.compact = false,
    this.expand = false,
    super.key,
  });
  final String label;
  final VoidCallback? onPressed;
  final PhosphorIconData? icon;
  final FlareButtonTone tone;
  final bool loading;
  final bool compact;
  final bool expand;

  @override
  State<FlareButton> createState() => _FlareButtonState();
}

final class _FlareButtonState extends State<FlareButton> {
  bool _pressed = false;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final enabled = widget.onPressed != null && !widget.loading;
    final (background, foreground, border) = switch (widget.tone) {
      FlareButtonTone.primary => (
        palette.accent,
        ThemeData.estimateBrightnessForColor(palette.accent) == Brightness.dark
            ? Colors.white
            : const Color(0xFF071018),
        palette.accent.withValues(alpha: 0.72),
      ),
      FlareButtonTone.danger => (
        palette.dangerSoft,
        palette.danger,
        Colors.transparent,
      ),
      FlareButtonTone.neutral => (
        palette.surfaceHigh.withValues(alpha: 0.9),
        palette.text,
        Colors.transparent,
      ),
    };
    final button = Semantics(
      button: true,
      enabled: enabled,
      label: widget.label,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTapDown: enabled ? (_) => setState(() => _pressed = true) : null,
        onTapUp: enabled ? (_) => setState(() => _pressed = false) : null,
        onTapCancel: enabled ? () => setState(() => _pressed = false) : null,
        onTap: enabled ? widget.onPressed : null,
        child: AnimatedScale(
          scale: _pressed ? 0.98 : 1,
          duration: const Duration(milliseconds: 120),
          curve: Curves.easeOut,
          child: AnimatedOpacity(
            opacity: enabled ? 1 : 0.42,
            duration: const Duration(milliseconds: 150),
            child: Container(
              constraints: BoxConstraints(minHeight: widget.compact ? 36 : 46),
              padding: EdgeInsets.symmetric(
                horizontal: widget.compact ? 12 : 17,
                vertical: widget.compact ? 8 : 12,
              ),
              decoration: BoxDecoration(
                color: background,
                borderRadius: BorderRadius.circular(widget.compact ? 12 : 15),
                border: border == Colors.transparent
                    ? null
                    : Border.all(color: border),
              ),
              child: Row(
                mainAxisSize: widget.expand
                    ? MainAxisSize.max
                    : MainAxisSize.min,
                mainAxisAlignment: MainAxisAlignment.center,
                children: <Widget>[
                  if (widget.loading)
                    SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 1.8,
                        color: foreground,
                      ),
                    )
                  else if (widget.icon != null)
                    PhosphorIcon(
                      widget.icon!,
                      size: widget.compact ? 16 : 18,
                      color: foreground,
                    ),
                  if (widget.loading || widget.icon != null)
                    const SizedBox(width: 8),
                  Flexible(
                    child: Text(
                      widget.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: FlareType.body.copyWith(
                        color: foreground,
                        fontWeight: FontWeight.w600,
                        fontSize: widget.compact ? 12 : 14,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    return widget.expand
        ? SizedBox(width: double.infinity, child: button)
        : button;
  }
}

final class FlareIconButton extends StatefulWidget {
  const FlareIconButton({
    required this.icon,
    required this.onPressed,
    required this.semanticLabel,
    this.accent = false,
    super.key,
  });
  final PhosphorIconData icon;
  final VoidCallback? onPressed;
  final String semanticLabel;
  final bool accent;
  @override
  State<FlareIconButton> createState() => _FlareIconButtonState();
}

final class _FlareIconButtonState extends State<FlareIconButton> {
  bool pressed = false;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Semantics(
      button: true,
      label: widget.semanticLabel,
      child: GestureDetector(
        onTapDown: widget.onPressed == null
            ? null
            : (_) => setState(() => pressed = true),
        onTapUp: widget.onPressed == null
            ? null
            : (_) => setState(() => pressed = false),
        onTapCancel: widget.onPressed == null
            ? null
            : () => setState(() => pressed = false),
        onTap: widget.onPressed,
        child: AnimatedScale(
          scale: pressed ? 0.94 : 1,
          duration: const Duration(milliseconds: 120),
          child: FlareGlassSurface(
            borderRadius: BorderRadius.circular(FlareRadii.normal),
            blurSigma: 7,
            backgroundColor: widget.accent
                ? palette.accent.withValues(alpha: 0.11)
                : palette.surface.withValues(alpha: 0.68),
            borderColor: widget.accent
                ? palette.accent.withValues(alpha: 0.24)
                : palette.text.withValues(alpha: 0.07),
            child: SizedBox(
              width: 42,
              height: 42,
              child: Center(
                child: PhosphorIcon(
                  widget.icon,
                  size: 20,
                  color: widget.accent ? palette.accent : palette.textSecondary,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class FlareTextField extends StatefulWidget {
  const FlareTextField({
    required this.controller,
    required this.label,
    this.hint,
    this.error,
    this.obscureText = false,
    this.keyboardType,
    this.textInputAction,
    this.onSubmitted,
    this.autofillHints,
    this.enabled = true,
    this.leading,
    super.key,
  });
  final TextEditingController controller;
  final String label;
  final String? hint;
  final String? error;
  final bool obscureText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final ValueChanged<String>? onSubmitted;
  final Iterable<String>? autofillHints;
  final bool enabled;
  final PhosphorIconData? leading;

  @override
  State<FlareTextField> createState() => _FlareTextFieldState();
}

final class _FlareTextFieldState extends State<FlareTextField> {
  final FocusNode _focus = FocusNode();
  bool _hide = false;

  @override
  void initState() {
    super.initState();
    _hide = widget.obscureText;
    _focus.addListener(_focusChanged);
  }

  @override
  void didUpdateWidget(covariant FlareTextField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.obscureText != widget.obscureText) _hide = widget.obscureText;
  }

  void _focusChanged() => setState(() {});

  @override
  void dispose() {
    _focus.removeListener(_focusChanged);
    _focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final error = widget.error;
    final borderColor = error != null
        ? palette.danger
        : _focus.hasFocus
        ? palette.accent
        : palette.borderStrong;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.only(left: 2, bottom: 7),
          child: Text(
            widget.label.toUpperCase(),
            style: FlareType.label.copyWith(
              color: _focus.hasFocus ? palette.accent : palette.muted,
            ),
          ),
        ),
        AnimatedContainer(
          duration: const Duration(milliseconds: 170),
          constraints: const BoxConstraints(minHeight: 54),
          padding: const EdgeInsets.symmetric(horizontal: 15),
          decoration: BoxDecoration(
            color: palette.surface.withValues(alpha: 0.9),
            borderRadius: BorderRadius.circular(15),
            border: Border.all(
              color: borderColor,
              width: _focus.hasFocus ? 1.2 : 1,
            ),
          ),
          child: Row(
            children: <Widget>[
              if (widget.leading != null) ...<Widget>[
                PhosphorIcon(
                  widget.leading!,
                  size: 19,
                  color: _focus.hasFocus ? palette.accent : palette.muted,
                ),
                const SizedBox(width: 10),
              ],
              Expanded(
                child: TextField(
                  controller: widget.controller,
                  focusNode: _focus,
                  enabled: widget.enabled,
                  obscureText: _hide,
                  keyboardType: widget.keyboardType,
                  textInputAction: widget.textInputAction,
                  onSubmitted: widget.onSubmitted,
                  autofillHints: widget.autofillHints,
                  style: FlareType.body.copyWith(
                    fontSize: 15,
                    color: palette.text,
                  ),
                  cursorColor: palette.accent,
                  decoration: InputDecoration.collapsed(
                    hintText: widget.hint,
                    hintStyle: FlareType.body.copyWith(color: palette.muted),
                  ),
                ),
              ),
              if (widget.obscureText)
                Semantics(
                  button: true,
                  label: _hide ? 'Show password' : 'Hide password',
                  child: InkWell(
                    onTap: () => setState(() => _hide = !_hide),
                    child: Padding(
                      padding: const EdgeInsets.all(8),
                      child: PhosphorIcon(
                        _hide
                            ? PhosphorIconsRegular.eye
                            : PhosphorIconsRegular.eyeSlash,
                        size: 18,
                        color: palette.muted,
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ),
        AnimatedSize(
          duration: const Duration(milliseconds: 160),
          alignment: Alignment.topLeft,
          child: error == null
              ? const SizedBox.shrink()
              : Padding(
                  padding: const EdgeInsets.only(top: 6, left: 2),
                  child: Text(
                    error,
                    style: FlareType.metadata.copyWith(color: palette.danger),
                  ),
                ),
        ),
      ],
    );
  }
}

final class FlareSearchField extends StatelessWidget {
  const FlareSearchField({
    required this.controller,
    required this.onChanged,
    this.hint = 'Search',
    super.key,
  });
  final TextEditingController controller;
  final ValueChanged<String> onChanged;
  final String hint;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Container(
      height: 46,
      padding: const EdgeInsets.symmetric(horizontal: 14),
      decoration: BoxDecoration(
        color: palette.surfaceHigh.withValues(alpha: 0.82),
        borderRadius: BorderRadius.circular(15),
        border: Border.all(color: palette.text.withValues(alpha: 0.045)),
      ),
      child: Row(
        children: <Widget>[
          PhosphorIcon(
            PhosphorIconsRegular.magnifyingGlass,
            size: 18,
            color: palette.muted,
          ),
          const SizedBox(width: 9),
          Expanded(
            child: TextField(
              controller: controller,
              onChanged: onChanged,
              style: FlareType.body.copyWith(color: palette.text),
              decoration: InputDecoration.collapsed(
                hintText: hint,
                hintStyle: FlareType.body.copyWith(color: palette.muted),
              ),
            ),
          ),
          if (controller.text.isNotEmpty)
            InkWell(
              onTap: () {
                controller.clear();
                onChanged('');
              },
              child: Padding(
                padding: const EdgeInsets.all(6),
                child: PhosphorIcon(
                  PhosphorIconsRegular.x,
                  size: 16,
                  color: palette.muted,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

enum FlareStatusTone { success, warning, danger, info, neutral }

final class FlareStatusBadge extends StatelessWidget {
  const FlareStatusBadge({
    required this.label,
    this.tone = FlareStatusTone.neutral,
    this.dot = true,
    super.key,
  });
  final String label;
  final FlareStatusTone tone;
  final bool dot;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final (color, background) = switch (tone) {
      FlareStatusTone.success => (palette.success, palette.successSoft),
      FlareStatusTone.warning => (palette.warning, palette.warningSoft),
      FlareStatusTone.danger => (palette.danger, palette.dangerSoft),
      FlareStatusTone.info => (palette.info, palette.infoSoft),
      FlareStatusTone.neutral => (
        palette.textSecondary,
        palette.muted.withValues(alpha: 0.08),
      ),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5.5),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(99),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          if (dot) ...<Widget>[
            Container(
              width: 6,
              height: 6,
              decoration: BoxDecoration(color: color, shape: BoxShape.circle),
            ),
            const SizedBox(width: 6),
          ],
          Text(
            label,
            style: FlareType.label.copyWith(
              fontSize: 10,
              color: color,
              letterSpacing: 0.15,
            ),
          ),
        ],
      ),
    );
  }
}

final class FlareSectionHeader extends StatelessWidget {
  const FlareSectionHeader({
    required this.title,
    this.actionLabel,
    this.onAction,
    super.key,
  });
  final String title;
  final String? actionLabel;
  final VoidCallback? onAction;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Row(
      children: <Widget>[
        Expanded(
          child: Text(
            title,
            style: FlareType.title.copyWith(color: palette.text),
          ),
        ),
        if (actionLabel != null)
          InkWell(
            onTap: onAction,
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(
                actionLabel!,
                style: FlareType.metadata.copyWith(
                  color: palette.accent,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ),
      ],
    );
  }
}

final class FlareDivider extends StatelessWidget {
  const FlareDivider({this.indent = 0, super.key});
  final double indent;
  @override
  Widget build(BuildContext context) => Container(
    height: 1,
    margin: EdgeInsets.only(left: indent),
    color: context.flare.text.withValues(alpha: 0.055),
  );
}

final class FlareGroupedSurface extends StatelessWidget {
  const FlareGroupedSurface({
    required this.child,
    this.padding = EdgeInsets.zero,
    this.margin = EdgeInsets.zero,
    super.key,
  });

  final Widget child;
  final EdgeInsets padding;
  final EdgeInsets margin;

  @override
  Widget build(BuildContext context) => Container(
    margin: margin,
    padding: padding,
    decoration: BoxDecoration(
      color: context.flare.surface.withValues(alpha: 0.86),
      borderRadius: BorderRadius.circular(FlareRadii.large),
      border: Border.all(color: context.flare.text.withValues(alpha: 0.045)),
    ),
    clipBehavior: Clip.antiAlias,
    child: child,
  );
}

final class FlareSegmentedControl<T> extends StatelessWidget {
  const FlareSegmentedControl({
    required this.value,
    required this.items,
    required this.onChanged,
    super.key,
  });

  final T value;
  final List<(T, String)> items;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: palette.surfaceHigh.withValues(alpha: 0.72),
        borderRadius: BorderRadius.circular(13),
        border: Border.all(color: palette.text.withValues(alpha: 0.04)),
      ),
      child: Row(
        children: items
            .map((item) {
              final selected = item.$1 == value;
              return Expanded(
                child: Semantics(
                  button: true,
                  selected: selected,
                  child: GestureDetector(
                    behavior: HitTestBehavior.opaque,
                    onTap: () {
                      if (!selected) {
                        safeSelectionHaptic();
                        onChanged(item.$1);
                      }
                    },
                    child: AnimatedContainer(
                      duration: const Duration(milliseconds: 190),
                      curve: Curves.easeOutCubic,
                      alignment: Alignment.center,
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 8,
                      ),
                      decoration: BoxDecoration(
                        color: selected
                            ? palette.surfaceHigh.withValues(alpha: 0.94)
                            : Colors.transparent,
                        borderRadius: BorderRadius.circular(10),
                        border: selected
                            ? Border.all(
                                color: palette.text.withValues(alpha: 0.065),
                              )
                            : null,
                      ),
                      child: AnimatedDefaultTextStyle(
                        duration: const Duration(milliseconds: 190),
                        style: FlareType.metadata.copyWith(
                          color: selected ? palette.text : palette.muted,
                          fontWeight: selected
                              ? FontWeight.w600
                              : FontWeight.w500,
                        ),
                        child: Text(
                          item.$2,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ),
                  ),
                ),
              );
            })
            .toList(growable: false),
      ),
    );
  }
}

final class FlareSwitch extends StatelessWidget {
  const FlareSwitch({
    required this.value,
    required this.onChanged,
    this.semanticLabel,
    super.key,
  });

  final bool value;
  final ValueChanged<bool>? onChanged;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final enabled = onChanged != null;
    return Semantics(
      toggled: value,
      enabled: enabled,
      label: semanticLabel,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: enabled
            ? () {
                safeSelectionHaptic();
                onChanged!(!value);
              }
            : null,
        child: AnimatedOpacity(
          duration: const Duration(milliseconds: 180),
          opacity: enabled ? 1 : 0.45,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 210),
            curve: Curves.easeOutCubic,
            width: 48,
            height: 28,
            padding: const EdgeInsets.all(3),
            decoration: BoxDecoration(
              color: value
                  ? palette.accent
                  : palette.muted.withValues(alpha: 0.28),
              borderRadius: BorderRadius.circular(99),
            ),
            child: AnimatedAlign(
              duration: const Duration(milliseconds: 210),
              curve: Curves.easeOutCubic,
              alignment: value ? Alignment.centerRight : Alignment.centerLeft,
              child: Container(
                width: 22,
                height: 22,
                decoration: BoxDecoration(
                  color: Colors.white,
                  shape: BoxShape.circle,
                  boxShadow: <BoxShadow>[
                    BoxShadow(
                      color: Colors.black.withValues(alpha: 0.12),
                      blurRadius: 3,
                      offset: const Offset(0, 1),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<void> safeHaptic() async {
  try {
    await HapticFeedback.mediumImpact();
  } on Object {
    // Haptics are optional and must never affect an administrative action.
  }
}

Future<void> safeSelectionHaptic() async {
  try {
    await HapticFeedback.selectionClick();
  } on Object {
    // Selection haptics are a progressive enhancement.
  }
}
