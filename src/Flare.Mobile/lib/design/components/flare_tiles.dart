import 'package:flutter/material.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/models.dart';
import '../../core/theme/flare_theme.dart';
import '../../core/utils/formatters.dart';
import 'flare_controls.dart';

FlareStatusTone containerTone(ContainerState state, HealthState health) {
  if (health == HealthState.unhealthy || state == ContainerState.dead) {
    return FlareStatusTone.danger;
  }
  if (health == HealthState.starting ||
      state == ContainerState.restarting ||
      state == ContainerState.paused) {
    return FlareStatusTone.warning;
  }
  return state == ContainerState.running
      ? FlareStatusTone.success
      : FlareStatusTone.neutral;
}

final class FlareContainerTile extends StatelessWidget {
  const FlareContainerTile({
    required this.container,
    required this.onTap,
    super.key,
  });
  final ContainerSummaryModel container;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final tone = containerTone(container.state, container.health);
    final status = container.health == HealthState.unhealthy
        ? 'Unhealthy'
        : container.state.name;
    final dotColor = switch (tone) {
      FlareStatusTone.success => palette.success,
      FlareStatusTone.warning => palette.warning,
      FlareStatusTone.danger => palette.danger,
      _ => palette.muted,
    };
    return Semantics(
      button: true,
      label: '${container.name}, $status',
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(FlareRadii.normal),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: FlareSpace.md),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: <Widget>[
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: dotColor.withValues(alpha: 0.11),
                  borderRadius: BorderRadius.circular(12),
                ),
                alignment: Alignment.center,
                child: Stack(
                  clipBehavior: Clip.none,
                  children: <Widget>[
                    PhosphorIcon(
                      PhosphorIconsRegular.cube,
                      size: 19,
                      color: dotColor,
                    ),
                    Positioned(
                      right: -4,
                      bottom: -4,
                      child: Container(
                        width: 8,
                        height: 8,
                        decoration: BoxDecoration(
                          color: dotColor,
                          shape: BoxShape.circle,
                          border: Border.all(
                            color: palette.surface,
                            width: 1.5,
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      container.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: FlareType.body.copyWith(
                        fontWeight: FontWeight.w600,
                        color: palette.text,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      container.image,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: FlareType.metadata.copyWith(color: palette.muted),
                    ),
                    const SizedBox(height: FlareSpace.xxs),
                    Text(
                      'CPU ${formatPercent(container.cpuPercent, decimals: 1)}  ·  RAM ${formatBytes(container.memoryBytes)}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: FlareType.caption.copyWith(
                        color: palette.textSecondary,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: FlareSpace.sm),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: <Widget>[
                  Text(
                    status,
                    style: FlareType.metadata.copyWith(
                      color: dotColor,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  PhosphorIcon(
                    PhosphorIconsRegular.caretRight,
                    size: 15,
                    color: palette.muted,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class FlareActivityTile extends StatelessWidget {
  const FlareActivityTile({required this.event, this.last = false, super.key});
  final ActivityEventModel event;
  final bool last;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final success = event.result == OperationResult.succeeded;
    final security = event.kind == ActivityKind.security;
    final color = success
        ? (security ? palette.info : palette.success)
        : palette.danger;
    final icon = security
        ? PhosphorIconsRegular.shieldCheck
        : PhosphorIconsRegular.pulse;
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: <Widget>[
          SizedBox(
            width: 34,
            child: Column(
              children: <Widget>[
                Container(
                  width: 28,
                  height: 28,
                  decoration: BoxDecoration(
                    color: color.withValues(alpha: 0.12),
                    shape: BoxShape.circle,
                  ),
                  child: Center(
                    child: PhosphorIcon(icon, size: 14, color: color),
                  ),
                ),
                if (!last)
                  Expanded(
                    child: Container(width: 1, color: palette.borderStrong),
                  ),
              ],
            ),
          ),
          const SizedBox(width: FlareSpace.sm),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(bottom: FlareSpace.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          titleCaseAction(event.action),
                          style: FlareType.body.copyWith(
                            fontWeight: FontWeight.w600,
                            color: palette.text,
                          ),
                        ),
                      ),
                      Text(
                        formatRelative(event.timestamp),
                        style: FlareType.metadata.copyWith(
                          color: palette.muted,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 2),
                  Text(
                    event.target,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: FlareType.metadata.copyWith(
                      color: palette.textSecondary,
                    ),
                  ),
                  if (event.actor != null) ...<Widget>[
                    const SizedBox(height: 2),
                    Text(
                      event.actor!,
                      style: FlareType.metadata.copyWith(color: palette.muted),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
