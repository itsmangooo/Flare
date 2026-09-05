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
                child: ListView.separated(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.only(top: 6, bottom: 24),
                  itemCount: items.length,
                  separatorBuilder: (_, _) => const FlareDivider(indent: 17),
                  itemBuilder: (context, index) => _DeploymentTile(
                    deployment: items[index],
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
  const _DeploymentTile({required this.deployment, required this.onTap});
  final DeploymentModel deployment;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final (tone, color, status) = deploymentStatus(deployment.status, palette);
    return InkWell(
      borderRadius: BorderRadius.circular(FlareRadii.small),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 3),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Container(
              width: 34,
              height: 34,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.11),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Center(
                child: PhosphorIcon(
                  PhosphorIconsRegular.rocketLaunch,
                  size: 17,
                  color: color,
                ),
              ),
            ),
            const SizedBox(width: 11),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Row(
                    children: <Widget>[
                      Expanded(
                        child: Text(
                          deployment.resourceName,
                          style: FlareType.body.copyWith(
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      FlareStatusBadge(label: status, tone: tone, dot: false),
                    ],
                  ),
                  const SizedBox(height: 5),
                  Row(
                    children: <Widget>[
                      if (deployment.branch != null)
                        Text(deployment.branch!, style: FlareType.metadata),
                      if (deployment.commit != null) ...<Widget>[
                        const Padding(
                          padding: EdgeInsets.symmetric(horizontal: 6),
                          child: Text('·', style: FlareType.metadata),
                        ),
                        Text(
                          deployment.commit!.substring(
                            0,
                            deployment.commit!.length < 8
                                ? deployment.commit!.length
                                : 8,
                          ),
                          style: FlareType.mono.copyWith(fontSize: 11),
                        ),
                      ],
                    ],
                  ),
                  if (deployment.commitMessage != null) ...<Widget>[
                    const SizedBox(height: 5),
                    Text(
                      deployment.commitMessage!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: FlareType.metadata.copyWith(color: palette.muted),
                    ),
                  ],
                  const SizedBox(height: 8),
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
            Padding(
              padding: const EdgeInsets.only(top: 10, left: 5),
              child: PhosphorIcon(
                PhosphorIconsRegular.caretRight,
                size: 16,
                color: palette.muted,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
