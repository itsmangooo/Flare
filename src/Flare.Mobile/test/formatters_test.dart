import 'package:flare_mobile/core/utils/formatters.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('formats byte rates without fabricating missing values', () {
    expect(formatBytes(1024, perSecond: true), '1.0 KB/s');
    expect(formatBytes(null, perSecond: true), 'Unavailable');
  });

  test('formats technical actions for activity display', () {
    expect(titleCaseAction('container.restart'), 'Container Restart');
    expect(titleCaseAction('login_failure'), 'Login Failure');
  });
}
