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
    final signal = Path()
      ..moveTo(34, 3)
      ..lineTo(50, 28)
      ..lineTo(42, 26)
      ..lineTo(54, 43)
      ..lineTo(36, 61)
      ..lineTo(17, 55)
      ..lineTo(10, 38)
      ..lineTo(28, 14)
      ..lineTo(28, 33)
      ..lineTo(39, 21)
      ..close();
    final upperFacet = Path()
      ..moveTo(34, 3)
      ..lineTo(50, 28)
      ..lineTo(42, 26)
      ..lineTo(28, 39)
      ..lineTo(28, 14)
      ..close();
    final core = Path()
      ..moveTo(29, 32)
      ..lineTo(43, 47)
      ..lineTo(35, 57)
      ..lineTo(23, 52)
      ..lineTo(19, 43)
      ..close();
    canvas.drawPath(signal, Paint()..color = FlareColors.brandDeep);
    canvas.drawPath(upperFacet, Paint()..color = FlareColors.accent);
    canvas.drawPath(core, Paint()..color = FlareColors.brandCyan);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
