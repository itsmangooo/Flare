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

enum _ContainerFilter { all, running, stopped, unhealthy }

final class ContainersPage extends ConsumerStatefulWidget {
  const ContainersPage({super.key});
  @override
  ConsumerState<ContainersPage> createState() => _ContainersPageState();
}

final class _ContainersPageState extends ConsumerState<ContainersPage> {
  final _search = TextEditingController();
  String _query = '';
  _ContainerFilter _filter = _ContainerFilter.all;

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  List<ContainerSummaryModel> _filtered(
    List<ContainerSummaryModel> source,
  ) => source
      .where((container) {
        final queryMatches =
            _query.isEmpty ||
            container.name.toLowerCase().contains(_query) ||
            container.image.toLowerCase().contains(_query);
        final filterMatches = switch (_filter) {
          _ContainerFilter.all => true,
          _ContainerFilter.running => container.state == ContainerState.running,
          _ContainerFilter.stopped => container.state == ContainerState.stopped,
          _ContainerFilter.unhealthy =>
            container.health == HealthState.unhealthy ||
                container.state == ContainerState.dead,
        };
        return queryMatches && filterMatches;
      })
      .toList(growable: false);

  @override
  Widget build(BuildContext context) {
    final containers = ref.watch(containersProvider);
    final palette = context.flare;
    return FlareScaffold(
      title: 'Containers',
      subtitle: 'Docker services and runtime health',
      actions: <Widget>[
        FlareIconButton(
          icon: PhosphorIconsRegular.userCircle,
          semanticLabel: 'Settings',
          onPressed: () => context.push('/settings'),
        ),
      ],
      body: Column(
        children: <Widget>[
          const SizedBox(height: 4),
          FlareSearchField(
            controller: _search,
            hint: 'Search containers',
            onChanged: (value) =>
                setState(() => _query = value.trim().toLowerCase()),
          ),
          const SizedBox(height: 10),
          FlareSegmentedControl<_ContainerFilter>(
            value: _filter,
            items: const <(_ContainerFilter, String)>[
              (_ContainerFilter.all, 'All'),
              (_ContainerFilter.running, 'Running'),
              (_ContainerFilter.stopped, 'Stopped'),
              (_ContainerFilter.unhealthy, 'Issues'),
            ],
            onChanged: (value) => setState(() => _filter = value),
          ),
          const SizedBox(height: 14),
          Expanded(
            child: containers.when(
              loading: () => const FlareLoading(label: 'Reading Docker state'),
              error: (error, _) => FlareErrorState(
                error: error,
                onRetry: () => ref.invalidate(containersProvider),
              ),
              data: (values) {
                final filtered = _filtered(values);
                if (filtered.isEmpty) {
                  return FlareEmptyState(
                    title: values.isEmpty ? 'No containers' : 'No matches',
                    message: values.isEmpty
                        ? 'Docker returned no containers.'
                        : 'Change the search or status filter.',
                    icon: PhosphorIconsRegular.cube,
                  );
                }
                return RefreshIndicator(
                  color: palette.accent,
                  backgroundColor: palette.surfaceHigh,
                  onRefresh: () async => ref.refresh(containersProvider.future),
                  child: ListView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    padding: const EdgeInsets.only(bottom: 24),
                    children: <Widget>[
                      FlareGroupedSurface(
                        padding: const EdgeInsets.symmetric(horizontal: 14),
                        child: Column(
                          children: List<Widget>.generate(
                            filtered.length * 2 - 1,
                            (index) {
                              if (index.isOdd) {
                                return const FlareDivider(indent: 36);
                              }
                              final container = filtered[index ~/ 2];
                              return FlareContainerTile(
                                key: ValueKey(container.id),
                                container: container,
                                onTap: () => context.push(
                                  '/containers/${Uri.encodeComponent(container.id)}',
                                ),
                              );
                            },
                          ),
                        ),
                      ),
                    ],
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
