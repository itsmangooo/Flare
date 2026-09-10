import 'package:flare_mobile/core/api/server_url_validator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ServerUrlValidator', () {
    test('normalizes a public HTTPS origin', () {
      expect(
        ServerUrlValidator.normalize(' flare.example.com/ '),
        'https://flare.example.com',
      );
    });

    test('rejects public HTTP', () {
      expect(
        () => ServerUrlValidator.normalize('http://flare.example.com'),
        throwsA(isA<FormatException>()),
      );
    });

    test('permits emulator loopback HTTP only when explicitly enabled', () {
      expect(
        ServerUrlValidator.normalize(
          'http://10.0.2.2:8080',
          allowHttpLoopback: true,
        ),
        'http://10.0.2.2:8080',
      );
      expect(
        () => ServerUrlValidator.normalize('http://10.0.2.2:8080'),
        throwsA(isA<FormatException>()),
      );
    });

    test('rejects API paths, credentials, query, and fragments', () {
      for (final value in <String>[
        'https://flare.example.com/api/v1',
        'https://user:secret@flare.example.com',
        'https://flare.example.com?redirect=x',
        'https://flare.example.com#fragment',
      ]) {
        expect(
          () => ServerUrlValidator.normalize(value),
          throwsA(isA<FormatException>()),
        );
      }
    });
  });
}
