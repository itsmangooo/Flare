import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_feedback.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';

enum _ResourceTab { applications, services, servers }

final class CoolifyResourcesPage extends ConsumerStatefulWidget {
  const CoolifyResourcesPage({super.key});
  @override
  ConsumerState<CoolifyResourcesPage> createState() =>
      _CoolifyResourcesPageState();
}

final class _CoolifyResourcesPageState
    extends ConsumerState<CoolifyResourcesPage> {
  _ResourceTab _tab = _ResourceTab.applications;
  String? _operation;

  Future<void> _run({
    required String path,
    required String label,
    required String subject,
    bool destructive = false,
  }) async {
    if (_operation != null) return;
    final confirmed = await FlareConfirmSheet.show(
      context,
      title: '$label resource?',
      subject: subject,
      message:
          'This operation will be sent to the configured Coolify instance.',
      confirmLabel: label,
      destructive: destructive,
    );
    if (!confirmed || !mounted || _operation != null) return;
    setState(() => _operation = '$path:$label');
    try {
      await ref.read(apiClientProvider).postJson(path);
      if (!mounted) return;
      ref.invalidate(coolifyProvider);
      await safeHaptic();
      if (!mounted) return;
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
          'Coolify could not complete this operation.',
          tone: FlareToastTone.error,
        );
      }
    } finally {
      if (mounted) setState(() => _operation = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final data = ref.watch(coolifyProvider);
    final palette = context.flare;
    return FlareScaffold(
      eyebrow: 'COOLIFY',
      title: 'Resources',
      subtitle: 'Applications and services managed by Coolify',
      leading: const FlareBackButton(),
      body: Column(
        children: <Widget>[
          _Tabs(
            selected: _tab,
            onSelected: (tab) => setState(() => _tab = tab),
          ),
          const SizedBox(height: FlareSpace.sm),
          Expanded(
            child: data.when(
              loading: () =>
                  const FlareLoading(label: 'Reading Coolify resources'),
              error: (error, _) => FlareErrorState(
                error: error,
                onRetry: () => ref.invalidate(coolifyProvider),
              ),
              data: (values) => RefreshIndicator(
                color: palette.accent,
                backgroundColor: palette.surfaceHigh,
                onRefresh: () async => ref.refresh(coolifyProvider.future),
                child: switch (_tab) {
                  _ResourceTab.applications => _ApplicationList(
                    items: values.applications,
                    operation: _operation,
                    run: _run,
                  ),
                  _ResourceTab.services => _ServiceList(
                    items: values.services,
                    operation: _operation,
                    run: _run,
                  ),
                  _ResourceTab.servers => _ServerList(items: values.servers),
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _Tabs extends StatelessWidget {
  const _Tabs({required this.selected, required this.onSelected});
  final _ResourceTab selected;
  final ValueChanged<_ResourceTab> onSelected;
  @override
  Widget build(BuildContext context) => FlareSegmentedControl<_ResourceTab>(
    value: selected,
    items: _ResourceTab.values
        .map((tab) => (tab, tab.name[0].toUpperCase() + tab.name.substring(1)))
        .toList(growable: false),
    onChanged: onSelected,
  );
}

typedef _RunResource =
    Future<void> Function({
      required String path,
      required String label,
      required String subject,
      required bool destructive,
    });

final class _ApplicationList extends StatelessWidget {
  const _ApplicationList({
    required this.items,
    required this.operation,
    required this.run,
  });
  final List<CoolifyApplicationModel> items;
  final String? operation;
  final _RunResource run;
  @override
  Widget build(BuildContext context) => items.isEmpty
      ? ListView(
          children: <Widget>[
            SizedBox(height: 140),
            FlareEmptyState(
              title: 'No applications',
              message: 'Coolify returned no applications.',
            ),
          ],
        )
      : ListView.separated(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: items.length,
          separatorBuilder: (_, _) => const SizedBox(height: 9),
          itemBuilder: (context, index) {
            final app = items[index];
            final base =
                'api/v1/coolify/applications/${Uri.encodeComponent(app.uuid)}';
            return FlareGroupedSurface(
              padding: const EdgeInsets.symmetric(horizontal: 14),
              child: _ResourceRow(
                name: app.name,
                subtitle: app.gitBranch ?? app.fqdn ?? 'Application',
                status: app.status,
                actions: <Widget>[
                  FlareButton(
                    label: 'Start',
                    compact: true,
                    loading: operation == '$base/start:Start',
                    onPressed: operation == null
                        ? () => run(
                            path: '$base/start',
                            label: 'Start',
                            subject: app.name,
                            destructive: false,
                          )
                        : null,
                  ),
                  FlareButton(
                    label: 'Restart',
                    compact: true,
                    loading: operation == '$base/restart:Restart',
                    onPressed: operation == null
                        ? () => run(
                            path: '$base/restart',
                            label: 'Restart',
                            subject: app.name,
                            destructive: false,
                          )
                        : null,
                  ),
                  FlareButton(
                    label: 'Stop',
                    compact: true,
                    tone: FlareButtonTone.danger,
                    loading: operation == '$base/stop:Stop',
                    onPressed: operation == null
                        ? () => run(
                            path: '$base/stop',
                            label: 'Stop',
                            subject: app.name,
                            destructive: true,
                          )
                        : null,
                  ),
                  FlareButton(
                    label: 'Redeploy',
                    compact: true,
                    tone: FlareButtonTone.primary,
                    loading: operation == '$base/redeploy:Redeploy',
                    onPressed: operation == null
                        ? () => run(
                            path: '$base/redeploy',
                            label: 'Redeploy',
                            subject: app.name,
                            destructive: false,
                          )
                        : null,
                  ),
                ],
              ),
            );
          },
        );
}

final class _ServiceList extends StatelessWidget {
  const _ServiceList({
    required this.items,
    required this.operation,
    required this.run,
  });
  final List<CoolifyServiceModel> items;
  final String? operation;
  final _RunResource run;
  @override
  Widget build(BuildContext context) => items.isEmpty
      ? ListView(
          children: <Widget>[
            SizedBox(height: 140),
            FlareEmptyState(
              title: 'No services',
              message: 'Coolify returned no services.',
            ),
          ],
        )
      : ListView.separated(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: items.length,
          separatorBuilder: (_, _) => const SizedBox(height: 9),
          itemBuilder: (context, index) {
            final service = items[index];
            final path =
                'api/v1/coolify/services/${Uri.encodeComponent(service.uuid)}/restart';
            return FlareGroupedSurface(
              padding: const EdgeInsets.symmetric(horizontal: 14),
              child: _ResourceRow(
                name: service.name,
                subtitle: service.description ?? 'Service',
                status: service.status,
                actions: <Widget>[
                  FlareButton(
                    label: 'Restart',
                    compact: true,
                    tone: FlareButtonTone.primary,
                    loading: operation == '$path:Restart',
                    onPressed: operation == null
                        ? () => run(
                            path: path,
                            label: 'Restart',
                            subject: service.name,
                            destructive: false,
                          )
                        : null,
                  ),
                ],
              ),
            );
          },
        );
}

final class _ServerList extends StatelessWidget {
  const _ServerList({required this.items});
  final List<CoolifyServerModel> items;
  @override
  Widget build(BuildContext context) => items.isEmpty
      ? ListView(
          children: <Widget>[
            SizedBox(height: 140),
            FlareEmptyState(
              title: 'No servers',
              message: 'Coolify returned no servers.',
            ),
          ],
        )
      : ListView.separated(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: items.length,
          separatorBuilder: (_, _) => const SizedBox(height: 9),
          itemBuilder: (context, index) =>
              _ServerExpansion(server: items[index]),
        );
}

final class _ServerExpansion extends ConsumerStatefulWidget {
  const _ServerExpansion({required this.server});
  final CoolifyServerModel server;
  @override
  ConsumerState<_ServerExpansion> createState() => _ServerExpansionState();
}

final class _ServerExpansionState extends ConsumerState<_ServerExpansion> {
  bool expanded = false;
  Future<List<CoolifyResourceModel>>? resources;
  void _toggle() {
    setState(() {
      expanded = !expanded;
      if (expanded) {
        resources ??= ref
            .read(apiClientProvider)
            .getList(
              'api/v1/coolify/servers/${Uri.encodeComponent(widget.server.uuid)}/resources',
            )
            .then(
              (items) => items
                  .map(
                    (item) => CoolifyResourceModel.fromJson(
                      (item as Map).map(
                        (key, value) => MapEntry(key.toString(), value),
                      ),
                    ),
                  )
                  .toList(growable: false),
            );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return FlareCard(
      onTap: _toggle,
      child: Column(
        children: <Widget>[
          Row(
            children: <Widget>[
              PhosphorIcon(
                PhosphorIconsRegular.hardDrives,
                size: 19,
                color: widget.server.isReachable == true
                    ? palette.success
                    : palette.danger,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  widget.server.name,
                  style: FlareType.body.copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              FlareStatusBadge(
                label: widget.server.isReachable == true
                    ? 'Reachable'
                    : 'Offline',
                tone: widget.server.isReachable == true
                    ? FlareStatusTone.success
                    : FlareStatusTone.danger,
                dot: false,
              ),
              const SizedBox(width: 7),
              AnimatedRotation(
                turns: expanded ? 0.5 : 0,
                duration: const Duration(milliseconds: 180),
                child: PhosphorIcon(
                  PhosphorIconsRegular.caretDown,
                  size: 15,
                  color: palette.muted,
                ),
              ),
            ],
          ),
          if (expanded) ...<Widget>[
            const SizedBox(height: FlareSpace.sm),
            const FlareDivider(),
            const SizedBox(height: 10),
            FutureBuilder<List<CoolifyResourceModel>>(
              future: resources,
              builder: (context, snapshot) {
                if (snapshot.hasError) {
                  return Text(
                    'Resources unavailable.',
                    style: FlareType.metadata.copyWith(color: palette.danger),
                  );
                }
                if (!snapshot.hasData) {
                  return LinearProgressIndicator(
                    minHeight: 2,
                    color: palette.accent,
                    backgroundColor: palette.border,
                  );
                }
                if (snapshot.data!.isEmpty) {
                  return const Text(
                    'No resources on this server.',
                    style: FlareType.metadata,
                  );
                }
                return Column(
                  children: snapshot.data!
                      .map(
                        (resource) => Padding(
                          padding: const EdgeInsets.symmetric(vertical: 6),
                          child: Row(
                            children: <Widget>[
                              Expanded(
                                child: Text(
                                  resource.name,
                                  style: FlareType.metadata.copyWith(
                                    color: palette.text,
                                  ),
                                ),
                              ),
                              Text(
                                resource.type,
                                style: FlareType.metadata.copyWith(
                                  color: palette.muted,
                                ),
                              ),
                              const SizedBox(width: 8),
                              Text(
                                resource.status ?? 'unknown',
                                style: FlareType.metadata,
                              ),
                            ],
                          ),
                        ),
                      )
                      .toList(growable: false),
                );
              },
            ),
          ],
        ],
      ),
    );
  }
}

final class _ResourceRow extends StatelessWidget {
  const _ResourceRow({
    required this.name,
    required this.subtitle,
    required this.status,
    required this.actions,
  });
  final String name;
  final String subtitle;
  final String? status;
  final List<Widget> actions;
  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 3),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            children: <Widget>[
              Expanded(
                child: Text(
                  name,
                  style: FlareType.body.copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              FlareStatusBadge(
                label: status ?? 'Unknown',
                tone: (status ?? '').toLowerCase().contains('running')
                    ? FlareStatusTone.success
                    : FlareStatusTone.neutral,
                dot: false,
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            subtitle,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: FlareType.metadata.copyWith(color: palette.muted),
          ),
          const SizedBox(height: 11),
          Wrap(spacing: 7, runSpacing: 7, children: actions),
        ],
      ),
    );
  }
}
