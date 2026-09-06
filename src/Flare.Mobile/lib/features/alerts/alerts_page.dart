import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../core/utils/formatters.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_feedback.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';

final class AlertsPage extends ConsumerStatefulWidget {
  const AlertsPage({super.key});

  @override
  ConsumerState<AlertsPage> createState() => _AlertsPageState();
}

final class _AlertsPageState extends ConsumerState<AlertsPage> {
  bool _unreadOnly = false;
  String? _updating;

  Future<void> _toggle(AlertModel alert) async {
    setState(() => _updating = alert.id);
    try {
      await ref
          .read(apiClientProvider)
          .setAlertRead(alert.id, read: !alert.isRead);
      ref.invalidate(alertsProvider(false));
      ref.invalidate(alertsProvider(true));
    } on Object catch (error) {
      if (mounted) {
        FlareToast.show(context, error.toString(), tone: FlareToastTone.error);
      }
    } finally {
      if (mounted) setState(() => _updating = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final data = ref.watch(alertsProvider(_unreadOnly));
    final palette = context.flare;
    return FlareScaffold(
      eyebrow: 'OPERATIONS',
      title: 'Alerts',
      subtitle: 'Active incidents and recovery history',
      leading: const FlareBackButton(),
      body: data.when(
        loading: () => const FlareLoading(label: 'Reading alert history'),
        error: (error, _) => FlareErrorState(
          error: error,
          onRetry: () => ref.invalidate(alertsProvider(_unreadOnly)),
        ),
        data: (value) => Column(
          children: <Widget>[
            Padding(
              padding: const EdgeInsets.only(top: 6, bottom: 12),
              child: Row(
                children: <Widget>[
                  Expanded(
                    child: Text(
                      value.unreadCount == 0
                          ? 'All caught up'
                          : '${value.unreadCount} unread',
                      style: FlareType.body.copyWith(
                        color: value.unreadCount == 0
                            ? palette.success
                            : palette.textSecondary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  SizedBox(
                    width: 150,
                    child: FlareSegmentedControl<bool>(
                      value: _unreadOnly,
                      items: const <(bool, String)>[
                        (false, 'All'),
                        (true, 'Unread'),
                      ],
                      onChanged: (selected) =>
                          setState(() => _unreadOnly = selected),
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: value.items.isEmpty
                  ? FlareEmptyState(
                      title: _unreadOnly ? 'No unread alerts' : 'No alerts',
                      message: _unreadOnly
                          ? 'New infrastructure alerts will appear here.'
                          : 'Flare has not recorded any alert conditions.',
                      icon: PhosphorIconsRegular.bell,
                    )
                  : RefreshIndicator(
                      color: palette.accent,
                      backgroundColor: palette.surfaceHigh,
                      onRefresh: () async =>
                          ref.refresh(alertsProvider(_unreadOnly).future),
                      child: ListView.separated(
                        physics: const AlwaysScrollableScrollPhysics(),
                        padding: const EdgeInsets.only(bottom: 24),
                        itemCount: value.items.length,
                        separatorBuilder: (_, _) => const SizedBox(height: 8),
                        itemBuilder: (context, index) {
                          final alert = value.items[index];
                          return _AlertTile(
                            alert: alert,
                            updating: _updating == alert.id,
                            onTap: () => _toggle(alert),
                          );
                        },
                      ),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _AlertTile extends StatelessWidget {
  const _AlertTile({
    required this.alert,
    required this.updating,
    required this.onTap,
  });

  final AlertModel alert;
  final bool updating;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final color = switch (alert.status) {
      AlertStatus.recovered => palette.success,
      AlertStatus.active => switch (alert.severity) {
        AlertSeverity.critical => palette.danger,
        AlertSeverity.warning => palette.warning,
        AlertSeverity.info => palette.info,
      },
    };
    return FlareCard(
      onTap: updating ? null : onTap,
      padding: const EdgeInsets.all(14),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Container(
            width: 34,
            height: 34,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(FlareRadii.small),
            ),
            child: Center(
              child: updating
                  ? SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: color,
                      ),
                    )
                  : PhosphorIcon(
                      alert.status == AlertStatus.recovered
                          ? PhosphorIconsRegular.checkCircle
                          : PhosphorIconsRegular.warning,
                      size: 18,
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
                        alert.title,
                        style: FlareType.body.copyWith(
                          color: alert.isRead
                              ? palette.textSecondary
                              : palette.text,
                          fontWeight: alert.isRead
                              ? FontWeight.w500
                              : FontWeight.w700,
                        ),
                      ),
                    ),
                    Text(
                      formatRelative(alert.lastSeenAt),
                      style: FlareType.metadata.copyWith(color: palette.muted),
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Text(
                  alert.message,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: FlareType.metadata.copyWith(
                    color: palette.textSecondary,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  '${alert.source.toUpperCase()} · ${alert.status.name.toUpperCase()}${alert.occurrenceCount > 1 ? ' · ${alert.occurrenceCount}×' : ''} · Tap to mark ${alert.isRead ? 'unread' : 'read'}',
                  style: FlareType.metadata.copyWith(color: color),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
