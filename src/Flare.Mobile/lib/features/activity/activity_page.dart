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
                padding: const EdgeInsets.only(top: 6, bottom: 12),
                child: Wrap(
                  spacing: 7,
                  children: _ActivityFilter.values
                      .map((filter) {
                        final label = switch (filter) {
                          _ActivityFilter.all => 'All',
                          _ActivityFilter.infrastructure => 'Infrastructure',
                          _ActivityFilter.security => 'Security',
                        };
                        return _FilterChip(
                          label: label,
                          selected: _filter == filter,
                          onTap: () => setState(() => _filter = filter),
                        );
                      })
                      .toList(growable: false),
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
                        child: ListView.builder(
                          physics: const AlwaysScrollableScrollPhysics(),
                          padding: const EdgeInsets.only(top: 8, bottom: 24),
                          itemCount: visible.length,
                          itemBuilder: (context, index) => FlareActivityTile(
                            event: visible[index],
                            last: index == visible.length - 1,
                          ),
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

final class _FilterChip extends StatelessWidget {
  const _FilterChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Semantics(
      button: true,
      selected: selected,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(FlareRadii.small),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 170),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            color: selected ? palette.accentSoft : palette.surface,
            borderRadius: BorderRadius.circular(FlareRadii.small),
            border: Border.all(
              color: selected
                  ? palette.accent.withValues(alpha: 0.33)
                  : palette.border,
            ),
          ),
          child: Text(
            label,
            style: FlareType.metadata.copyWith(
              color: selected ? palette.accent : palette.textSecondary,
            ),
          ),
        ),
      ),
    );
  }
}
