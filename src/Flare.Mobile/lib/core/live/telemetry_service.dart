import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:signalr_netcore/http_connection_options.dart';
import 'package:signalr_netcore/hub_connection.dart';
import 'package:signalr_netcore/hub_connection_builder.dart';

import '../api/api_client.dart';
import '../api/models.dart';
import '../auth/session_store.dart';

final class TelemetryService with WidgetsBindingObserver {
  TelemetryService(this._api, this._session) {
    WidgetsBinding.instance.addObserver(this);
  }

  final ApiClient _api;
  final SessionStore _session;
  final StreamController<OverviewModel> _snapshots =
      StreamController<OverviewModel>.broadcast();
  HubConnection? _connection;
  Timer? _freshnessTimer;
  DateTime? _lastSnapshotAt;
  OverviewModel? _lastSnapshot;
  bool _starting = false;
  bool _disposed = false;
  bool _foreground = true;

  Stream<OverviewModel> get snapshots => _snapshots.stream;

  Future<void> start() async {
    if (_disposed || !_foreground || _starting) return;
    final state = _connection?.state;
    if (state == HubConnectionState.Connected ||
        state == HubConnectionState.Connecting) {
      return;
    }
    _starting = true;
    try {
      await _fetchFreshSnapshot();
      final serverUrl = await _session.serverUrl;
      if (serverUrl == null || !await _session.hasUsableSession()) {
        return;
      }
      _connection ??= _buildConnection(serverUrl);
      await _connection!.start();
      _emitFreshness(DataFreshness.live);
      _startFreshnessMonitor();
    } on Object catch (error, stackTrace) {
      _emitFreshness(
        _lastSnapshot == null ? DataFreshness.offline : DataFreshness.stale,
      );
      if (_lastSnapshot == null && !_snapshots.isClosed) {
        _snapshots.addError(error, stackTrace);
      }
    } finally {
      _starting = false;
    }
  }

  Future<void> refresh() => _fetchFreshSnapshot();

  Future<void> stop({bool markStale = true}) async {
    _freshnessTimer?.cancel();
    _freshnessTimer = null;
    final connection = _connection;
    if (connection != null &&
        connection.state != HubConnectionState.Disconnected) {
      try {
        await connection.stop();
      } on Object {
        // Stopping live telemetry must never terminate a lifecycle transition.
      }
    }
    if (markStale) {
      _emitFreshness(
        _lastSnapshot == null ? DataFreshness.offline : DataFreshness.stale,
      );
    }
  }

  HubConnection _buildConnection(String serverUrl) {
    final options = HttpConnectionOptions(
      accessTokenFactory: _api.getValidAccessToken,
      logMessageContent: false,
      requestTimeout: 15000,
    );
    final connection = HubConnectionBuilder()
        .withUrl('$serverUrl/hubs/telemetry', options: options)
        .withAutomaticReconnect(retryDelays: <int>[0, 2000, 5000, 10000, 30000])
        .build();
    connection.on('SnapshotUpdated', (arguments) {
      if (arguments == null || arguments.isEmpty || arguments.first is! Map) {
        return;
      }
      try {
        final raw = (arguments.first! as Map).map(
          (key, value) => MapEntry(key.toString(), value),
        );
        _publish(OverviewModel.fromJson(raw));
      } on Object {
        // Ignore malformed hub messages; the periodic freshness state will expose staleness.
      }
    });
    connection.onreconnecting(
      ({error}) => _emitFreshness(DataFreshness.reconnecting),
    );
    connection.onreconnected(({connectionId}) {
      _emitFreshness(DataFreshness.live);
      unawaited(_fetchFreshSnapshot());
    });
    connection.onclose(({error}) {
      _freshnessTimer?.cancel();
      _emitFreshness(
        _lastSnapshot == null ? DataFreshness.offline : DataFreshness.stale,
      );
    });
    return connection;
  }

  Future<void> _fetchFreshSnapshot() async {
    try {
      final snapshot = OverviewModel.fromJson(
        await _api.getJson('api/v1/overview'),
      );
      _publish(snapshot);
    } on Object {
      _emitFreshness(
        _lastSnapshot == null ? DataFreshness.offline : DataFreshness.stale,
      );
      rethrow;
    }
  }

  void _publish(OverviewModel snapshot) {
    _lastSnapshot = snapshot;
    _lastSnapshotAt = DateTime.now();
    if (!_snapshots.isClosed) _snapshots.add(snapshot);
  }

  void _emitFreshness(DataFreshness freshness) {
    final snapshot = _lastSnapshot;
    if (snapshot == null || snapshot.freshness == freshness) return;
    _publish(snapshot.withFreshness(freshness));
  }

  void _startFreshnessMonitor() {
    _freshnessTimer?.cancel();
    _freshnessTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      final observed = _lastSnapshotAt;
      if (observed != null &&
          DateTime.now().difference(observed) > const Duration(seconds: 10)) {
        _emitFreshness(DataFreshness.stale);
      }
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    if (_foreground) {
      unawaited(start());
    } else if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.detached) {
      unawaited(stop());
    }
  }

  Future<void> reset() async {
    await stop(markStale: false);
    _connection = null;
    _lastSnapshot = null;
    _lastSnapshotAt = null;
  }

  Future<void> dispose() async {
    _disposed = true;
    WidgetsBinding.instance.removeObserver(this);
    await stop(markStale: false);
    await _snapshots.close();
  }
}
