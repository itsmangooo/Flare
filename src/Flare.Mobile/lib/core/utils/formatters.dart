import 'package:intl/intl.dart';

String formatPercent(double? value, {int decimals = 0}) =>
    value == null ? '—' : '${value.toStringAsFixed(decimals)}%';

String formatBytes(num? bytes, {bool perSecond = false}) {
  if (bytes == null) return 'Unavailable';
  const units = <String>['B', 'KB', 'MB', 'GB', 'TB'];
  var amount = bytes.toDouble();
  var index = 0;
  while (amount.abs() >= 1024 && index < units.length - 1) {
    amount /= 1024;
    index++;
  }
  final digits = amount.abs() >= 100 || index == 0 ? 0 : 1;
  return '${amount.toStringAsFixed(digits)} ${units[index]}${perSecond ? '/s' : ''}';
}

String formatDuration(Duration? value) {
  if (value == null) return 'Unavailable';
  final days = value.inDays;
  final hours = value.inHours.remainder(24);
  final minutes = value.inMinutes.remainder(60);
  if (days > 0) return '${days}d ${hours}h';
  if (hours > 0) return '${hours}h ${minutes}m';
  if (minutes > 0) return '${minutes}m';
  return '${value.inSeconds}s';
}

String formatRelative(DateTime? value) {
  if (value == null) return 'Unknown';
  final delta = DateTime.now().difference(value);
  if (delta.isNegative) return 'now';
  if (delta.inSeconds < 60) return '${delta.inSeconds}s ago';
  if (delta.inMinutes < 60) return '${delta.inMinutes}m ago';
  if (delta.inHours < 24) return '${delta.inHours}h ago';
  return '${delta.inDays}d ago';
}

String formatClock(DateTime value) => DateFormat.Hm().format(value);

String formatDateTime(DateTime? value) =>
    value == null ? '—' : DateFormat('MMM d, yyyy · HH:mm').format(value);

String titleCaseAction(String value) {
  final cleaned = value.replaceAll('.', ' ').replaceAll('_', ' ').trim();
  if (cleaned.isEmpty) return 'Activity';
  return cleaned
      .split(RegExp(r'\s+'))
      .map((word) => '${word[0].toUpperCase()}${word.substring(1)}')
      .join(' ');
}
