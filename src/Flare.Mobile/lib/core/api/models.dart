enum DataFreshness { live, reconnecting, stale, offline }

enum ContainerState { running, stopped, paused, restarting, dead, unknown }

enum HealthState { healthy, unhealthy, starting, none, unknown }

enum OperationResult { succeeded, failed }

enum ActivityKind { infrastructure, security }

T _enumValue<T extends Enum>(List<T> values, Object? raw, T fallback) {
  final value = raw?.toString().toLowerCase();
  return values.firstWhere(
    (item) => item.name.toLowerCase() == value,
    orElse: () => fallback,
  );
}

DateTime? _date(Object? value) =>
    value is String ? DateTime.tryParse(value)?.toLocal() : null;

double? _double(Object? value) => value is num ? value.toDouble() : null;

int? _int(Object? value) => value is num ? value.toInt() : null;

Duration? parseDotNetDuration(Object? raw) {
  if (raw is! String || raw.isEmpty) return null;
  final parts = raw.split(':');
  if (parts.length != 3) return null;
  final dayAndHour = parts[0].split('.');
  final days = dayAndHour.length == 2 ? int.tryParse(dayAndHour[0]) ?? 0 : 0;
  final hours = int.tryParse(dayAndHour.last) ?? 0;
  final minutes = int.tryParse(parts[1]) ?? 0;
  final seconds = double.tryParse(parts[2])?.round() ?? 0;
  return Duration(days: days, hours: hours, minutes: minutes, seconds: seconds);
}

Map<String, dynamic> _map(Object? value) => value is Map<String, dynamic>
    ? value
    : Map<String, dynamic>.from(value! as Map<Object?, Object?>);

List<T> _list<T>(Object? value, T Function(Map<String, dynamic>) decode) =>
    value is List
    ? value.map((item) => decode(_map(item))).toList(growable: false)
    : <T>[];

final class UserModel {
  const UserModel({required this.id, required this.email, required this.roles});
  factory UserModel.fromJson(Map<String, dynamic> json) => UserModel(
    id: json['id']?.toString() ?? '',
    email: json['email']?.toString() ?? '',
    roles: (json['roles'] as List<dynamic>? ?? const <dynamic>[])
        .map((role) => role.toString())
        .toList(growable: false),
  );
  final String id;
  final String email;
  final List<String> roles;
  bool get isAdministrator =>
      roles.any((role) => role.toLowerCase() == 'administrator');
}

final class TokenModel {
  const TokenModel({
    required this.accessToken,
    required this.accessTokenExpiresAt,
    required this.refreshToken,
    required this.refreshTokenExpiresAt,
    required this.user,
  });
  factory TokenModel.fromJson(Map<String, dynamic> json) => TokenModel(
    accessToken: json['accessToken'] as String,
    accessTokenExpiresAt: DateTime.parse(
      json['accessTokenExpiresAt'] as String,
    ),
    refreshToken: json['refreshToken'] as String,
    refreshTokenExpiresAt: DateTime.parse(
      json['refreshTokenExpiresAt'] as String,
    ),
    user: UserModel.fromJson(_map(json['user'])),
  );
  final String accessToken;
  final DateTime accessTokenExpiresAt;
  final String refreshToken;
  final DateTime refreshTokenExpiresAt;
  final UserModel user;
}

final class HostMetricsModel {
  const HostMetricsModel({
    required this.hostName,
    required this.observedAt,
    this.cpuPercent,
    this.loadAverage,
    this.memoryUsedBytes,
    this.memoryTotalBytes,
    this.diskUsedBytes,
    this.diskTotalBytes,
    this.networkReceiveBytesPerSecond,
    this.networkTransmitBytesPerSecond,
    this.uptime,
  });
  factory HostMetricsModel.fromJson(Map<String, dynamic> json) =>
      HostMetricsModel(
        hostName: json['hostName']?.toString() ?? 'Homelab',
        observedAt: _date(json['observedAt']) ?? DateTime.now(),
        cpuPercent: _double(json['cpuPercent']),
        loadAverage: _double(json['loadAverage']),
        memoryUsedBytes: _int(json['memoryUsedBytes']),
        memoryTotalBytes: _int(json['memoryTotalBytes']),
        diskUsedBytes: _int(json['diskUsedBytes']),
        diskTotalBytes: _int(json['diskTotalBytes']),
        networkReceiveBytesPerSecond: _double(
          json['networkReceiveBytesPerSecond'],
        ),
        networkTransmitBytesPerSecond: _double(
          json['networkTransmitBytesPerSecond'],
        ),
        uptime: parseDotNetDuration(json['uptime']),
      );
  final String hostName;
  final DateTime observedAt;
  final double? cpuPercent;
  final double? loadAverage;
  final int? memoryUsedBytes;
  final int? memoryTotalBytes;
  final int? diskUsedBytes;
  final int? diskTotalBytes;
  final double? networkReceiveBytesPerSecond;
  final double? networkTransmitBytesPerSecond;
  final Duration? uptime;
  double? get memoryPercent =>
      memoryUsedBytes != null &&
          memoryTotalBytes != null &&
          memoryTotalBytes! > 0
      ? memoryUsedBytes! / memoryTotalBytes! * 100
      : null;
  double? get diskPercent =>
      diskUsedBytes != null && diskTotalBytes != null && diskTotalBytes! > 0
      ? diskUsedBytes! / diskTotalBytes! * 100
      : null;
}

final class MetricPointModel {
  const MetricPointModel({
    required this.timestamp,
    this.cpuPercent,
    this.memoryPercent,
  });
  factory MetricPointModel.fromJson(Map<String, dynamic> json) =>
      MetricPointModel(
        timestamp: _date(json['timestamp']) ?? DateTime.now(),
        cpuPercent: _double(json['cpuPercent']),
        memoryPercent: _double(json['memoryPercent']),
      );
  final DateTime timestamp;
  final double? cpuPercent;
  final double? memoryPercent;
}

final class ContainerTotalsModel {
  const ContainerTotalsModel({this.running, this.stopped, this.unhealthy});
  factory ContainerTotalsModel.fromJson(Map<String, dynamic> json) =>
      ContainerTotalsModel(
        running: _int(json['running']),
        stopped: _int(json['stopped']),
        unhealthy: _int(json['unhealthy']),
      );
  final int? running;
  final int? stopped;
  final int? unhealthy;
}

final class ActivityEventModel {
  const ActivityEventModel({
    required this.id,
    required this.kind,
    required this.action,
    required this.target,
    required this.timestamp,
    required this.result,
    this.actor,
  });
  factory ActivityEventModel.fromJson(Map<String, dynamic> json) =>
      ActivityEventModel(
        id: json['id']?.toString() ?? '',
        kind: _enumValue(
          ActivityKind.values,
          json['kind'],
          ActivityKind.infrastructure,
        ),
        action: json['action']?.toString() ?? 'Infrastructure event',
        target: json['target']?.toString() ?? '',
        timestamp: _date(json['timestamp']) ?? DateTime.now(),
        result: _enumValue(
          OperationResult.values,
          json['result'],
          OperationResult.failed,
        ),
        actor: json['actor']?.toString(),
      );
  final String id;
  final ActivityKind kind;
  final String action;
  final String target;
  final DateTime timestamp;
  final OperationResult result;
  final String? actor;
}

final class OverviewModel {
  const OverviewModel({
    required this.generatedAt,
    required this.freshness,
    required this.host,
    required this.containers,
    required this.history,
    required this.recentActivity,
  });
  factory OverviewModel.fromJson(Map<String, dynamic> json) => OverviewModel(
    generatedAt: _date(json['generatedAt']) ?? DateTime.now(),
    freshness: _enumValue(
      DataFreshness.values,
      json['freshness'],
      DataFreshness.offline,
    ),
    host: HostMetricsModel.fromJson(_map(json['host'])),
    containers: ContainerTotalsModel.fromJson(_map(json['containers'])),
    history: _list(json['history'], MetricPointModel.fromJson),
    recentActivity: _list(json['recentActivity'], ActivityEventModel.fromJson),
  );
  final DateTime generatedAt;
  final DataFreshness freshness;
  final HostMetricsModel host;
  final ContainerTotalsModel containers;
  final List<MetricPointModel> history;
  final List<ActivityEventModel> recentActivity;
  OverviewModel withFreshness(DataFreshness value) => OverviewModel(
    generatedAt: generatedAt,
    freshness: value,
    host: host,
    containers: containers,
    history: history,
    recentActivity: recentActivity,
  );
}

final class ContainerSummaryModel {
  const ContainerSummaryModel({
    required this.id,
    required this.name,
    required this.image,
    required this.state,
    required this.health,
    this.startedAt,
    this.cpuPercent,
    this.memoryBytes,
  });
  factory ContainerSummaryModel.fromJson(Map<String, dynamic> json) =>
      ContainerSummaryModel(
        id: json['id']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Unnamed container',
        image: json['image']?.toString() ?? 'Unknown image',
        state: _enumValue(
          ContainerState.values,
          json['state'],
          ContainerState.unknown,
        ),
        health: _enumValue(
          HealthState.values,
          json['health'],
          HealthState.unknown,
        ),
        startedAt: _date(json['startedAt']),
        cpuPercent: _double(json['cpuPercent']),
        memoryBytes: _int(json['memoryBytes']),
      );
  final String id;
  final String name;
  final String image;
  final ContainerState state;
  final HealthState health;
  final DateTime? startedAt;
  final double? cpuPercent;
  final int? memoryBytes;
}

final class PortBindingModel {
  const PortBindingModel({
    required this.privatePort,
    required this.protocol,
    this.publicPort,
    this.hostIp,
  });
  factory PortBindingModel.fromJson(Map<String, dynamic> json) =>
      PortBindingModel(
        privatePort: _int(json['privatePort']) ?? 0,
        publicPort: _int(json['publicPort']),
        protocol: json['protocol']?.toString() ?? 'tcp',
        hostIp: json['hostIp']?.toString(),
      );
  final int privatePort;
  final int? publicPort;
  final String protocol;
  final String? hostIp;
}

final class ContainerDetailModel {
  const ContainerDetailModel({
    required this.id,
    required this.shortId,
    required this.name,
    required this.image,
    required this.state,
    required this.health,
    required this.createdAt,
    required this.restartCount,
    required this.ports,
    required this.labels,
    this.startedAt,
    this.cpuPercent,
    this.memoryBytes,
    this.memoryLimitBytes,
    this.networkReceiveBytes,
    this.networkTransmitBytes,
  });
  factory ContainerDetailModel.fromJson(
    Map<String, dynamic> json,
  ) => ContainerDetailModel(
    id: json['id']?.toString() ?? '',
    shortId: json['shortId']?.toString() ?? '',
    name: json['name']?.toString() ?? 'Container',
    image: json['image']?.toString() ?? 'Unknown image',
    state: _enumValue(
      ContainerState.values,
      json['state'],
      ContainerState.unknown,
    ),
    health: _enumValue(HealthState.values, json['health'], HealthState.unknown),
    createdAt: _date(json['createdAt']) ?? DateTime.now(),
    startedAt: _date(json['startedAt']),
    restartCount: _int(json['restartCount']) ?? 0,
    cpuPercent: _double(json['cpuPercent']),
    memoryBytes: _int(json['memoryBytes']),
    memoryLimitBytes: _int(json['memoryLimitBytes']),
    networkReceiveBytes: _int(json['networkReceiveBytes']),
    networkTransmitBytes: _int(json['networkTransmitBytes']),
    ports: _list(json['ports'], PortBindingModel.fromJson),
    labels:
        (json['labels'] as Map<Object?, Object?>? ?? const <Object?, Object?>{})
            .map((key, value) => MapEntry(key.toString(), value.toString())),
  );
  final String id;
  final String shortId;
  final String name;
  final String image;
  final ContainerState state;
  final HealthState health;
  final DateTime createdAt;
  final DateTime? startedAt;
  final int restartCount;
  final double? cpuPercent;
  final int? memoryBytes;
  final int? memoryLimitBytes;
  final int? networkReceiveBytes;
  final int? networkTransmitBytes;
  final List<PortBindingModel> ports;
  final Map<String, String> labels;
}

final class LogPageModel {
  const LogPageModel({
    required this.lines,
    required this.retrievedAt,
    required this.truncated,
    required this.requestedTail,
    this.oldestTimestamp,
    this.newestTimestamp,
  });
  factory LogPageModel.fromJson(Map<String, dynamic> json) => LogPageModel(
    lines: (json['lines'] as List<dynamic>? ?? const <dynamic>[])
        .map((line) => line.toString())
        .toList(growable: false),
    retrievedAt: _date(json['retrievedAt']) ?? DateTime.now(),
    truncated: json['truncated'] == true,
    requestedTail: _int(json['requestedTail']) ?? 300,
    oldestTimestamp: _date(json['oldestTimestamp']),
    newestTimestamp: _date(json['newestTimestamp']),
  );
  final List<String> lines;
  final DateTime retrievedAt;
  final bool truncated;
  final int requestedTail;
  final DateTime? oldestTimestamp;
  final DateTime? newestTimestamp;
}

final class DeploymentModel {
  const DeploymentModel({
    required this.uuid,
    required this.resourceUuid,
    required this.resourceName,
    this.status,
    this.branch,
    this.commit,
    this.commitMessage,
    this.startedAt,
    this.finishedAt,
    this.duration,
    this.logs,
  });
  factory DeploymentModel.fromJson(Map<String, dynamic> json) =>
      DeploymentModel(
        uuid: json['uuid']?.toString() ?? '',
        resourceUuid: json['resourceUuid']?.toString() ?? '',
        resourceName: json['resourceName']?.toString() ?? 'Application',
        status: json['status']?.toString(),
        branch: json['branch']?.toString(),
        commit: json['commit']?.toString(),
        commitMessage: json['commitMessage']?.toString(),
        startedAt: _date(json['startedAt']),
        finishedAt: _date(json['finishedAt']),
        duration: parseDotNetDuration(json['duration']),
        logs: json['logs']?.toString(),
      );
  final String uuid;
  final String resourceUuid;
  final String resourceName;
  final String? status;
  final String? branch;
  final String? commit;
  final String? commitMessage;
  final DateTime? startedAt;
  final DateTime? finishedAt;
  final Duration? duration;
  final String? logs;
}

final class PagedModel<T> {
  const PagedModel({
    required this.items,
    required this.page,
    required this.pageSize,
    required this.hasMore,
  });
  factory PagedModel.fromJson(
    Map<String, dynamic> json,
    T Function(Map<String, dynamic>) decode,
  ) => PagedModel(
    items: _list(json['items'], decode),
    page: _int(json['page']) ?? 1,
    pageSize: _int(json['pageSize']) ?? 0,
    hasMore: json['hasMore'] == true,
  );
  final List<T> items;
  final int page;
  final int pageSize;
  final bool hasMore;
}

final class CoolifyServerModel {
  const CoolifyServerModel({
    required this.uuid,
    required this.name,
    this.isReachable,
    this.isUsable,
  });
  factory CoolifyServerModel.fromJson(Map<String, dynamic> json) =>
      CoolifyServerModel(
        uuid: json['uuid']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Server',
        isReachable: json['isReachable'] as bool?,
        isUsable: json['isUsable'] as bool?,
      );
  final String uuid;
  final String name;
  final bool? isReachable;
  final bool? isUsable;
}

final class CoolifyApplicationModel {
  const CoolifyApplicationModel({
    required this.uuid,
    required this.name,
    this.status,
    this.fqdn,
    this.gitBranch,
  });
  factory CoolifyApplicationModel.fromJson(Map<String, dynamic> json) =>
      CoolifyApplicationModel(
        uuid: json['uuid']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Application',
        status: json['status']?.toString(),
        fqdn: json['fqdn']?.toString(),
        gitBranch: json['gitBranch']?.toString(),
      );
  final String uuid;
  final String name;
  final String? status;
  final String? fqdn;
  final String? gitBranch;
}

final class CoolifyServiceModel {
  const CoolifyServiceModel({
    required this.uuid,
    required this.name,
    this.status,
    this.description,
  });
  factory CoolifyServiceModel.fromJson(Map<String, dynamic> json) =>
      CoolifyServiceModel(
        uuid: json['uuid']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Service',
        status: json['status']?.toString(),
        description: json['description']?.toString(),
      );
  final String uuid;
  final String name;
  final String? status;
  final String? description;
}

final class CoolifyResourceModel {
  const CoolifyResourceModel({
    required this.uuid,
    required this.name,
    required this.type,
    this.status,
  });
  factory CoolifyResourceModel.fromJson(Map<String, dynamic> json) =>
      CoolifyResourceModel(
        uuid: json['uuid']?.toString() ?? '',
        name: json['name']?.toString() ?? 'Resource',
        type: json['type']?.toString() ?? 'unknown',
        status: json['status']?.toString(),
      );
  final String uuid;
  final String name;
  final String type;
  final String? status;
}

final class ServerInfoModel {
  const ServerInfoModel({
    required this.name,
    required this.apiVersion,
    required this.serverVersion,
    required this.serverTime,
  });
  factory ServerInfoModel.fromJson(Map<String, dynamic> json) =>
      ServerInfoModel(
        name: json['name']?.toString() ?? 'Flare',
        apiVersion: json['apiVersion']?.toString() ?? 'unknown',
        serverVersion: json['serverVersion']?.toString() ?? 'unknown',
        serverTime: _date(json['serverTime']) ?? DateTime.now(),
      );
  final String name;
  final String apiVersion;
  final String serverVersion;
  final DateTime serverTime;
}
