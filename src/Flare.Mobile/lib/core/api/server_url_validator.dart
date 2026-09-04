final class ServerUrlValidator {
  const ServerUrlValidator._();

  static String normalize(String input, {bool allowHttpLoopback = false}) {
    final trimmed = input.trim();
    final candidate = trimmed.contains('://') ? trimmed : 'https://$trimmed';
    final uri = Uri.tryParse(candidate);
    if (uri == null ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment) {
      throw const FormatException('Enter a valid Flare server URL.');
    }
    final loopback =
        uri.host == 'localhost' ||
        uri.host == '127.0.0.1' ||
        uri.host == '10.0.2.2';
    if (uri.scheme != 'https' &&
        !(allowHttpLoopback && loopback && uri.scheme == 'http')) {
      throw const FormatException('Flare requires an HTTPS server URL.');
    }
    if (uri.path != '' && uri.path != '/') {
      throw const FormatException('Use the server origin without an API path.');
    }
    return uri
        .replace(path: '', query: null, fragment: null)
        .toString()
        .replaceFirst(RegExp(r'/+$'), '');
  }
}
