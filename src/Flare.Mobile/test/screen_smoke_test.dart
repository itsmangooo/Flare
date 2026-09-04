import 'package:flare_mobile/core/api/models.dart';
import 'package:flare_mobile/core/providers.dart';
import 'package:flare_mobile/core/theme/flare_theme.dart';
import 'package:flare_mobile/features/activity/activity_page.dart';
import 'package:flare_mobile/features/containers/containers_page.dart';
import 'package:flare_mobile/features/deployments/deployments_page.dart';
import 'package:flare_mobile/features/overview/overview_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final now = DateTime(2026, 9, 4, 10);
  final activity = ActivityEventModel(
    id: 'event-1',
    kind: ActivityKind.infrastructure,
    action: 'container.restart',
    target: 'postgres',
    timestamp: now,
    result: OperationResult.succeeded,
  );

  Widget app(Widget page) => MaterialApp(theme: buildFlareTheme(), home: page);

  testWidgets('overview renders real metric structure', (tester) async {
    final overview = OverviewModel(
      generatedAt: now,
      freshness: DataFreshness.live,
      host: HostMetricsModel(
        hostName: 'lab-01',
        observedAt: now,
        cpuPercent: 42,
        loadAverage: 1.2,
        memoryUsedBytes: 8 * 1024 * 1024 * 1024,
        memoryTotalBytes: 16 * 1024 * 1024 * 1024,
        diskUsedBytes: 200 * 1024 * 1024 * 1024,
        diskTotalBytes: 500 * 1024 * 1024 * 1024,
        networkReceiveBytesPerSecond: 1024 * 1024,
        networkTransmitBytesPerSecond: 512 * 1024,
        uptime: const Duration(days: 12),
      ),
      containers: const ContainerTotalsModel(
        running: 4,
        stopped: 1,
        unhealthy: 0,
      ),
      history: <MetricPointModel>[
        MetricPointModel(
          timestamp: now.subtract(const Duration(seconds: 3)),
          cpuPercent: 38,
          memoryPercent: 49,
        ),
        MetricPointModel(timestamp: now, cpuPercent: 42, memoryPercent: 50),
      ],
      recentActivity: <ActivityEventModel>[activity],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          overviewProvider.overrideWith(
            (ref) => Stream<OverviewModel>.value(overview),
          ),
        ],
        child: app(const OverviewPage()),
      ),
    );
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('Overview'), findsOneWidget);
    expect(find.text('lab-01'), findsOneWidget);
    expect(find.text('42%'), findsOneWidget);
    expect(find.text('Containers'), findsOneWidget);
  });

  testWidgets('containers render dense backend rows', (tester) async {
    final container = ContainerSummaryModel(
      id: 'abc',
      name: 'postgres',
      image: 'postgres:17',
      state: ContainerState.running,
      health: HealthState.healthy,
      startedAt: now.subtract(const Duration(days: 5)),
      cpuPercent: 1.4,
      memoryBytes: 256 * 1024 * 1024,
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          containersProvider.overrideWith(
            (ref) async => <ContainerSummaryModel>[container],
          ),
        ],
        child: app(const ContainersPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('postgres'), findsOneWidget);
    expect(find.text('postgres:17'), findsOneWidget);
    expect(find.textContaining('CPU 1.4%'), findsOneWidget);
  });

  testWidgets('deployments render Coolify data', (tester) async {
    final deployment = DeploymentModel(
      uuid: 'deployment-1',
      resourceUuid: 'app-1',
      resourceName: 'flare-api',
      status: 'finished',
      branch: 'main',
      commit: 'a92bd3100000',
      commitMessage: 'ship Flutter client',
      startedAt: now.subtract(const Duration(minutes: 8)),
      finishedAt: now.subtract(const Duration(minutes: 7)),
      duration: const Duration(minutes: 1),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          deploymentsProvider.overrideWith(
            (ref) async => <DeploymentModel>[deployment],
          ),
        ],
        child: app(const DeploymentsPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('flare-api'), findsOneWidget);
    expect(find.text('SUCCESSFUL'), findsOneWidget);
    expect(find.text('a92bd310'), findsOneWidget);
  });

  testWidgets('activity renders infrastructure event timeline', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activityProvider.overrideWith(
            (ref) async => <ActivityEventModel>[activity],
          ),
        ],
        child: app(const ActivityPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Container Restart'), findsOneWidget);
    expect(find.text('postgres'), findsOneWidget);
    expect(find.text('Infrastructure'), findsOneWidget);
  });
}
