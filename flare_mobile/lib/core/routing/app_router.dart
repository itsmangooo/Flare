import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../design/components/flare_scaffold.dart';
import '../../features/activity/activity_page.dart';
import '../../features/alerts/alerts_page.dart';
import '../../features/auth/login_page.dart';
import '../../features/connect/connect_page.dart';
import '../../features/connect/startup_page.dart';
import '../../features/containers/container_detail_page.dart';
import '../../features/containers/containers_page.dart';
import '../../features/deployments/coolify_resources_page.dart';
import '../../features/deployments/deployment_detail_page.dart';
import '../../features/deployments/deployments_page.dart';
import '../../features/domains/domains_page.dart';
import '../../features/overview/overview_page.dart';
import '../../features/settings/notification_preferences_page.dart';
import '../../features/settings/settings_page.dart';

final _rootNavigatorKey = GlobalKey<NavigatorState>(debugLabel: 'root');

final GoRouter flareRouter = GoRouter(
  navigatorKey: _rootNavigatorKey,
  initialLocation: '/',
  routes: <RouteBase>[
    GoRoute(path: '/', builder: (context, state) => const StartupPage()),
    GoRoute(path: '/connect', builder: (context, state) => const ConnectPage()),
    GoRoute(path: '/login', builder: (context, state) => const LoginPage()),
    StatefulShellRoute.indexedStack(
      builder: (context, state, shell) => _FlareNavigationShell(shell: shell),
      branches: <StatefulShellBranch>[
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(
              path: '/overview',
              builder: (context, state) => const OverviewPage(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(
              path: '/containers',
              builder: (context, state) => const ContainersPage(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(
              path: '/deployments',
              builder: (context, state) => const DeploymentsPage(),
            ),
          ],
        ),
        StatefulShellBranch(
          routes: <RouteBase>[
            GoRoute(
              path: '/activity',
              builder: (context, state) => const ActivityPage(),
            ),
          ],
        ),
      ],
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/alerts',
      builder: (context, state) => const AlertsPage(),
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/containers/:id',
      builder: (context, state) =>
          ContainerDetailPage(id: state.pathParameters['id'] ?? ''),
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/deployments/:id',
      builder: (context, state) =>
          DeploymentDetailPage(uuid: state.pathParameters['id'] ?? ''),
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/coolify',
      builder: (context, state) => const CoolifyResourcesPage(),
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/domains',
      builder: (context, state) => const DomainsPage(),
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/settings',
      builder: (context, state) => const SettingsPage(),
    ),
    GoRoute(
      parentNavigatorKey: _rootNavigatorKey,
      path: '/settings/notifications',
      builder: (context, state) => const NotificationPreferencesPage(),
    ),
  ],
);

final class _FlareNavigationShell extends StatelessWidget {
  const _FlareNavigationShell({
    required this.shell,
  });

  final StatefulNavigationShell shell;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.transparent,
      extendBody: true,
      body: Stack(
        fit: StackFit.expand,
        children: <Widget>[
          shell,

          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: FlareBottomNav(
              index: shell.currentIndex,
              onSelected: (index) => shell.goBranch(
                index,
                initialLocation: index == shell.currentIndex,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
