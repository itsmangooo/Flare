import 'package:flare_mobile/core/api/models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses .NET duration with days and fractional seconds', () {
    expect(
      parseDotNetDuration('18.04:03:02.5000000'),
      const Duration(days: 18, hours: 4, minutes: 3, seconds: 3),
    );
  });

  test('parses overview contract and nullable host metrics', () {
    final result = OverviewModel.fromJson(<String, dynamic>{
      'generatedAt': '2026-09-04T08:00:00Z',
      'freshness': 'Live',
      'host': <String, dynamic>{
        'hostName': 'lab-01',
        'observedAt': '2026-09-04T08:00:00Z',
        'cpuPercent': 68.2,
        'memoryUsedBytes': 10800000000,
        'memoryTotalBytes': 16000000000,
      },
      'containers': <String, dynamic>{
        'running': 13,
        'stopped': 2,
        'unhealthy': 1,
      },
      'history': <dynamic>[],
      'recentActivity': <dynamic>[],
    });

    expect(result.freshness, DataFreshness.live);
    expect(result.host.hostName, 'lab-01');
    expect(result.host.cpuPercent, 68.2);
    expect(result.host.diskPercent, isNull);
    expect(result.containers.running, 13);
  });

  test('unknown enum values degrade safely', () {
    final model = ContainerSummaryModel.fromJson(<String, dynamic>{
      'id': 'abc',
      'name': 'worker',
      'image': 'flare-worker',
      'state': 'a-future-state',
      'health': 'a-future-health',
    });

    expect(model.state, ContainerState.unknown);
    expect(model.health, HealthState.unknown);
  });

  test('parses Cloudflare domain models without private origin data', () {
    final status = CloudflareStatusModel.fromJson(<String, dynamic>{
      'configured': true,
      'tunnelsConfigured': true,
    });
    final record = DNSRecordModel.fromJson(<String, dynamic>{
      'id': 'record-1',
      'zoneId': 'zone-1',
      'name': 'flare.example.test',
      'type': 'CNAME',
      'target': 'tunnel.example.test',
      'ttl': 1,
      'proxied': true,
      'proxiable': true,
    });
    final route = TunnelRouteModel.fromJson(<String, dynamic>{
      'tunnelId': 'tunnel-1',
      'hostname': 'flare.example.test',
      'path': '/api/*',
      'originKind': 'http',
      'service': 'http://private-host:8080',
    });

    expect(status.configured, isTrue);
    expect(record.proxied, isTrue);
    expect(record.target, 'tunnel.example.test');
    expect(route.hostname, 'flare.example.test');
    expect(route.originKind, 'http');
  });
}
