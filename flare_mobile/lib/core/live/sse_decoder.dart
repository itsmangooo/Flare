import 'dart:async';

final class SseEvent {
  const SseEvent(this.name, this.data);

  final String name;
  final String data;
}

final class SseEventDecoder extends StreamTransformerBase<String, SseEvent> {
  const SseEventDecoder();

  @override
  Stream<SseEvent> bind(Stream<String> stream) async* {
    var name = 'message';
    final data = <String>[];
    await for (final line in stream) {
      if (line.isEmpty) {
        if (data.isNotEmpty) yield SseEvent(name, data.join('\n'));
        name = 'message';
        data.clear();
        continue;
      }
      if (line.startsWith(':')) continue;
      final separator = line.indexOf(':');
      final field = separator < 0 ? line : line.substring(0, separator);
      var value = separator < 0 ? '' : line.substring(separator + 1);
      if (value.startsWith(' ')) value = value.substring(1);
      if (field == 'event') {
        name = value;
      } else if (field == 'data') {
        data.add(value);
      }
    }
    if (data.isNotEmpty) yield SseEvent(name, data.join('\n'));
  }
}
