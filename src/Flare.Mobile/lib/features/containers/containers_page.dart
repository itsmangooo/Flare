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

  void _nextFilter() => setState(() {
    _filter = _ContainerFilter
        .values[(_filter.index + 1) % _ContainerFilter.values.length];
  });

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
    return FlareScaffold(
      title: 'Containers',
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
          Row(
            children: <Widget>[
              Expanded(
                child: FlareSearchField(
                  controller: _search,
                  hint: 'Search containers',
                  onChanged: (value) =>
                      setState(() => _query = value.trim().toLowerCase()),
                ),
              ),
              const SizedBox(width: 9),
              FlareIconButton(
                icon: _filter == _ContainerFilter.all
                    ? PhosphorIconsRegular.funnel
                    : PhosphorIconsFill.funnel,
                semanticLabel: 'Filter: ${_filter.name}',
                accent: _filter != _ContainerFilter.all,
                onPressed: _nextFilter,
              ),
            ],
          ),
          if (_filter != _ContainerFilter.all) ...<Widget>[
            const SizedBox(height: 9),
            Align(
              alignment: Alignment.centerLeft,
              child: FlareStatusBadge(
                label: _filter.name,
                tone: FlareStatusTone.info,
                dot: false,
              ),
            ),
          ],
          const SizedBox(height: 9),
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
                  color: FlareColors.accent,
                  backgroundColor: FlareColors.surfaceHigh,
                  onRefresh: () async => ref.refresh(containersProvider.future),
                  child: ListView.separated(
                    physics: const AlwaysScrollableScrollPhysics(),
                    itemCount: filtered.length,
                    separatorBuilder: (_, _) => const FlareDivider(indent: 20),
                    itemBuilder: (context, index) {
                      final container = filtered[index];
                      return FlareContainerTile(
                        key: ValueKey(container.id),
                        container: container,
                        onTap: () => context.push(
                          '/containers/${Uri.encodeComponent(container.id)}',
                        ),
                      );
                    },
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
