import 'package:flutter/material.dart';

import '../../core/theme/flare_theme.dart';

final class FlareMark extends StatelessWidget {
  const FlareMark({this.size = 58, super.key});
  final double size;
  @override
  Widget build(BuildContext context) => Semantics(
    label: 'Flare',
    image: true,
    child: SizedBox.square(
      dimension: size,
      child: CustomPaint(painter: _FlareMarkPainter()),
    ),
  );
}

final class _FlareMarkPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final scale = size.width / 64;
    canvas.scale(scale, scale);
    final outer = Path()
      ..moveTo(35, 3)
      ..cubicTo(39, 17, 52, 22, 52, 38)
      ..cubicTo(52, 51, 43, 60, 31, 60)
      ..cubicTo(19, 60, 11, 51, 11, 40)
      ..cubicTo(11, 29, 18, 22, 29, 11)
      ..cubicTo(28, 21, 33, 23, 35, 3)
      ..close();
    final inner = Path()
      ..moveTo(33, 27)
      ..cubicTo(35, 36, 42, 38, 41, 46)
      ..cubicTo(40, 53, 35, 56, 30, 56)
      ..cubicTo(24, 56, 20, 52, 20, 46)
      ..cubicTo(20, 40, 24, 36, 33, 27)
      ..close();
    canvas.drawPath(outer, Paint()..color = FlareColors.accent);
    canvas.drawPath(inner, Paint()..color = const Color(0xFFFFA07F));
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
