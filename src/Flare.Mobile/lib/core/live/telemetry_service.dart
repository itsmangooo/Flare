import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';

import '../api/api_client.dart';
import '../api/models.dart';
import '../auth/session_store.dart';
import 'sse_decoder.dart';

final class TelemetryService with WidgetsBindingObserver {
  TelemetryService(this._api, this._session) {
    WidgetsBinding.instance.addObserver(this);
  }

  static const _retryDelays = <Duration>[
    Duration.zero,
    Duration(seconds: 2),
    Duration(seconds: 5),
    Duration(seconds: 10),
    Duration(seconds: 30),
  ];

  final ApiClient _api;
  final SessionStore _session;
  final StreamController<OverviewModel> _snapshots =
      StreamController<OverviewModel>.broadcast();
  HttpClient? _client;
  Future<void>? _streamTask;
  Completer<void>? _stopSignal;
  Timer? _freshnessTimer;
  DateTime? _lastSnapshotAt;
  OverviewModel? _lastSnapshot;
  int _generation = 0;
  bool _starting = false;
  bool _disposed = false;
  bool _foreground = true;

  Stream<OverviewModel> get snapshots => _snapshots.stream;

  Future<void> start() async {
    if (_disposed || !_foreground || _starting || _streamTask != null) return;
    _starting = true;
    try {
      await _fetchFreshSnapshot();
      final serverUrl = await _session.serverUrl;
      if (serverUrl == null || !await _session.hasUsableSession()) return;
      _startFreshnessMonitor();
      final generation = ++_generation;
      final stopSignal = Completer<void>();
      _stopSignal = stopSignal;
      final task = _runStream(serverUrl, generation, stopSignal.future);
      _streamTask = task;
      unawaited(
        task.whenComplete(() {
          if (_generation == generation) {
            _streamTask = null;
            _stopSignal = null;
          }
        }),
      );
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
    _generation++;
    if (!(_stopSignal?.isCompleted ?? true)) _stopSignal!.complete();
    _stopSignal = null;
    final task = _streamTask;
    _streamTask = null;
    _client?.close(force: true);
    _client = null;
    if (task != null) {
      try {
        await task;
      } on Object {
        // Closing the HTTP client is expected to terminate the active stream.
      }
    }
    if (markStale) {
      _emitFreshness(
        _lastSnapshot == null ? DataFreshness.offline : DataFreshness.stale,
      );
    }
  }

  Future<void> _runStream(
    String serverUrl,
    int generation,
    Future<void> stopSignal,
  ) async {
    var attempt = 0;
    while (_isActive(generation)) {
      final delay = _retryDelays[attempt.clamp(0, _retryDelays.length - 1)];
      if (delay > Duration.zero) {
        _emitFreshness(DataFreshness.reconnecting);
        await Future.any<void>(<Future<void>>[
          Future<void>.delayed(delay),
          stopSignal,
        ]);
        if (!_isActive(generation)) return;
      }
      try {
        await _consumeStream(serverUrl, generation);
        attempt = 1;
      } on Object {
        if (!_isActive(generation)) return;
        _emitFreshness(
          _lastSnapshot == null ? DataFreshness.offline : DataFreshness.stale,
        );
        attempt = (attempt + 1).clamp(1, _retryDelays.length - 1);
      } finally {
        _client?.close(force: true);
        _client = null;
      }
    }
  }

  Future<void> _consumeStream(String serverUrl, int generation) async {
    final token = await _api.getValidAccessToken();
    if (!_isActive(generation)) return;
    final client = HttpClient()
      ..connectionTimeout = const Duration(seconds: 12);
    _client = client;
    final streamUri = Uri.parse('$serverUrl/api/v1/telemetry');
    final request = await client.getUrl(streamUri);
    request.headers
      ..set(HttpHeaders.acceptHeader, 'text/event-stream')
      ..set(HttpHeaders.authorizationHeader, 'Bearer $token')
      ..set(HttpHeaders.cacheControlHeader, 'no-cache');
    final response = await request.close().timeout(const Duration(seconds: 20));
    if (response.statusCode != HttpStatus.ok) {
      await response.drain<void>();
      throw HttpException(
        'Telemetry stream returned HTTP ${response.statusCode}.',
        uri: streamUri,
      );
    }

    _emitFreshness(DataFreshness.live);
    await for (final event
        in response
            .transform(utf8.decoder)
            .transform(const LineSplitter())
            .transform(const SseEventDecoder())) {
      if (!_isActive(generation)) return;
      if (event.name != 'snapshot') continue;
      final value = jsonDecode(event.data);
      if (value is! Map) continue;
      try {
        final raw = value.map((key, item) => MapEntry(key.toString(), item));
        _publish(OverviewModel.fromJson(raw));
      } on Object {
        // Ignore malformed events; freshness monitoring exposes stale streams.
      }
    }
    if (_isActive(generation)) {
      throw const HttpException('Telemetry stream ended unexpectedly.');
    }
  }

  bool _isActive(int generation) =>
      !_disposed && _foreground && _generation == generation;

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
