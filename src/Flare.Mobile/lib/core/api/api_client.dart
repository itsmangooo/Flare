import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

import '../auth/session_store.dart';
import 'models.dart';
import 'server_url_validator.dart';

final class FlareApiException implements Exception {
  const FlareApiException(this.message, {this.statusCode, this.cause});
  final String message;
  final int? statusCode;
  final Object? cause;
  bool get isUnauthorized => statusCode == HttpStatus.unauthorized;
  @override
  String toString() => message;
}

enum ConnectionOutcome {
  connected,
  invalidUrl,
  timeout,
  tlsError,
  unreachable,
  notReady,
  invalidResponse,
  storageError,
}

final class ConnectionResult {
  const ConnectionResult(this.outcome, this.message, {this.serverUrl});
  final ConnectionOutcome outcome;
  final String message;
  final String? serverUrl;
  bool get connected => outcome == ConnectionOutcome.connected;
}

final class ApiClient {
  ApiClient(this._session)
    : _dio = Dio(
        BaseOptions(
          connectTimeout: const Duration(seconds: 12),
          receiveTimeout: const Duration(seconds: 20),
          sendTimeout: const Duration(seconds: 20),
          responseType: ResponseType.json,
          headers: const <String, Object>{'Accept': 'application/json'},
          validateStatus: (_) => true,
        ),
      );

  final SessionStore _session;
  final Dio _dio;
  Future<TokenModel>? _refreshInFlight;

  Future<ConnectionResult> connect(String input) async {
    late final String normalized;
    try {
      normalized = ServerUrlValidator.normalize(
        input,
        allowHttpLoopback: !kReleaseMode,
      );
    } on FormatException catch (error) {
      return ConnectionResult(ConnectionOutcome.invalidUrl, error.message);
    }

    try {
      final response = await _dio.getUri<dynamic>(
        Uri.parse('$normalized/health/ready'),
        options: Options(followRedirects: false, validateStatus: (_) => true),
      );
      final status = response.statusCode ?? 0;
      if (status >= 300 && status < 400) {
        return const ConnectionResult(
          ConnectionOutcome.invalidResponse,
          'Flare returned an invalid redirect. Check its public URL and proxy routing.',
        );
      }
      if (status < 200 || status >= 300) {
        return ConnectionResult(
          ConnectionOutcome.notReady,
          'Flare is reachable but not ready (HTTP $status).',
        );
      }
      await _session.saveServerUrl(normalized);
      return ConnectionResult(
        ConnectionOutcome.connected,
        'Connected',
        serverUrl: normalized,
      );
    } on DioException catch (error) {
      return switch (error.type) {
        DioExceptionType.connectionTimeout ||
        DioExceptionType.receiveTimeout ||
        DioExceptionType.sendTimeout => const ConnectionResult(
          ConnectionOutcome.timeout,
          'Flare server timed out.',
        ),
        DioExceptionType.badCertificate => const ConnectionResult(
          ConnectionOutcome.tlsError,
          'Flare server TLS validation failed.',
        ),
        _ => const ConnectionResult(
          ConnectionOutcome.unreachable,
          'Flare server unreachable.',
        ),
      };
    } on Object {
      return const ConnectionResult(
        ConnectionOutcome.unreachable,
        'Flare server unreachable.',
      );
    }
  }

  Future<TokenModel> login(String email, String password) async {
    final response = await postJson(
      'api/v1/auth/login',
      data: <String, Object>{'email': email.trim(), 'password': password},
      authenticated: false,
    );
    final tokens = TokenModel.fromJson(response);
    await _session.saveTokens(tokens);
    return tokens;
  }

  Future<void> logout() async {
    final refreshToken = await _session.refreshToken;
    try {
      if (refreshToken != null && refreshToken.isNotEmpty) {
        await postJson(
          'api/v1/auth/logout',
          data: <String, String>{'refreshToken': refreshToken},
        );
      }
    } on FlareApiException {
      // Local revocation is mandatory even when the server is offline.
    } finally {
      await _session.clearAuthentication();
    }
  }

  Future<Map<String, dynamic>> getJson(
    String path, {
    Map<String, Object?>? query,
  }) async {
    final value = await _request<dynamic>('GET', path, query: query);
    return _asMap(value);
  }

  Future<List<dynamic>> getList(
    String path, {
    Map<String, Object?>? query,
  }) async {
    final value = await _request<dynamic>('GET', path, query: query);
    if (value is List<dynamic>) return value;
    throw const FlareApiException('Flare returned an invalid response.');
  }

  Future<Map<String, dynamic>> postJson(
    String path, {
    Object? data,
    bool authenticated = true,
  }) async {
    final value = await _request<dynamic>(
      'POST',
      path,
      data: data,
      authenticated: authenticated,
    );
    if (value == null || value == '') return <String, dynamic>{};
    return _asMap(value);
  }

  Future<String> getValidAccessToken() async {
    final access = await _session.accessToken;
    final expiry = await _session.accessExpiry;
    if (access != null &&
        expiry != null &&
        expiry.isAfter(
          DateTime.now().toUtc().add(const Duration(minutes: 1)),
        )) {
      return access;
    }
    return (await _refresh()).accessToken;
  }

  Future<dynamic> _request<T>(
    String method,
    String path, {
    Object? data,
    Map<String, Object?>? query,
    bool authenticated = true,
    bool allowRefresh = true,
  }) async {
    final serverUrl = await _session.serverUrl;
    if (serverUrl == null) {
      throw const FlareApiException('Connect to a Flare server first.');
    }
    final headers = <String, Object>{};
    if (authenticated) {
      headers['Authorization'] = 'Bearer ${await getValidAccessToken()}';
    }

    final requestUri =
        Uri.parse(
          '$serverUrl/${path.replaceFirst(RegExp(r'^/+'), '')}',
        ).replace(
          queryParameters: query?.map(
            (key, value) => MapEntry(key, value?.toString()),
          ),
        );

    try {
      final response = await _dio.requestUri<dynamic>(
        requestUri,
        data: data,
        options: Options(
          method: method,
          headers: headers,
          validateStatus: (_) => true,
        ),
      );
      final status = response.statusCode ?? 0;
      if (status == HttpStatus.unauthorized && authenticated && allowRefresh) {
        final tokens = await _refresh(force: true);
        headers['Authorization'] = 'Bearer ${tokens.accessToken}';
        final retried = await _dio.requestUri<dynamic>(
          requestUri,
          data: data,
          options: Options(
            method: method,
            headers: headers,
            validateStatus: (_) => true,
          ),
        );
        return _read(retried);
      }
      return _read(response);
    } on FlareApiException {
      rethrow;
    } on DioException catch (error) {
      throw _mapDio(error);
    } on FormatException catch (error) {
      throw FlareApiException(
        'Flare returned an invalid response.',
        cause: error,
      );
    }
  }

  dynamic _read(Response<dynamic> response) {
    final status = response.statusCode ?? 0;
    if (status >= 200 && status < 300) return response.data;
    final problem = response.data is Map
        ? _asMap(response.data)
        : const <String, dynamic>{};
    final serverMessage =
        problem['detail']?.toString() ?? problem['title']?.toString();
    final message = switch (status) {
      HttpStatus.unauthorized =>
        serverMessage ?? 'Session expired. Sign in again.',
      HttpStatus.forbidden =>
        'You do not have permission to perform this action.',
      HttpStatus.tooManyRequests =>
        'Too many requests. Wait a moment and try again.',
      >= 500 => 'Flare server could not complete the request.',
      _ => serverMessage ?? 'The request could not be completed.',
    };
    throw FlareApiException(message, statusCode: status);
  }

  Future<TokenModel> _refresh({bool force = false}) {
    if (!force && _refreshInFlight != null) return _refreshInFlight!;
    return _refreshInFlight ??= _performRefresh().whenComplete(
      () => _refreshInFlight = null,
    );
  }

  Future<TokenModel> _performRefresh() async {
    final serverUrl = await _session.serverUrl;
    final refreshToken = await _session.refreshToken;
    if (serverUrl == null || refreshToken == null || refreshToken.isEmpty) {
      await _session.clearAuthentication();
      throw const FlareApiException(
        'Session expired. Sign in again.',
        statusCode: HttpStatus.unauthorized,
      );
    }
    try {
      final response = await _dio.postUri<dynamic>(
        Uri.parse('$serverUrl/api/v1/auth/refresh'),
        data: <String, String>{'refreshToken': refreshToken},
        options: Options(validateStatus: (_) => true),
      );
      if (response.statusCode == HttpStatus.unauthorized) {
        await _session.clearAuthentication();
        throw const FlareApiException(
          'Session expired. Sign in again.',
          statusCode: HttpStatus.unauthorized,
        );
      }
      final value = _read(response);
      final tokens = TokenModel.fromJson(_asMap(value));
      await _session.saveTokens(tokens);
      return tokens;
    } on DioException catch (error) {
      throw _mapDio(error);
    }
  }

  FlareApiException _mapDio(DioException error) => switch (error.type) {
    DioExceptionType.connectionTimeout ||
    DioExceptionType.receiveTimeout ||
    DioExceptionType.sendTimeout => FlareApiException(
      'Flare server timed out.',
      cause: error,
    ),
    DioExceptionType.badCertificate => FlareApiException(
      'Flare server TLS validation failed.',
      cause: error,
    ),
    DioExceptionType.cancel => FlareApiException(
      'Request cancelled.',
      cause: error,
    ),
    _ => FlareApiException('Flare server unreachable.', cause: error),
  };

  static Map<String, dynamic> _asMap(Object? value) {
    if (value is Map<String, dynamic>) return value;
    if (value is Map) {
      return value.map((key, item) => MapEntry(key.toString(), item));
    }
    throw const FlareApiException('Flare returned an invalid response.');
  }

  void dispose() => _dio.close(force: true);
}
