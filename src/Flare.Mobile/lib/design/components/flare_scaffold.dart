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
    final palette = context.flare;
    return Scaffold(
      backgroundColor: Colors.transparent,
      resizeToAvoidBottomInset: true,
      body: DecoratedBox(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: const Alignment(0, -0.15),
            colors: <Color>[
              palette.backgroundSecondary.withValues(alpha: 0.68),
              palette.background,
            ],
          ),
        ),
        child: SafeArea(
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
                const SizedBox(height: 5),
                Text(
                  title,
                  style: FlareType.display.copyWith(color: palette.text),
                ),
                if (subtitle != null) ...<Widget>[
                  const SizedBox(height: 5),
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
          selectedIcon: PhosphorIconsFill.gauge,
        ),
        (
          label: 'Containers',
          icon: PhosphorIconsRegular.cube,
          selectedIcon: PhosphorIconsFill.cube,
        ),
        (
          label: 'Deployments',
          icon: PhosphorIconsRegular.rocketLaunch,
          selectedIcon: PhosphorIconsFill.rocketLaunch,
        ),
        (
          label: 'Activity',
          icon: PhosphorIconsRegular.pulse,
          selectedIcon: PhosphorIconsFill.pulse,
        ),
      ];

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final dockRadius = BorderRadius.circular(FlareRadii.dock);
    return SafeArea(
      minimum: const EdgeInsets.fromLTRB(14, 0, 14, 10),
      child: SizedBox(
        height: 66,
        child: FlareGlassSurface(
          borderRadius: dockRadius,
          blurSigma: 12,
          backgroundColor: palette.surfaceHigh.withValues(alpha: 0.7),
          borderColor: palette.text.withValues(alpha: 0.07),
          child: Padding(
            padding: const EdgeInsets.all(5),
            child: LayoutBuilder(
              builder: (context, constraints) {
                return Material(
                  color: Colors.transparent,
                  child: Row(
                    children: List<Widget>.generate(_items.length, (itemIndex) {
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
                                AnimatedContainer(
                                  duration: const Duration(milliseconds: 190),
                                  curve: Curves.easeOutCubic,
                                  width: 34,
                                  height: 31,
                                  alignment: Alignment.center,
                                  decoration: BoxDecoration(
                                    color: selected
                                        ? palette.accent.withValues(alpha: 0.13)
                                        : Colors.transparent,
                                    borderRadius: BorderRadius.circular(11),
                                  ),
                                  child: AnimatedScale(
                                    scale: selected ? 1.06 : 1,
                                    duration: const Duration(milliseconds: 190),
                                    curve: Curves.easeOutCubic,
                                    child: PhosphorIcon(
                                      selected ? item.selectedIcon : item.icon,
                                      size: selected ? 20.5 : 19,
                                      color: selected
                                          ? palette.accent
                                          : palette.textSecondary.withValues(
                                              alpha: 0.68,
                                            ),
                                    ),
                                  ),
                                ),
                                const SizedBox(height: 1),
                                AnimatedDefaultTextStyle(
                                  duration: const Duration(milliseconds: 190),
                                  curve: Curves.easeOutCubic,
                                  style: FlareType.metadata.copyWith(
                                    fontSize: 9.2,
                                    color: selected
                                        ? palette.text
                                        : palette.textSecondary.withValues(
                                            alpha: 0.68,
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
        child: FlareGlassSurface(
          borderRadius: BorderRadius.circular(20),
          blurSigma: 7,
          backgroundColor: palette.surface.withValues(alpha: 0.68),
          borderColor: palette.text.withValues(alpha: 0.07),
          child: SizedBox(
            width: 40,
            height: 40,
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
