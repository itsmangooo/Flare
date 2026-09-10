import 'package:flutter/material.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/theme/flare_theme.dart';
import 'flare_controls.dart';

final class FlareScaffold extends StatelessWidget {
  const FlareScaffold({
    required this.title,
    required this.body,
    this.eyebrow = 'FLARE',
    this.subtitle,
    this.actions = const <Widget>[],
    this.bottomNavigation,
    this.leading,
    this.bodyPadding = const EdgeInsets.symmetric(horizontal: FlareSpace.md),
    super.key,
  });

  final String eyebrow;
  final String title;
  final String? subtitle;
  final Widget body;
  final List<Widget> actions;
  final Widget? bottomNavigation;
  final Widget? leading;
  final EdgeInsets bodyPadding;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.transparent,
      resizeToAvoidBottomInset: true,
      body: SafeArea(
        child: Column(
          children: <Widget>[
            FlareTopBar(
              eyebrow: eyebrow,
              title: title,
              subtitle: subtitle,
              actions: actions,
              leading: leading,
            ),
            Expanded(
              child: Padding(padding: bodyPadding, child: body),
            ),
          ],
        ),
      ),
      bottomNavigationBar: bottomNavigation,
    );
  }
}

final class FlareTopBar extends StatelessWidget {
  const FlareTopBar({
    required this.eyebrow,
    required this.title,
    this.subtitle,
    required this.actions,
    this.leading,
    super.key,
  });

  final String eyebrow;
  final String title;
  final String? subtitle;
  final List<Widget> actions;
  final Widget? leading;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Padding(
      padding: const EdgeInsets.fromLTRB(FlareSpace.md, 20, FlareSpace.md, 16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: <Widget>[
          if (leading != null) ...<Widget>[
            leading!,
            const SizedBox(width: FlareSpace.sm),
          ],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  eyebrow.toUpperCase(),
                  style: FlareType.label.copyWith(
                    color: palette.accent,
                    letterSpacing: 2.1,
                  ),
                ),
                const SizedBox(height: FlareSpace.xxs),
                Text(
                  title,
                  style: FlareType.display.copyWith(color: palette.text),
                ),
                if (subtitle != null) ...<Widget>[
                  const SizedBox(height: FlareSpace.xxs),
                  Text(
                    subtitle!,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: FlareType.metadata.copyWith(
                      color: palette.textSecondary,
                    ),
                  ),
                ],
              ],
            ),
          ),
          ...actions.map(
            (action) => Padding(
              padding: const EdgeInsets.only(left: FlareSpace.xs),
              child: action,
            ),
          ),
        ],
      ),
    );
  }
}

final class FlareBottomNav extends StatelessWidget {
  const FlareBottomNav({
    required this.index,
    required this.onSelected,
    super.key,
  });

  final int index;
  final ValueChanged<int> onSelected;

  static final _items =
      <({String label, PhosphorIconData icon, PhosphorIconData selectedIcon})>[
        (
          label: 'Overview',
          icon: PhosphorIconsRegular.gauge,
          selectedIcon: PhosphorIconsRegular.gauge,
        ),
        (
          label: 'Containers',
          icon: PhosphorIconsRegular.cube,
          selectedIcon: PhosphorIconsRegular.cube,
        ),
        (
          label: 'Deployments',
          icon: PhosphorIconsRegular.rocketLaunch,
          selectedIcon: PhosphorIconsRegular.rocketLaunch,
        ),
        (
          label: 'Activity',
          icon: PhosphorIconsRegular.pulse,
          selectedIcon: PhosphorIconsRegular.pulse,
        ),
      ];

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final dockRadius = BorderRadius.circular(FlareRadii.dock);
    return SafeArea(
      maintainBottomViewPadding: true,
      minimum: const EdgeInsets.fromLTRB(FlareSpace.md, 0, FlareSpace.md, 14),
      child: SizedBox(
        height: 66,
        child: AppGlassSurface(
          borderRadius: dockRadius,
          level: AppGlassLevel.dock,
          blurSigma: AppGlassTokens.dockBlur,
          opacity: AppGlassTokens.dockOpacity,
          grainOpacity: AppGlassTokens.grainOpacity,
          backgroundColor: palette.surfaceHigh,
          child: Padding(
            padding: const EdgeInsets.all(FlareSpace.xxs),
            child: LayoutBuilder(
              builder: (context, constraints) {
                final itemWidth = constraints.maxWidth / _items.length;
                final duration = MediaQuery.disableAnimationsOf(context)
                    ? Duration.zero
                    : const Duration(milliseconds: 220);
                return Material(
                  color: Colors.transparent,
                  child: Stack(
                    children: <Widget>[
                      AnimatedPositioned(
                        duration: duration,
                        curve: Curves.easeOutCubic,
                        left: itemWidth * index + (itemWidth - 34) / 2,
                        top: 4,
                        width: 34,
                        height: 31,
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            color: palette.accent.withValues(alpha: 0.13),
                            borderRadius: BorderRadius.circular(
                              FlareRadii.small,
                            ),
                            border: Border.all(
                              color: palette.accent.withValues(alpha: 0.1),
                            ),
                          ),
                        ),
                      ),
                      Row(
                        children: List<Widget>.generate(_items.length, (
                          itemIndex,
                        ) {
                          final item = _items[itemIndex];
                          final selected = index == itemIndex;
                          return Expanded(
                            child: Semantics(
                              button: true,
                              selected: selected,
                              label: item.label,
                              child: InkWell(
                                customBorder: const StadiumBorder(),
                                onTap: () {
                                  if (!selected) safeSelectionHaptic();
                                  onSelected(itemIndex);
                                },
                                child: Column(
                                  mainAxisAlignment: MainAxisAlignment.center,
                                  children: <Widget>[
                                    SizedBox(
                                      width: 34,
                                      height: 31,
                                      child: AnimatedScale(
                                        scale: selected ? 1.06 : 1,
                                        duration: duration,
                                        curve: Curves.easeOutCubic,
                                        child: Center(
                                          child: PhosphorIcon(
                                            selected
                                                ? item.selectedIcon
                                                : item.icon,
                                            size: selected ? 20.5 : 19,
                                            color: selected
                                                ? palette.accent
                                                : palette.textSecondary
                                                      .withValues(alpha: 0.62),
                                          ),
                                        ),
                                      ),
                                    ),
                                    const SizedBox(height: 1),
                                    AnimatedDefaultTextStyle(
                                      duration: duration,
                                      curve: Curves.easeOutCubic,
                                      style: FlareType.navigation.copyWith(
                                        color: selected
                                            ? palette.text
                                            : palette.textSecondary.withValues(
                                                alpha: 0.62,
                                              ),
                                        fontWeight: selected
                                            ? FontWeight.w600
                                            : FontWeight.w500,
                                      ),
                                      child: Text(item.label, maxLines: 1),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          );
                        }),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}

final class FlareBackButton extends StatelessWidget {
  const FlareBackButton({this.onPressed, super.key});
  final VoidCallback? onPressed;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Semantics(
      button: true,
      label: 'Back',
      child: InkWell(
        customBorder: const CircleBorder(),
        onTap: onPressed ?? () => Navigator.of(context).maybePop(),
        child: AppGlassSurface(
          borderRadius: BorderRadius.circular(FlareRadii.small),
          level: AppGlassLevel.compact,
          blurSigma: AppGlassTokens.compactBlur,
          opacity: 0.68,
          backgroundColor: palette.surface,
          child: SizedBox(
            width: 44,
            height: 44,
            child: Center(
              child: PhosphorIcon(
                PhosphorIconsRegular.arrowLeft,
                size: 20,
                color: palette.text,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
