import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../core/utils/formatters.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';

final class DeploymentsPage extends ConsumerWidget {
  const DeploymentsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final deployments = ref.watch(deploymentsProvider);
    final palette = context.flare;
    return FlareScaffold(
      title: 'Deployments',
      subtitle: 'Chronological Coolify release history',
      actions: <Widget>[
        FlareIconButton(
          icon: PhosphorIconsRegular.stack,
          semanticLabel: 'Coolify resources',
          onPressed: () => context.push('/coolify'),
        ),
        FlareIconButton(
          icon: PhosphorIconsRegular.userCircle,
          semanticLabel: 'Settings',
          onPressed: () => context.push('/settings'),
        ),
      ],
      body: deployments.when(
        loading: () => const FlareLoading(label: 'Reading Coolify deployments'),
        error: (error, _) => FlareErrorState(
          error: error,
          onRetry: () => ref.invalidate(deploymentsProvider),
        ),
        data: (items) => items.isEmpty
            ? const FlareEmptyState(
                title: 'No deployments',
                message: 'Coolify returned no recent deployments.',
                icon: PhosphorIconsRegular.rocketLaunch,
              )
            : RefreshIndicator(
                color: palette.accent,
                backgroundColor: palette.surfaceHigh,
                onRefresh: () async => ref.refresh(deploymentsProvider.future),
                child: ListView.builder(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.only(top: 8, bottom: 24),
                  itemCount: items.length,
                  itemBuilder: (context, index) => _DeploymentTile(
                    deployment: items[index],
                    last: index == items.length - 1,
                    onTap: () => context.push(
                      '/deployments/${Uri.encodeComponent(items[index].uuid)}',
                    ),
                  ),
                ),
              ),
      ),
    );
  }
}

(FlareStatusTone, Color, String) deploymentStatus(
  String? raw,
  FlarePalette palette,
) {
  final status = raw?.toLowerCase() ?? 'unknown';
  if (status.contains('success') ||
      status.contains('finish') ||
      status.contains('complete')) {
    return (FlareStatusTone.success, palette.success, 'Successful');
  }
  if (status.contains('fail') ||
      status.contains('error') ||
      status.contains('cancel')) {
    return (
      FlareStatusTone.danger,
      palette.danger,
      status.contains('cancel') ? 'Cancelled' : 'Failed',
    );
  }
  if (status.contains('progress') ||
      status.contains('running') ||
      status.contains('queue')) {
    return (FlareStatusTone.warning, palette.warning, 'In progress');
  }
  return (FlareStatusTone.neutral, palette.muted, raw ?? 'Unknown');
}

final class _DeploymentTile extends StatelessWidget {
  const _DeploymentTile({
    required this.deployment,
    required this.onTap,
    required this.last,
  });

  final DeploymentModel deployment;
  final VoidCallback onTap;
  final bool last;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final (_, color, status) = deploymentStatus(deployment.status, palette);
    return IntrinsicHeight(
      child: InkWell(
        borderRadius: BorderRadius.circular(FlareRadii.normal),
        onTap: onTap,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: <Widget>[
            SizedBox(
              width: 42,
              child: Column(
                children: <Widget>[
                  Container(
                    width: 36,
                    height: 36,
                    decoration: BoxDecoration(
                      color: color.withValues(alpha: 0.12),
                      shape: BoxShape.circle,
                    ),
                    alignment: Alignment.center,
                    child: PhosphorIcon(
                      PhosphorIconsRegular.rocketLaunch,
                      size: 17,
                      color: color,
                    ),
                  ),
                  if (!last)
                    Expanded(
                      child: Container(
                        width: 1,
                        color: palette.text.withValues(alpha: 0.07),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Padding(
                padding: EdgeInsets.only(bottom: last ? 8 : 22),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Row(
                      children: <Widget>[
                        Expanded(
                          child: Text(
                            deployment.resourceName,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: FlareType.body.copyWith(
                              fontSize: 15,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                        Text(
                          status,
                          style: FlareType.metadata.copyWith(
                            color: color,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(width: 5),
                        PhosphorIcon(
                          PhosphorIconsRegular.caretRight,
                          size: 14,
                          color: palette.muted,
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Row(
                      children: <Widget>[
                        if (deployment.branch != null)
                          Text(
                            deployment.branch!,
                            style: FlareType.metadata.copyWith(
                              color: palette.textSecondary,
                            ),
                          ),
                        if (deployment.commit != null) ...<Widget>[
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 6),
                            child: Text(
                              '·',
                              style: FlareType.metadata.copyWith(
                                color: palette.muted,
                              ),
                            ),
                          ),
                          Text(
                            deployment.commit!.substring(
                              0,
                              deployment.commit!.length < 8
                                  ? deployment.commit!.length
                                  : 8,
                            ),
                            style: FlareType.mono.copyWith(
                              fontSize: 11,
                              color: palette.textSecondary,
                            ),
                          ),
                        ],
                      ],
                    ),
                    if (deployment.commitMessage != null) ...<Widget>[
                      const SizedBox(height: 4),
                      Text(
                        deployment.commitMessage!,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: FlareType.metadata.copyWith(
                          color: palette.muted,
                        ),
                      ),
                    ],
                    const SizedBox(height: 7),
                    Row(
                      children: <Widget>[
                        Text(
                          formatRelative(deployment.startedAt),
                          style: FlareType.metadata.copyWith(
                            color: palette.muted,
                          ),
                        ),
                        if (deployment.duration != null) ...<Widget>[
                          const SizedBox(width: 13),
                          Text(
                            formatDuration(deployment.duration),
                            style: FlareType.metadata.copyWith(
                              color: palette.muted,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
