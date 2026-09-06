import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';
import '../../design/components/flare_tiles.dart';

enum _ActivityFilter { all, infrastructure, security }

final class ActivityPage extends ConsumerStatefulWidget {
  const ActivityPage({super.key});

  @override
  ConsumerState<ActivityPage> createState() => _ActivityPageState();
}

final class _ActivityPageState extends ConsumerState<ActivityPage> {
  _ActivityFilter _filter = _ActivityFilter.all;

  List<ActivityEventModel> _filtered(List<ActivityEventModel> events) =>
      switch (_filter) {
        _ActivityFilter.all => events,
        _ActivityFilter.infrastructure =>
          events
              .where((event) => event.kind == ActivityKind.infrastructure)
              .toList(growable: false),
        _ActivityFilter.security =>
          events
              .where((event) => event.kind == ActivityKind.security)
              .toList(growable: false),
      };

  @override
  Widget build(BuildContext context) {
    final activity = ref.watch(activityProvider);
    final palette = context.flare;
    return FlareScaffold(
      title: 'Activity',
      subtitle: 'Infrastructure and security events',
      actions: <Widget>[
        FlareIconButton(
          icon: PhosphorIconsRegular.userCircle,
          semanticLabel: 'Settings',
          onPressed: () => context.push('/settings'),
        ),
      ],
      body: activity.when(
        loading: () =>
            const FlareLoading(label: 'Reading infrastructure activity'),
        error: (error, _) => FlareErrorState(
          error: error,
          onRetry: () => ref.invalidate(activityProvider),
        ),
        data: (events) {
          final visible = _filtered(events);
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Padding(
                padding: const EdgeInsets.only(top: 4, bottom: 14),
                child: FlareSegmentedControl<_ActivityFilter>(
                  value: _filter,
                  items: const <(_ActivityFilter, String)>[
                    (_ActivityFilter.all, 'All'),
                    (_ActivityFilter.infrastructure, 'Infrastructure'),
                    (_ActivityFilter.security, 'Security'),
                  ],
                  onChanged: (value) => setState(() => _filter = value),
                ),
              ),
              Expanded(
                child: visible.isEmpty
                    ? const FlareEmptyState(
                        title: 'No activity',
                        message: 'No events match the selected filter.',
                        icon: PhosphorIconsRegular.pulse,
                      )
                    : RefreshIndicator(
                        color: palette.accent,
                        backgroundColor: palette.surfaceHigh,
                        onRefresh: () async =>
                            ref.refresh(activityProvider.future),
                        child: ListView(
                          physics: const AlwaysScrollableScrollPhysics(),
                          padding: const EdgeInsets.only(bottom: 24),
                          children: <Widget>[
                            FlareGroupedSurface(
                              padding: const EdgeInsets.fromLTRB(14, 16, 14, 0),
                              child: Column(
                                children: List<Widget>.generate(
                                  visible.length,
                                  (index) => FlareActivityTile(
                                    event: visible[index],
                                    last: index == visible.length - 1,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
              ),
            ],
          );
        },
      ),
    );
  }
}
