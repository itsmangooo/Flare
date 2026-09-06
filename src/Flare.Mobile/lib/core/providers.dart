import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'api/api_client.dart';
import 'api/models.dart';
import 'auth/session_store.dart';
import 'live/telemetry_service.dart';

enum StartupDestination { connect, login, overview }

final sessionStoreProvider = Provider<SessionStore>((ref) {
  final store = SessionStore();
  ref.onDispose(store.dispose);
  return store;
});

final apiClientProvider = Provider<ApiClient>((ref) {
  final client = ApiClient(ref.watch(sessionStoreProvider));
  ref.onDispose(client.dispose);
  return client;
});

final startupDestinationProvider = FutureProvider<StartupDestination>((
  ref,
) async {
  final session = ref.watch(sessionStoreProvider);
  if (await session.serverUrl == null) return StartupDestination.connect;
  return await session.hasUsableSession()
      ? StartupDestination.overview
      : StartupDestination.login;
});

final telemetryServiceProvider = Provider<TelemetryService>((ref) {
  final service = TelemetryService(
    ref.watch(apiClientProvider),
    ref.watch(sessionStoreProvider),
  );
  ref.onDispose(() => unawaited(service.dispose()));
  return service;
});

final overviewProvider = StreamProvider.autoDispose<OverviewModel>((ref) {
  final service = ref.watch(telemetryServiceProvider);
  unawaited(service.start());
  ref.onDispose(() => unawaited(service.stop()));
  return service.snapshots;
});

final containersProvider =
    FutureProvider.autoDispose<List<ContainerSummaryModel>>((ref) async {
      final json = await ref
          .watch(apiClientProvider)
          .getList('api/v1/containers');
      return json
          .whereType<Map>()
          .map(
            (item) => ContainerSummaryModel.fromJson(
              item.map((key, value) => MapEntry(key.toString(), value)),
            ),
          )
          .toList(growable: false);
    });

final deploymentsProvider = FutureProvider.autoDispose<List<DeploymentModel>>((
  ref,
) async {
  final json = await ref
      .watch(apiClientProvider)
      .getJson(
        'api/v1/coolify/deployments',
        query: <String, Object>{'page': 1, 'pageSize': 50},
      );
  return PagedModel<DeploymentModel>.fromJson(
    json,
    DeploymentModel.fromJson,
  ).items;
});

final activityProvider = FutureProvider.autoDispose<List<ActivityEventModel>>((
  ref,
) async {
  final json = await ref
      .watch(apiClientProvider)
      .getJson(
        'api/v1/activity',
        query: <String, Object>{'page': 1, 'pageSize': 60},
      );
  return PagedModel<ActivityEventModel>.fromJson(
    json,
    ActivityEventModel.fromJson,
  ).items;
});

typedef AlertsData = ({List<AlertModel> items, int unreadCount});

final alertsProvider = FutureProvider.autoDispose.family<AlertsData, bool>((
  ref,
  unreadOnly,
) async {
  final json = await ref
      .watch(apiClientProvider)
      .getJson(
        'api/v1/alerts',
        query: <String, Object>{
          'page': 1,
          'pageSize': 100,
          'unreadOnly': unreadOnly,
        },
      );
  final items = (json['items'] as List<dynamic>? ?? const <dynamic>[])
      .whereType<Map>()
      .map(
        (item) => AlertModel.fromJson(
          item.map((key, value) => MapEntry(key.toString(), value)),
        ),
      )
      .toList(growable: false);
  return (items: items, unreadCount: _providerInt(json['unreadCount']));
});

int _providerInt(Object? value) => value is num ? value.toInt() : 0;

typedef CoolifyData = ({
  List<CoolifyServerModel> servers,
  List<CoolifyApplicationModel> applications,
  List<CoolifyServiceModel> services,
});

final coolifyProvider = FutureProvider.autoDispose<CoolifyData>((ref) async {
  final api = ref.watch(apiClientProvider);
  final values = await Future.wait<List<dynamic>>(<Future<List<dynamic>>>[
    api.getList('api/v1/coolify/servers'),
    api.getList('api/v1/coolify/applications'),
    api.getList('api/v1/coolify/services'),
  ]);
  Map<String, dynamic> convert(Object value) =>
      (value as Map).map((key, item) => MapEntry(key.toString(), item));
  return (
    servers: values[0]
        .map((item) => CoolifyServerModel.fromJson(convert(item)))
        .toList(growable: false),
    applications: values[1]
        .map((item) => CoolifyApplicationModel.fromJson(convert(item)))
        .toList(growable: false),
    services: values[2]
        .map((item) => CoolifyServiceModel.fromJson(convert(item)))
        .toList(growable: false),
  );
});

typedef DomainsData = ({
  CloudflareStatusModel status,
  List<DomainZoneModel> zones,
  List<CloudflareTunnelModel> tunnels,
});

final domainsProvider = FutureProvider.autoDispose<DomainsData>((ref) async {
  final api = ref.watch(apiClientProvider);
  final status = CloudflareStatusModel.fromJson(
    await api.getJson('api/v1/domains/status'),
  );
  if (!status.configured) {
    return (
      status: status,
      zones: const <DomainZoneModel>[],
      tunnels: const <CloudflareTunnelModel>[],
    );
  }

  Map<String, dynamic> convert(Object value) =>
      (value as Map).map((key, item) => MapEntry(key.toString(), item));
  final zones = (await api.getList('api/v1/domains/zones'))
      .map((item) => DomainZoneModel.fromJson(convert(item)))
      .toList(growable: false);
  final tunnels = status.tunnelsConfigured
      ? (await api.getList('api/v1/domains/tunnels'))
            .map((item) => CloudflareTunnelModel.fromJson(convert(item)))
            .toList(growable: false)
      : const <CloudflareTunnelModel>[];
  return (status: status, zones: zones, tunnels: tunnels);
});

final domainRecordsProvider = FutureProvider.autoDispose
    .family<List<DNSRecordModel>, String>((ref, zoneId) async {
      final items = await ref
          .watch(apiClientProvider)
          .getList(
            'api/v1/domains/zones/${Uri.encodeComponent(zoneId)}/records',
          );
      return items
          .map(
            (item) => DNSRecordModel.fromJson(
              (item as Map).map(
                (key, value) => MapEntry(key.toString(), value),
              ),
            ),
          )
          .toList(growable: false);
    });

final tunnelRoutesProvider = FutureProvider.autoDispose
    .family<List<TunnelRouteModel>, String>((ref, tunnelId) async {
      final items = await ref
          .watch(apiClientProvider)
          .getList(
            'api/v1/domains/tunnels/${Uri.encodeComponent(tunnelId)}/routes',
          );
      return items
          .map(
            (item) => TunnelRouteModel.fromJson(
              (item as Map).map(
                (key, value) => MapEntry(key.toString(), value),
              ),
            ),
          )
          .toList(growable: false);
    });
