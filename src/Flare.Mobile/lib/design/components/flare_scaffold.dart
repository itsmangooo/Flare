import 'package:flutter/material.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/theme/flare_theme.dart';

final class FlareScaffold extends StatelessWidget {
  const FlareScaffold({
    required this.title,
    required this.body,
    this.eyebrow = 'FLARE',
    this.actions = const <Widget>[],
    this.bottomNavigation,
    this.leading,
    this.bodyPadding = const EdgeInsets.symmetric(horizontal: FlareSpace.md),
    super.key,
  });

  final String eyebrow;
  final String title;
  final Widget body;
  final List<Widget> actions;
  final Widget? bottomNavigation;
  final Widget? leading;
  final EdgeInsets bodyPadding;

  @override
  Widget build(BuildContext context) => Scaffold(
    backgroundColor: Colors.transparent,
    resizeToAvoidBottomInset: true,
    body: DecoratedBox(
      decoration: const BoxDecoration(
        color: FlareColors.background,
        gradient: RadialGradient(
          center: Alignment(1.15, -1.05),
          radius: 1.05,
          colors: <Color>[Color(0x162A1713), FlareColors.background],
          stops: <double>[0, 0.76],
        ),
      ),
      child: SafeArea(
        bottom: false,
        child: Column(
          children: <Widget>[
            FlareTopBar(
              eyebrow: eyebrow,
              title: title,
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

final class FlareTopBar extends StatelessWidget {
  const FlareTopBar({
    required this.eyebrow,
    required this.title,
    required this.actions,
    this.leading,
    super.key,
  });

  final String eyebrow;
  final String title;
  final List<Widget> actions;
  final Widget? leading;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(
      FlareSpace.md,
      FlareSpace.md,
      FlareSpace.md,
      FlareSpace.sm,
    ),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.end,
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
                  color: FlareColors.accent,
                  letterSpacing: 2.1,
                ),
              ),
              const SizedBox(height: 5),
              Text(title, style: FlareType.display),
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
  Widget build(BuildContext context) => ColoredBox(
    color: Colors.transparent,
    child: SafeArea(
      minimum: const EdgeInsets.fromLTRB(12, 0, 12, 10),
      child: Container(
        height: 68,
        padding: const EdgeInsets.all(5),
        decoration: BoxDecoration(
          color: const Color(0xF211151B),
          borderRadius: BorderRadius.circular(FlareRadii.dock),
          border: Border.all(color: FlareColors.borderStrong),
          boxShadow: const <BoxShadow>[
            BoxShadow(
              color: Color(0xA6000000),
              blurRadius: 24,
              offset: Offset(0, 10),
            ),
          ],
        ),
        child: LayoutBuilder(
          builder: (context, constraints) {
            final itemWidth = constraints.maxWidth / _items.length;
            return Stack(
              children: <Widget>[
                AnimatedPositioned(
                  duration: const Duration(milliseconds: 190),
                  curve: Curves.easeOutCubic,
                  left: itemWidth * index,
                  top: 0,
                  width: itemWidth,
                  bottom: 0,
                  child: Padding(
                    padding: const EdgeInsets.all(2),
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: FlareColors.accentSoft,
                        borderRadius: BorderRadius.circular(22),
                        border: Border.all(color: const Color(0x36FF7A5C)),
                      ),
                    ),
                  ),
                ),
                Row(
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
                          onTap: () => onSelected(itemIndex),
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: <Widget>[
                              AnimatedScale(
                                scale: selected ? 1.05 : 1,
                                duration: const Duration(milliseconds: 180),
                                child: PhosphorIcon(
                                  selected ? item.selectedIcon : item.icon,
                                  size: 20,
                                  color: selected
                                      ? FlareColors.accent
                                      : FlareColors.muted,
                                ),
                              ),
                              const SizedBox(height: 3),
                              Text(
                                item.label,
                                maxLines: 1,
                                style: FlareType.metadata.copyWith(
                                  fontSize: 9.5,
                                  color: selected
                                      ? FlareColors.text
                                      : FlareColors.muted,
                                  fontWeight: selected
                                      ? FontWeight.w600
                                      : FontWeight.w500,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    );
                  }),
                ),
              ],
            );
          },
        ),
      ),
    ),
  );
}

final class FlareBackButton extends StatelessWidget {
  const FlareBackButton({this.onPressed, super.key});
  final VoidCallback? onPressed;
  @override
  Widget build(BuildContext context) => Semantics(
    button: true,
    label: 'Back',
    child: InkWell(
      customBorder: const CircleBorder(),
      onTap: onPressed ?? () => Navigator.of(context).maybePop(),
      child: const SizedBox(
        width: 40,
        height: 40,
        child: Center(
          child: PhosphorIcon(
            PhosphorIconsRegular.arrowLeft,
            size: 20,
            color: FlareColors.text,
          ),
        ),
      ),
    ),
  );
}
