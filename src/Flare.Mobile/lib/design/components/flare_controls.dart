import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/theme/flare_theme.dart';

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
    final content = AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      padding: padding,
      decoration: BoxDecoration(
        color: FlareColors.surface,
        borderRadius: BorderRadius.circular(FlareRadii.normal),
        border: Border.all(color: FlareColors.border),
      ),
      child: child,
    );
    return onTap == null
        ? content
        : Semantics(
            button: true,
            child: InkWell(
              borderRadius: BorderRadius.circular(FlareRadii.normal),
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
    final enabled = widget.onPressed != null && !widget.loading;
    final (background, foreground, border) = switch (widget.tone) {
      FlareButtonTone.primary => (
        FlareColors.accent,
        const Color(0xFF04101E),
        const Color(0xFF68B2FF),
      ),
      FlareButtonTone.danger => (
        FlareColors.dangerSoft,
        const Color(0xFFFFC7C9),
        const Color(0x55F16F75),
      ),
      FlareButtonTone.neutral => (
        FlareColors.surfaceHigh,
        FlareColors.text,
        FlareColors.borderStrong,
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
                borderRadius: BorderRadius.circular(
                  widget.compact ? FlareRadii.small : FlareRadii.normal,
                ),
                border: Border.all(color: border),
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
  Widget build(BuildContext context) => Semantics(
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
        child: Container(
          width: 42,
          height: 42,
          decoration: BoxDecoration(
            color: widget.accent ? FlareColors.accentSoft : FlareColors.surface,
            borderRadius: BorderRadius.circular(FlareRadii.normal),
            border: Border.all(
              color: widget.accent
                  ? const Color(0x442F81F7)
                  : FlareColors.border,
            ),
          ),
          alignment: Alignment.center,
          child: PhosphorIcon(
            widget.icon,
            size: 20,
            color: widget.accent
                ? FlareColors.accent
                : FlareColors.textSecondary,
          ),
        ),
      ),
    ),
  );
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
    final error = widget.error;
    final borderColor = error != null
        ? FlareColors.danger
        : _focus.hasFocus
        ? FlareColors.accent
        : FlareColors.borderStrong;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Padding(
          padding: const EdgeInsets.only(left: 2, bottom: 7),
          child: Text(
            widget.label.toUpperCase(),
            style: FlareType.label.copyWith(
              color: _focus.hasFocus ? FlareColors.accent : FlareColors.muted,
            ),
          ),
        ),
        AnimatedContainer(
          duration: const Duration(milliseconds: 170),
          constraints: const BoxConstraints(minHeight: 52),
          padding: const EdgeInsets.symmetric(horizontal: 14),
          decoration: BoxDecoration(
            color: FlareColors.surface,
            borderRadius: BorderRadius.circular(FlareRadii.normal),
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
                  color: _focus.hasFocus
                      ? FlareColors.accent
                      : FlareColors.muted,
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
                  style: FlareType.body.copyWith(fontSize: 15),
                  cursorColor: FlareColors.accent,
                  decoration: InputDecoration.collapsed(
                    hintText: widget.hint,
                    hintStyle: FlareType.body.copyWith(
                      color: FlareColors.muted,
                    ),
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
                        color: FlareColors.muted,
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
                    style: FlareType.metadata.copyWith(
                      color: FlareColors.danger,
                    ),
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
  Widget build(BuildContext context) => Container(
    height: 44,
    padding: const EdgeInsets.symmetric(horizontal: 13),
    decoration: BoxDecoration(
      color: FlareColors.surface,
      borderRadius: BorderRadius.circular(FlareRadii.normal),
      border: Border.all(color: FlareColors.border),
    ),
    child: Row(
      children: <Widget>[
        const PhosphorIcon(
          PhosphorIconsRegular.magnifyingGlass,
          size: 18,
          color: FlareColors.muted,
        ),
        const SizedBox(width: 9),
        Expanded(
          child: TextField(
            controller: controller,
            onChanged: onChanged,
            style: FlareType.body,
            decoration: InputDecoration.collapsed(
              hintText: hint,
              hintStyle: FlareType.body.copyWith(color: FlareColors.muted),
            ),
          ),
        ),
        if (controller.text.isNotEmpty)
          InkWell(
            onTap: () {
              controller.clear();
              onChanged('');
            },
            child: const Padding(
              padding: EdgeInsets.all(6),
              child: PhosphorIcon(
                PhosphorIconsRegular.x,
                size: 16,
                color: FlareColors.muted,
              ),
            ),
          ),
      ],
    ),
  );
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
    final (color, background) = switch (tone) {
      FlareStatusTone.success => (FlareColors.success, FlareColors.successSoft),
      FlareStatusTone.warning => (FlareColors.warning, FlareColors.warningSoft),
      FlareStatusTone.danger => (FlareColors.danger, FlareColors.dangerSoft),
      FlareStatusTone.info => (FlareColors.info, FlareColors.infoSoft),
      FlareStatusTone.neutral => (
        FlareColors.textSecondary,
        const Color(0x1467707C),
      ),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(99),
        border: Border.all(color: color.withValues(alpha: 0.22)),
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
            label.toUpperCase(),
            style: FlareType.label.copyWith(
              fontSize: 9.5,
              color: color,
              letterSpacing: 0.65,
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
  Widget build(BuildContext context) => Row(
    children: <Widget>[
      Expanded(child: Text(title, style: FlareType.title)),
      if (actionLabel != null)
        InkWell(
          onTap: onAction,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text(
              actionLabel!,
              style: FlareType.metadata.copyWith(
                color: FlareColors.accent,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
    ],
  );
}

final class FlareDivider extends StatelessWidget {
  const FlareDivider({this.indent = 0, super.key});
  final double indent;
  @override
  Widget build(BuildContext context) => Container(
    height: 1,
    margin: EdgeInsets.only(left: indent),
    color: FlareColors.border,
  );
}

Future<void> safeHaptic() async {
  try {
    await HapticFeedback.mediumImpact();
  } on Object {
    // Haptics are optional and must never affect an administrative action.
  }
}
