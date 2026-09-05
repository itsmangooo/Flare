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
      FlareStatusTone.success => FlareColors.success,
      FlareStatusTone.warning => FlareColors.warning,
      FlareStatusTone.danger => FlareColors.danger,
      _ => palette.muted,
    };
    return Semantics(
      button: true,
      label: '${container.name}, $status',
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(FlareRadii.small),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 13, horizontal: 3),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Padding(
                padding: const EdgeInsets.only(top: 7),
                child: Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    color: dotColor,
                    shape: BoxShape.circle,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Row(
                      children: <Widget>[
                        Expanded(
                          child: Text(
                            container.name,
                            style: FlareType.body.copyWith(
                              fontWeight: FontWeight.w600,
                              fontSize: 15,
                              color: palette.text,
                            ),
                          ),
                        ),
                        FlareStatusBadge(label: status, tone: tone, dot: false),
                      ],
                    ),
                    const SizedBox(height: 3),
                    Text(
                      container.image,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: FlareType.metadata.copyWith(color: palette.muted),
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: <Widget>[
                        Text(
                          'CPU ${formatPercent(container.cpuPercent, decimals: 1)}',
                          style: FlareType.metadata.copyWith(
                            color: palette.textSecondary,
                          ),
                        ),
                        const SizedBox(width: 18),
                        Text(
                          'RAM ${formatBytes(container.memoryBytes)}',
                          style: FlareType.metadata.copyWith(
                            color: palette.textSecondary,
                          ),
                        ),
                        const Spacer(),
                        if (container.startedAt != null)
                          Text(
                            formatRelative(container.startedAt),
                            style: FlareType.metadata.copyWith(
                              color: palette.muted,
                            ),
                          ),
                      ],
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 5),
              Padding(
                padding: const EdgeInsets.only(top: 18),
                child: PhosphorIcon(
                  PhosphorIconsRegular.caretRight,
                  size: 16,
                  color: palette.muted,
                ),
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
        ? (security ? FlareColors.info : FlareColors.success)
        : FlareColors.danger;
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
          const SizedBox(width: 10),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(bottom: 18),
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
