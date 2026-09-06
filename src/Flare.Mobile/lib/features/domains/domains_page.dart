import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';

final class DomainsPage extends ConsumerWidget {
  const DomainsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final data = ref.watch(domainsProvider);
    final palette = context.flare;
    return FlareScaffold(
      eyebrow: 'INFRASTRUCTURE',
      title: 'Domains',
      subtitle: 'Cloudflare zones, records and reachability',
      leading: const FlareBackButton(),
      body: data.when(
        loading: () => const FlareLoading(label: 'Reading Cloudflare'),
        error: (error, _) => FlareErrorState(
          error: error,
          onRetry: () => ref.invalidate(domainsProvider),
        ),
        data: (value) {
          if (!value.status.configured) {
            return const FlareEmptyState(
              title: 'Cloudflare not configured',
              message:
                  'Configure CLOUDFLARE_API_TOKEN on the Flare server to inspect domains and DNS.',
              icon: PhosphorIconsRegular.cloudSlash,
            );
          }
          return RefreshIndicator(
            color: palette.accent,
            backgroundColor: palette.surfaceHigh,
            onRefresh: () {
              ref.invalidate(domainRecordsProvider);
              ref.invalidate(tunnelRoutesProvider);
              return ref.refresh(domainsProvider.future);
            },
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              children: <Widget>[
                _IntegrationSummary(
                  zoneCount: value.zones.length,
                  tunnelCount: value.tunnels.length,
                  tunnelsConfigured: value.status.tunnelsConfigured,
                ),
                const SizedBox(height: FlareSpace.lg),
                FlareSectionHeader(
                  title: 'Zones',
                  actionLabel: '${value.zones.length}',
                ),
                const SizedBox(height: FlareSpace.xs),
                if (value.zones.isEmpty)
                  const _InlineEmpty(message: 'No Cloudflare zones found.')
                else
                  ...value.zones.indexed.expand(
                    (entry) => <Widget>[
                      _ZonePanel(zone: entry.$2),
                      if (entry.$1 != value.zones.length - 1)
                        const FlareDivider(),
                    ],
                  ),
                const SizedBox(height: FlareSpace.lg),
                FlareSectionHeader(
                  title: 'Tunnels',
                  actionLabel: value.status.tunnelsConfigured
                      ? '${value.tunnels.length}'
                      : null,
                ),
                const SizedBox(height: FlareSpace.xs),
                if (!value.status.tunnelsConfigured)
                  const _InlineEmpty(
                    message:
                        'Tunnel access is not configured. Add CLOUDFLARE_ACCOUNT_ID to inspect public routes.',
                  )
                else if (value.tunnels.isEmpty)
                  const _InlineEmpty(message: 'No Cloudflare tunnels found.')
                else
                  ...value.tunnels.indexed.expand(
                    (entry) => <Widget>[
                      _TunnelPanel(tunnel: entry.$2),
                      if (entry.$1 != value.tunnels.length - 1)
                        const FlareDivider(),
                    ],
                  ),
                const SizedBox(height: FlareSpace.xl),
              ],
            ),
          );
        },
      ),
    );
  }
}

final class _IntegrationSummary extends StatelessWidget {
  const _IntegrationSummary({
    required this.zoneCount,
    required this.tunnelCount,
    required this.tunnelsConfigured,
  });
  final int zoneCount;
  final int tunnelCount;
  final bool tunnelsConfigured;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Padding(
      padding: const EdgeInsets.only(top: FlareSpace.xs),
      child: Row(
        children: <Widget>[
          Container(
            width: 38,
            height: 38,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: palette.accentSoft,
              borderRadius: BorderRadius.circular(FlareRadii.small),
            ),
            child: PhosphorIcon(
              PhosphorIconsRegular.cloud,
              size: 20,
              color: palette.accent,
            ),
          ),
          const SizedBox(width: FlareSpace.sm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  'Cloudflare',
                  style: FlareType.body.copyWith(
                    color: palette.text,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  '$zoneCount zones · ${tunnelsConfigured ? '$tunnelCount tunnels' : 'tunnels unavailable'}',
                  style: FlareType.metadata.copyWith(
                    color: palette.textSecondary,
                  ),
                ),
              ],
            ),
          ),
          const FlareStatusBadge(
            label: 'Connected',
            tone: FlareStatusTone.success,
          ),
        ],
      ),
    );
  }
}

final class _ZonePanel extends ConsumerStatefulWidget {
  const _ZonePanel({required this.zone});
  final DomainZoneModel zone;

  @override
  ConsumerState<_ZonePanel> createState() => _ZonePanelState();
}

final class _ZonePanelState extends ConsumerState<_ZonePanel> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final records = _expanded
        ? ref.watch(domainRecordsProvider(widget.zone.id))
        : null;
    final active = widget.zone.status.toLowerCase() == 'active';
    return Column(
      children: <Widget>[
        Semantics(
          button: true,
          expanded: _expanded,
          label: '${widget.zone.name}, ${widget.zone.status}',
          child: InkWell(
            onTap: () => setState(() => _expanded = !_expanded),
            borderRadius: BorderRadius.circular(FlareRadii.small),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 13, horizontal: 3),
              child: Row(
                children: <Widget>[
                  PhosphorIcon(
                    PhosphorIconsRegular.globeHemisphereWest,
                    size: 19,
                    color: active ? palette.success : palette.warning,
                  ),
                  const SizedBox(width: FlareSpace.sm),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text(
                          widget.zone.name,
                          style: FlareType.body.copyWith(
                            color: palette.text,
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          widget.zone.type.toUpperCase(),
                          style: FlareType.metadata.copyWith(
                            color: palette.muted,
                          ),
                        ),
                      ],
                    ),
                  ),
                  FlareStatusBadge(
                    label: widget.zone.status,
                    tone: active
                        ? FlareStatusTone.success
                        : FlareStatusTone.warning,
                    dot: false,
                  ),
                  const SizedBox(width: 7),
                  AnimatedRotation(
                    turns: _expanded ? 0.5 : 0,
                    duration: const Duration(milliseconds: 180),
                    child: PhosphorIcon(
                      PhosphorIconsRegular.caretDown,
                      size: 15,
                      color: palette.muted,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        AnimatedSwitcher(
          duration: const Duration(milliseconds: 180),
          child: _expanded
              ? _RecordsBody(key: const ValueKey('records'), records: records!)
              : const SizedBox.shrink(key: ValueKey('closed')),
        ),
      ],
    );
  }
}

final class _RecordsBody extends StatelessWidget {
  const _RecordsBody({required this.records, super.key});
  final AsyncValue<List<DNSRecordModel>> records;

  @override
  Widget build(BuildContext context) => records.when(
    loading: () => const _InlineProgress(label: 'Loading DNS records'),
    error: (error, _) =>
        const _InlineError(message: 'DNS records unavailable.'),
    data: (items) => items.isEmpty
        ? const _InlineEmpty(message: 'No DNS records found.')
        : Padding(
            padding: const EdgeInsets.only(left: FlareSpace.lg, bottom: 10),
            child: Column(
              children: items.indexed
                  .map(
                    (entry) => Column(
                      children: <Widget>[
                        _RecordRow(record: entry.$2),
                        if (entry.$1 != items.length - 1)
                          const FlareDivider(indent: 36),
                      ],
                    ),
                  )
                  .toList(growable: false),
            ),
          ),
  );
}

final class _RecordRow extends StatelessWidget {
  const _RecordRow({required this.record});
  final DNSRecordModel record;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final proxyLabel = record.proxied == true
        ? 'Proxied'
        : record.proxiable
        ? 'DNS only'
        : 'Direct';
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 11),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          SizedBox(
            width: 43,
            child: Text(
              record.type,
              style: FlareType.label.copyWith(color: palette.accent),
            ),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  record.name,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: FlareType.body.copyWith(
                    color: palette.text,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  record.target,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: FlareType.metadata.copyWith(
                    color: palette.textSecondary,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: FlareSpace.xs),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: <Widget>[
              Text(
                proxyLabel,
                style: FlareType.metadata.copyWith(
                  color: record.proxied == true
                      ? palette.warning
                      : palette.muted,
                ),
              ),
              Text(
                record.ttl == 1 ? 'Auto TTL' : '${record.ttl}s TTL',
                style: FlareType.metadata.copyWith(color: palette.muted),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

final class _TunnelPanel extends ConsumerStatefulWidget {
  const _TunnelPanel({required this.tunnel});
  final CloudflareTunnelModel tunnel;

  @override
  ConsumerState<_TunnelPanel> createState() => _TunnelPanelState();
}

final class _TunnelPanelState extends ConsumerState<_TunnelPanel> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final routes = _expanded
        ? ref.watch(tunnelRoutesProvider(widget.tunnel.id))
        : null;
    final healthy = const {
      'healthy',
      'active',
      'up',
    }.contains(widget.tunnel.status.toLowerCase());
    return Column(
      children: <Widget>[
        Semantics(
          button: true,
          expanded: _expanded,
          label: '${widget.tunnel.name}, ${widget.tunnel.status}',
          child: InkWell(
            onTap: () => setState(() => _expanded = !_expanded),
            borderRadius: BorderRadius.circular(FlareRadii.small),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 13, horizontal: 3),
              child: Row(
                children: <Widget>[
                  PhosphorIcon(
                    PhosphorIconsRegular.gitBranch,
                    size: 19,
                    color: healthy ? palette.success : palette.warning,
                  ),
                  const SizedBox(width: FlareSpace.sm),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Text(
                          widget.tunnel.name,
                          style: FlareType.body.copyWith(
                            color: palette.text,
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          widget.tunnel.configSource,
                          style: FlareType.metadata.copyWith(
                            color: palette.muted,
                          ),
                        ),
                      ],
                    ),
                  ),
                  FlareStatusBadge(
                    label: widget.tunnel.status,
                    tone: healthy
                        ? FlareStatusTone.success
                        : FlareStatusTone.warning,
                    dot: false,
                  ),
                  const SizedBox(width: 7),
                  AnimatedRotation(
                    turns: _expanded ? 0.5 : 0,
                    duration: const Duration(milliseconds: 180),
                    child: PhosphorIcon(
                      PhosphorIconsRegular.caretDown,
                      size: 15,
                      color: palette.muted,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        AnimatedSwitcher(
          duration: const Duration(milliseconds: 180),
          child: _expanded
              ? _RoutesBody(key: const ValueKey('routes'), routes: routes!)
              : const SizedBox.shrink(key: ValueKey('closed')),
        ),
      ],
    );
  }
}

final class _RoutesBody extends StatelessWidget {
  const _RoutesBody({required this.routes, super.key});
  final AsyncValue<List<TunnelRouteModel>> routes;

  @override
  Widget build(BuildContext context) => routes.when(
    loading: () => const _InlineProgress(label: 'Loading public routes'),
    error: (error, _) => const _InlineError(message: 'Routes unavailable.'),
    data: (items) => items.isEmpty
        ? const _InlineEmpty(message: 'No public routes found.')
        : Padding(
            padding: const EdgeInsets.only(left: FlareSpace.lg, bottom: 10),
            child: Column(
              children: items.indexed
                  .map(
                    (entry) => Column(
                      children: <Widget>[
                        _RouteRow(route: entry.$2),
                        if (entry.$1 != items.length - 1)
                          const FlareDivider(indent: 34),
                      ],
                    ),
                  )
                  .toList(growable: false),
            ),
          ),
  );
}

final class _RouteRow extends StatelessWidget {
  const _RouteRow({required this.route});
  final TunnelRouteModel route;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 11),
      child: Row(
        children: <Widget>[
          PhosphorIcon(
            PhosphorIconsRegular.arrowBendDownRight,
            size: 16,
            color: palette.accent,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  route.hostname,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: FlareType.body.copyWith(
                    color: palette.text,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                if (route.path != null && route.path!.isNotEmpty)
                  Text(
                    route.path!,
                    style: FlareType.metadata.copyWith(color: palette.muted),
                  ),
              ],
            ),
          ),
          FlareStatusBadge(label: route.originKind, dot: false),
        ],
      ),
    );
  }
}

final class _InlineProgress extends StatelessWidget {
  const _InlineProgress({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: FlareSpace.md),
    child: Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: <Widget>[
        SizedBox(
          width: 15,
          height: 15,
          child: CircularProgressIndicator(
            strokeWidth: 1.5,
            color: context.flare.accent,
          ),
        ),
        const SizedBox(width: 9),
        Text(label, style: FlareType.metadata),
      ],
    ),
  );
}

final class _InlineEmpty extends StatelessWidget {
  const _InlineEmpty({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: FlareSpace.md),
    child: Text(
      message,
      style: FlareType.metadata.copyWith(color: context.flare.textSecondary),
    ),
  );
}

final class _InlineError extends StatelessWidget {
  const _InlineError({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: FlareSpace.md),
    child: Row(
      children: <Widget>[
        PhosphorIcon(
          PhosphorIconsRegular.warningCircle,
          size: 16,
          color: context.flare.danger,
        ),
        const SizedBox(width: 8),
        Text(
          message,
          style: FlareType.metadata.copyWith(color: context.flare.danger),
        ),
      ],
    ),
  );
}
