import 'dart:async';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/models.dart';

final class SessionStore {
  SessionStore({FlutterSecureStorage? secureStorage})
    : _secure =
          secureStorage ??
          const FlutterSecureStorage(aOptions: AndroidOptions());

  static const _serverKey = 'flare.server_url';
  static const _accessKey = 'flare.access_token';
  static const _accessExpiryKey = 'flare.access_expiry';
  static const _refreshKey = 'flare.refresh_token';
  static const _refreshExpiryKey = 'flare.refresh_expiry';
  static const _emailKey = 'flare.user_email';
  static const _rolesKey = 'flare.user_roles';

  final FlutterSecureStorage _secure;
  final StreamController<void> _changes = StreamController<void>.broadcast();

  Stream<void> get changes => _changes.stream;

  Future<String?> get serverUrl async =>
      (await SharedPreferences.getInstance()).getString(_serverKey);
  Future<String?> get userEmail async =>
      (await SharedPreferences.getInstance()).getString(_emailKey);
  Future<List<String>> get roles async =>
      (await SharedPreferences.getInstance()).getStringList(_rolesKey) ??
      const <String>[];
  Future<String?> get accessToken => _readSecure(_accessKey);
  Future<String?> get refreshToken => _readSecure(_refreshKey);
  Future<DateTime?> get accessExpiry async =>
      DateTime.tryParse(await _readSecure(_accessExpiryKey) ?? '');
  Future<DateTime?> get refreshExpiry async =>
      DateTime.tryParse(await _readSecure(_refreshExpiryKey) ?? '');

  Future<void> saveServerUrl(String value) async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(
      _serverKey,
      value.replaceFirst(RegExp(r'/+$'), ''),
    );
    _changes.add(null);
  }

  Future<void> saveTokens(TokenModel tokens) async {
    await Future.wait(<Future<void>>[
      _secure.write(key: _accessKey, value: tokens.accessToken),
      _secure.write(
        key: _accessExpiryKey,
        value: tokens.accessTokenExpiresAt.toUtc().toIso8601String(),
      ),
      _secure.write(key: _refreshKey, value: tokens.refreshToken),
      _secure.write(
        key: _refreshExpiryKey,
        value: tokens.refreshTokenExpiresAt.toUtc().toIso8601String(),
      ),
    ]);
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_emailKey, tokens.user.email);
    await preferences.setStringList(_rolesKey, tokens.user.roles);
    _changes.add(null);
  }

  Future<bool> hasUsableSession() async {
    try {
      final token = await refreshToken;
      final expiry = await refreshExpiry;
      return token != null &&
          token.isNotEmpty &&
          expiry != null &&
          expiry.isAfter(DateTime.now().toUtc());
    } on Object {
      await clearAuthentication();
      return false;
    }
  }

  Future<void> clearAuthentication() async {
    await Future.wait(<Future<void>>[
      _secure.delete(key: _accessKey),
      _secure.delete(key: _accessExpiryKey),
      _secure.delete(key: _refreshKey),
      _secure.delete(key: _refreshExpiryKey),
    ]);
    final preferences = await SharedPreferences.getInstance();
    await preferences.remove(_emailKey);
    await preferences.remove(_rolesKey);
    _changes.add(null);
  }

  Future<void> clearServer() async {
    await clearAuthentication();
    final preferences = await SharedPreferences.getInstance();
    await preferences.remove(_serverKey);
    _changes.add(null);
  }

  Future<String?> _readSecure(String key) async {
    try {
      return await _secure.read(key: key);
    } on Object {
      await _secure.deleteAll();
      return null;
    }
  }

  void dispose() => _changes.close();
}
