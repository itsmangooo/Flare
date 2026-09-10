import 'package:flare_mobile/core/live/sse_decoder.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('decodes named and multiline server-sent events', () async {
    final events = await Stream<String>.fromIterable(<String>[
      'retry: 3000',
      ': heartbeat',
      'event: snapshot',
      'data: {"freshness":',
      'data: "Live"}',
      '',
    ]).transform(const SseEventDecoder()).toList();

    expect(events, hasLength(1));
    expect(events.single.name, 'snapshot');
    expect(events.single.data, '{"freshness":\n"Live"}');
  });

  test('flushes a final event when the stream closes', () async {
    final events = await Stream<String>.fromIterable(<String>[
      'event: snapshot',
      'data: {}',
    ]).transform(const SseEventDecoder()).toList();

    expect(events.single.name, 'snapshot');
    expect(events.single.data, '{}');
  });
}
