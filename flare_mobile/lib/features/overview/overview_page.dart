import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../core/utils/formatters.dart';
import '../../design/components/flare_chart.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_feedback.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';
import '../../design/components/flare_tiles.dart';

final class OverviewPage extends ConsumerWidget {
  const OverviewPage({super.key});

  Future<void> _refresh(BuildContext context, WidgetRef ref) async {
    try {
      await ref.read(telemetryServiceProvider).refresh();
    } on FlareApiException catch (error) {
      if (context.mounted) {
        FlareToast.show(context, error.message, tone: FlareToastTone.error);
      }
    } on Object {
      if (context.mounted) {
        FlareToast.show(
          context,
          'Flare server unreachable.',
          tone: FlareToastTone.error,
        );
      }
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final snapshot = ref.watch(overviewProvider);
    final palette = context.flare;
    return FlareScaffold(
      title: 'Overview',
      subtitle: 'Your homelab at a glance',
      actions: <Widget>[
        FlareIconButton(
          icon: PhosphorIconsRegular.bell,
          semanticLabel: 'Alerts',
          onPressed: () => context.push('/alerts'),
        ),
        FlareIconButton(
          icon: PhosphorIconsRegular.globeHemisphereWest,
          semanticLabel: 'Domains',
          onPressed: () => context.push('/domains'),
        ),
        FlareIconButton(
          icon: PhosphorIconsRegular.userCircle,
          semanticLabel: 'Settings',
          onPressed: () => context.push('/settings'),
        ),
      ],
      body: snapshot.when(
        loading: () => const FlareLoading(label: 'Reading homelab telemetry'),
        error: (error, _) => FlareErrorState(
          error: error,
          onRetry: () => ref.invalidate(overviewProvider),
        ),
        data: (data) => RefreshIndicator(
          color: palette.accent,
          backgroundColor: palette.surfaceHigh,
          onRefresh: () => _refresh(context, ref),
          child: CustomScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            slivers: <Widget>[
              SliverToBoxAdapter(child: _HostHeader(data: data)),
              SliverToBoxAdapter(child: _MetricsGrid(data: data)),
              SliverToBoxAdapter(child: _ContainerSummary(data: data)),
              SliverToBoxAdapter(
                child: Padding(
                  padding: const EdgeInsets.only(top: FlareSpace.lg),
                  child: FlareSectionHeader(
                    title: 'Recent activity',
                    actionLabel: 'View all',
                    onAction: () => context.go('/activity'),
                  ),
                ),
              ),
              if (data.recentActivity.isEmpty)
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 26),
                    child: Text(
                      'No recent activity.',
                      style: FlareType.metadata.copyWith(
                        color: palette.textSecondary,
                      ),
                    ),
                  ),
                )
              else
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.only(top: 10),
                    child: FlareGroupedSurface(
                      padding: const EdgeInsets.fromLTRB(14, 16, 14, 0),
                      child: Column(
                        children: List<Widget>.generate(
                          data.recentActivity.take(4).length,
                          (index) => FlareActivityTile(
                            event: data.recentActivity[index],
                            last:
                                index == data.recentActivity.take(4).length - 1,
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              const SliverToBoxAdapter(child: SizedBox(height: 24)),
            ],
          ),
        ),
      ),
    );
  }
}

final class _HostHeader extends StatelessWidget {
  const _HostHeader({required this.data});
  final OverviewModel data;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final (label, tone) = switch (data.freshness) {
      DataFreshness.live => ('Online', FlareStatusTone.success),
      DataFreshness.reconnecting => ('Reconnecting', FlareStatusTone.warning),
      DataFreshness.stale => ('Stale', FlareStatusTone.warning),
      DataFreshness.offline => ('Offline', FlareStatusTone.danger),
    };
    final statusColor = switch (tone) {
      FlareStatusTone.success => palette.success,
      FlareStatusTone.warning => palette.warning,
      FlareStatusTone.danger => palette.danger,
      _ => palette.info,
    };
    return Padding(
      padding: const EdgeInsets.fromLTRB(0, 6, 0, 24),
      child: FlareCard(
        padding: const EdgeInsets.all(18),
        child: Row(
          children: <Widget>[
            Container(
              width: 50,
              height: 50,
              decoration: BoxDecoration(
                color: statusColor.withValues(alpha: 0.12),
                shape: BoxShape.circle,
              ),
              alignment: Alignment.center,
              child: PhosphorIcon(
                PhosphorIconsRegular.houseLine,
                size: 23,
                color: statusColor,
              ),
            ),
            const SizedBox(width: 15),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  Text(
                    data.host.hostName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: FlareType.title.copyWith(color: palette.text),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Uptime ${formatDuration(data.host.uptime)}',
                    style: FlareType.metadata.copyWith(
                      color: palette.textSecondary,
                    ),
                  ),
                ],
              ),
            ),
            FlareStatusBadge(label: label, tone: tone),
          ],
        ),
      ),
    );
  }
}

final class _MetricsGrid extends StatelessWidget {
  const _MetricsGrid({required this.data});
  final OverviewModel data;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final memorySecondary =
        '${formatBytes(data.host.memoryUsedBytes)} / ${formatBytes(data.host.memoryTotalBytes)}';
    final diskSecondary =
        '${formatBytes(data.host.diskUsedBytes)} / ${formatBytes(data.host.diskTotalBytes)}';
    return LayoutBuilder(
      builder: (context, constraints) {
        final cardWidth = (constraints.maxWidth - 10) / 2;
        return SizedBox(
          width: double.infinity,
          height: 160,
          child: ListView(
            scrollDirection: Axis.horizontal,
            physics: const BouncingScrollPhysics(),
            clipBehavior: Clip.none,
            children: <Widget>[
              SizedBox(
                width: cardWidth,
                child: FlareMetricCard(
                  title: 'CPU',
                  value: formatPercent(data.host.cpuPercent),
                  secondary: data.host.loadAverage == null
                      ? 'Load unavailable'
                      : 'Load ${data.host.loadAverage!.toStringAsFixed(2)}',
                  chart: data.history
                      .map((point) => point.cpuPercent)
                      .toList(growable: false),
                ),
              ),
              const SizedBox(width: 10),
              SizedBox(
                width: cardWidth,
                child: FlareMetricCard(
                  title: 'Memory',
                  value: formatPercent(data.host.memoryPercent),
                  secondary: memorySecondary,
                  chart: data.history
                      .map((point) => point.memoryPercent)
                      .toList(growable: false),
                  accent: palette.info,
                ),
              ),
              const SizedBox(width: 10),
              SizedBox(
                width: cardWidth,
                child: FlareMetricCard(
                  title: 'Disk',
                  value: formatPercent(data.host.diskPercent),
                  secondary: diskSecondary,
                  usage: data.host.diskPercent,
                  accent: palette.warning,
                ),
              ),
              const SizedBox(width: 10),
              SizedBox(
                width: cardWidth,
                child: _NetworkCard(
                  receive: data.host.networkReceiveBytesPerSecond,
                  transmit: data.host.networkTransmitBytesPerSecond,
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

final class _NetworkCard extends StatelessWidget {
  const _NetworkCard({required this.receive, required this.transmit});
  final double? receive;
  final double? transmit;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return FlareCard(
      padding: const EdgeInsets.all(14),
      child: SizedBox(
        height: 126,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            Text(
              'NETWORK',
              style: FlareType.label.copyWith(color: palette.muted),
            ),
            const Spacer(),
            _NetworkLine(
              icon: PhosphorIconsRegular.arrowDown,
              label: 'Down',
              value: formatBytes(receive, perSecond: true),
              color: palette.success,
            ),
            const SizedBox(height: 12),
            _NetworkLine(
              icon: PhosphorIconsRegular.arrowUp,
              label: 'Up',
              value: formatBytes(transmit, perSecond: true),
              color: palette.info,
            ),
            const Spacer(),
          ],
        ),
      ),
    );
  }
}

final class _NetworkLine extends StatelessWidget {
  const _NetworkLine({
    required this.icon,
    required this.label,
    required this.value,
    required this.color,
  });
  final PhosphorIconData icon;
  final String label;
  final String value;
  final Color color;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Row(
      children: <Widget>[
        PhosphorIcon(icon, size: 15, color: color),
        const SizedBox(width: 7),
        Expanded(
          child: Text(
            label,
            style: FlareType.metadata.copyWith(color: palette.muted),
          ),
        ),
        Text(
          value,
          style: FlareType.metadata.copyWith(
            color: palette.text,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }
}

final class _ContainerSummary extends StatelessWidget {
  const _ContainerSummary({required this.data});
  final OverviewModel data;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Padding(
      padding: const EdgeInsets.only(top: 26),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          const FlareSectionHeader(title: 'Containers'),
          const SizedBox(height: 9),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(
              horizontal: FlareSpace.md,
              vertical: FlareSpace.sm,
            ),
            child: Row(
              children: <Widget>[
                _SummaryValue(
                  label: 'Running',
                  value: data.containers.running,
                  color: palette.success,
                ),
                const _SummaryDivider(),
                _SummaryValue(
                  label: 'Stopped',
                  value: data.containers.stopped,
                  color: palette.textSecondary,
                ),
                const _SummaryDivider(),
                _SummaryValue(
                  label: 'Unhealthy',
                  value: data.containers.unhealthy,
                  color: palette.danger,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

final class _SummaryValue extends StatelessWidget {
  const _SummaryValue({
    required this.label,
    required this.value,
    required this.color,
  });
  final String label;
  final int? value;
  final Color color;
  @override
  Widget build(BuildContext context) => Expanded(
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          value?.toString() ?? '—',
          style: FlareType.metricCompact.copyWith(color: color),
        ),
        const SizedBox(height: 3),
        Text(
          label,
          style: FlareType.metadata.copyWith(color: context.flare.muted),
        ),
      ],
    ),
  );
}

final class _SummaryDivider extends StatelessWidget {
  const _SummaryDivider();
  @override
  Widget build(BuildContext context) => Container(
    width: 1,
    height: 34,
    margin: const EdgeInsets.symmetric(horizontal: 10),
    color: context.flare.border,
  );
}
