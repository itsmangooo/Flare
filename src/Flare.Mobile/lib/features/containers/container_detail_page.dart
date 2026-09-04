import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
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

final class ContainerDetailPage extends ConsumerStatefulWidget {
  const ContainerDetailPage({required this.id, super.key});
  final String id;
  @override
  ConsumerState<ContainerDetailPage> createState() =>
      _ContainerDetailPageState();
}

final class _ContainerDetailPageState
    extends ConsumerState<ContainerDetailPage> {
  ContainerDetailModel? _detail;
  LogPageModel? _logs;
  Object? _error;
  bool _loading = true;
  bool _loadingLogs = false;
  String? _action;
  int _tail = 300;
  bool _following = false;
  Timer? _logTimer;

  String get _encodedId => Uri.encodeComponent(widget.id);

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _logTimer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    if (_action != null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = ref.read(apiClientProvider);
      final values = await Future.wait<Object>(<Future<Object>>[
        api.getJson('api/v1/containers/$_encodedId'),
        api.getJson(
          'api/v1/containers/$_encodedId/logs',
          query: <String, Object>{'tail': _tail},
        ),
      ]);
      if (!mounted) return;
      setState(() {
        _detail = ContainerDetailModel.fromJson(
          values[0] as Map<String, dynamic>,
        );
        _logs = LogPageModel.fromJson(values[1] as Map<String, dynamic>);
      });
    } on Object catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _loadLogs({DateTime? before}) async {
    if (_loadingLogs) return;
    setState(() => _loadingLogs = true);
    try {
      final query = <String, Object>{'tail': _tail};
      if (before != null) query['before'] = before.toUtc().toIso8601String();
      final json = await ref
          .read(apiClientProvider)
          .getJson('api/v1/containers/$_encodedId/logs', query: query);
      if (mounted) setState(() => _logs = LogPageModel.fromJson(json));
    } on FlareApiException catch (error) {
      if (mounted) {
        FlareToast.show(context, error.message, tone: FlareToastTone.error);
      }
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Container logs could not be refreshed.',
          tone: FlareToastTone.error,
        );
      }
    } finally {
      if (mounted) setState(() => _loadingLogs = false);
    }
  }

  void _setFollowing(bool value) {
    _logTimer?.cancel();
    setState(() => _following = value);
    if (value) {
      _loadLogs();
      _logTimer = Timer.periodic(
        const Duration(seconds: 5),
        (_) => _loadLogs(),
      );
    }
  }

  Future<void> _runAction(String action) async {
    final detail = _detail;
    if (detail == null || _action != null) return;
    final label = '${action[0].toUpperCase()}${action.substring(1)}';
    final confirmed = await FlareConfirmSheet.show(
      context,
      title: '$label container?',
      subject: detail.name,
      message: action == 'stop'
          ? 'The service will become unavailable until it is started again.'
          : 'The service may be unavailable briefly.',
      confirmLabel: label,
      destructive: action == 'stop',
    );
    if (!confirmed || !mounted || _action != null) return;
    setState(() => _action = action);
    await safeHaptic();
    try {
      await ref
          .read(apiClientProvider)
          .postJson('api/v1/containers/$_encodedId/$action');
      await Future<void>.delayed(const Duration(milliseconds: 350));
      final json = await ref
          .read(apiClientProvider)
          .getJson('api/v1/containers/$_encodedId');
      if (!mounted) return;
      setState(() => _detail = ContainerDetailModel.fromJson(json));
      ref.invalidate(containersProvider);
      FlareToast.show(
        context,
        '$label request accepted.',
        tone: FlareToastTone.success,
      );
    } on FlareApiException catch (error) {
      if (mounted) {
        FlareToast.show(context, error.message, tone: FlareToastTone.error);
      }
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Flare could not $action this container.',
          tone: FlareToastTone.error,
        );
      }
    } finally {
      if (mounted) setState(() => _action = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final detail = _detail;
    return FlareScaffold(
      eyebrow: 'CONTAINER',
      title: detail?.name ?? 'Container',
      leading: const FlareBackButton(),
      actions: <Widget>[
        if (detail != null)
          FlareStatusBadge(
            label: detail.state.name,
            tone: containerTone(detail.state, detail.health),
          ),
      ],
      body: _loading && detail == null
          ? const FlareLoading(label: 'Inspecting container')
          : _error != null && detail == null
          ? FlareErrorState(error: _error!, onRetry: _load)
          : detail == null
          ? const FlareEmptyState(
              title: 'Container unavailable',
              message: 'Docker no longer reports this container.',
            )
          : RefreshIndicator(
              color: FlareColors.accent,
              backgroundColor: FlareColors.surfaceHigh,
              onRefresh: _load,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.only(top: 6, bottom: 26),
                children: <Widget>[
                  _ActionBar(action: _action, onAction: _runAction),
                  const SizedBox(height: 24),
                  const FlareSectionHeader(title: 'Overview'),
                  const SizedBox(height: 10),
                  _Metrics(detail: detail),
                  const SizedBox(height: 10),
                  _InfoRows(detail: detail),
                  const SizedBox(height: 24),
                  _LogsHeader(
                    tail: _tail,
                    following: _following,
                    loading: _loadingLogs,
                    onTail: (value) {
                      setState(() => _tail = value);
                      _loadLogs();
                    },
                    onFollow: _setFollowing,
                    onRefresh: _loadLogs,
                  ),
                  const SizedBox(height: 10),
                  _LogConsole(logs: _logs),
                  if (_logs case LogPageModel(
                    truncated: true,
                    oldestTimestamp: final oldest?,
                  )) ...<Widget>[
                    const SizedBox(height: 10),
                    FlareButton(
                      label: 'Load older',
                      icon: PhosphorIconsRegular.clockCounterClockwise,
                      onPressed: _loadingLogs
                          ? null
                          : () => _loadLogs(
                              before: oldest.subtract(
                                const Duration(microseconds: 1),
                              ),
                            ),
                    ),
                  ],
                  if (detail.ports.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 24),
                    const FlareSectionHeader(title: 'Ports'),
                    const SizedBox(height: 8),
                    FlareCard(
                      child: Column(
                        children: detail.ports
                            .map(
                              (port) => Padding(
                                padding: const EdgeInsets.symmetric(
                                  vertical: 5,
                                ),
                                child: Row(
                                  children: <Widget>[
                                    Expanded(
                                      child: Text(
                                        '${port.privatePort}/${port.protocol}',
                                        style: FlareType.mono,
                                      ),
                                    ),
                                    Text(
                                      port.publicPort == null
                                          ? 'internal'
                                          : '${port.hostIp ?? '*'}:${port.publicPort}',
                                      style: FlareType.metadata,
                                    ),
                                  ],
                                ),
                              ),
                            )
                            .toList(growable: false),
                      ),
                    ),
                  ],
                  if (detail.labels.isNotEmpty) ...<Widget>[
                    const SizedBox(height: 24),
                    const FlareSectionHeader(title: 'Labels'),
                    const SizedBox(height: 8),
                    FlareCard(
                      child: SelectableText(
                        detail.labels.entries
                            .map((entry) => '${entry.key}=${entry.value}')
                            .join('\n'),
                        style: FlareType.mono.copyWith(
                          color: FlareColors.muted,
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
    );
  }
}

final class _ActionBar extends StatelessWidget {
  const _ActionBar({required this.action, required this.onAction});
  final String? action;
  final ValueChanged<String> onAction;
  @override
  Widget build(BuildContext context) => Row(
    children: <Widget>[
      Expanded(
        child: FlareButton(
          label: 'Start',
          icon: PhosphorIconsRegular.play,
          loading: action == 'start',
          onPressed: action == null ? () => onAction('start') : null,
        ),
      ),
      const SizedBox(width: 8),
      Expanded(
        child: FlareButton(
          label: 'Restart',
          icon: PhosphorIconsRegular.arrowClockwise,
          loading: action == 'restart',
          onPressed: action == null ? () => onAction('restart') : null,
        ),
      ),
      const SizedBox(width: 8),
      Expanded(
        child: FlareButton(
          label: 'Stop',
          icon: PhosphorIconsRegular.stop,
          tone: FlareButtonTone.danger,
          loading: action == 'stop',
          onPressed: action == null ? () => onAction('stop') : null,
        ),
      ),
    ],
  );
}

final class _Metrics extends StatelessWidget {
  const _Metrics({required this.detail});
  final ContainerDetailModel detail;
  @override
  Widget build(BuildContext context) => Row(
    children: <Widget>[
      Expanded(
        child: _CompactMetric(
          label: 'CPU',
          value: formatPercent(detail.cpuPercent, decimals: 1),
          usage: detail.cpuPercent,
        ),
      ),
      const SizedBox(width: 10),
      Expanded(
        child: _CompactMetric(
          label: 'MEMORY',
          value: formatBytes(detail.memoryBytes),
          usage:
              detail.memoryBytes != null &&
                  detail.memoryLimitBytes != null &&
                  detail.memoryLimitBytes! > 0
              ? detail.memoryBytes! / detail.memoryLimitBytes! * 100
              : null,
          color: FlareColors.info,
        ),
      ),
    ],
  );
}

final class _CompactMetric extends StatelessWidget {
  const _CompactMetric({
    required this.label,
    required this.value,
    required this.usage,
    this.color = FlareColors.accent,
  });
  final String label;
  final String value;
  final double? usage;
  final Color color;
  @override
  Widget build(BuildContext context) => FlareCard(
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(label, style: FlareType.label),
        const SizedBox(height: 10),
        Text(value, style: FlareType.metric.copyWith(fontSize: 23)),
        const SizedBox(height: 13),
        FlareUsageBar(value: usage, color: color),
      ],
    ),
  );
}

final class _InfoRows extends StatelessWidget {
  const _InfoRows({required this.detail});
  final ContainerDetailModel detail;
  @override
  Widget build(BuildContext context) => FlareCard(
    padding: const EdgeInsets.symmetric(horizontal: 15),
    child: Column(
      children: <Widget>[
        _InfoRow(label: 'Image', value: detail.image),
        const FlareDivider(),
        _InfoRow(label: 'Container ID', value: detail.shortId, mono: true),
        const FlareDivider(),
        _InfoRow(
          label: 'Uptime',
          value: detail.startedAt == null
              ? 'Not running'
              : formatDuration(DateTime.now().difference(detail.startedAt!)),
        ),
        const FlareDivider(),
        _InfoRow(label: 'Restarts', value: detail.restartCount.toString()),
        const FlareDivider(),
        _InfoRow(
          label: 'Network RX',
          value: formatBytes(detail.networkReceiveBytes),
        ),
        const FlareDivider(),
        _InfoRow(
          label: 'Network TX',
          value: formatBytes(detail.networkTransmitBytes),
        ),
      ],
    ),
  );
}

final class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value, this.mono = false});
  final String label;
  final String value;
  final bool mono;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        SizedBox(
          width: 92,
          child: Text(
            label,
            style: FlareType.metadata.copyWith(color: FlareColors.muted),
          ),
        ),
        Expanded(
          child: Text(
            value,
            textAlign: TextAlign.right,
            style: mono
                ? FlareType.mono
                : FlareType.metadata.copyWith(color: FlareColors.text),
          ),
        ),
      ],
    ),
  );
}

final class _LogsHeader extends StatelessWidget {
  const _LogsHeader({
    required this.tail,
    required this.following,
    required this.loading,
    required this.onTail,
    required this.onFollow,
    required this.onRefresh,
  });
  final int tail;
  final bool following;
  final bool loading;
  final ValueChanged<int> onTail;
  final ValueChanged<bool> onFollow;
  final VoidCallback onRefresh;
  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: <Widget>[
      Row(
        children: <Widget>[
          const Expanded(child: Text('Logs', style: FlareType.title)),
          FlareIconButton(
            icon: PhosphorIconsRegular.arrowClockwise,
            semanticLabel: 'Refresh logs',
            onPressed: loading ? null : onRefresh,
          ),
        ],
      ),
      const SizedBox(height: 9),
      Wrap(
        spacing: 7,
        runSpacing: 7,
        children: <Widget>[
          for (final value in <int>[100, 300, 1000])
            _LogOption(
              label: '$value',
              selected: tail == value,
              onTap: () => onTail(value),
            ),
          _LogOption(
            label: following ? 'Live on' : 'Live off',
            selected: following,
            onTap: () => onFollow(!following),
            icon: PhosphorIconsRegular.broadcast,
          ),
        ],
      ),
    ],
  );
}

final class _LogOption extends StatelessWidget {
  const _LogOption({
    required this.label,
    required this.selected,
    required this.onTap,
    this.icon,
  });
  final String label;
  final bool selected;
  final VoidCallback onTap;
  final PhosphorIconData? icon;
  @override
  Widget build(BuildContext context) => InkWell(
    onTap: onTap,
    borderRadius: BorderRadius.circular(6),
    child: Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: selected ? FlareColors.accentSoft : FlareColors.surface,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(
          color: selected ? const Color(0x552F81F7) : FlareColors.border,
        ),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: <Widget>[
          if (icon != null) ...<Widget>[
            PhosphorIcon(
              icon!,
              size: 13,
              color: selected ? FlareColors.accent : FlareColors.muted,
            ),
            const SizedBox(width: 5),
          ],
          Text(
            label,
            style: FlareType.metadata.copyWith(
              color: selected ? FlareColors.accent : FlareColors.textSecondary,
              fontSize: 11,
            ),
          ),
        ],
      ),
    ),
  );
}

final class _LogConsole extends StatelessWidget {
  const _LogConsole({required this.logs});
  final LogPageModel? logs;
  @override
  Widget build(BuildContext context) => Container(
    constraints: const BoxConstraints(minHeight: 180, maxHeight: 390),
    width: double.infinity,
    padding: const EdgeInsets.all(13),
    decoration: BoxDecoration(
      color: const Color(0xFF05070A),
      borderRadius: BorderRadius.circular(FlareRadii.small),
      border: Border.all(color: FlareColors.borderStrong),
    ),
    child: SingleChildScrollView(
      child: SelectableText(
        logs == null
            ? 'Loading log output…'
            : logs!.lines.isEmpty
            ? 'No log output.'
            : logs!.lines.join('\n'),
        style: FlareType.mono,
      ),
    ),
  );
}
