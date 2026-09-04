import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';

import '../../core/theme/flare_theme.dart';
import 'flare_controls.dart';

final class FlareMetricCard extends StatelessWidget {
  const FlareMetricCard({
    required this.title,
    required this.value,
    required this.secondary,
    this.chart,
    this.usage,
    this.accent = FlareColors.accent,
    super.key,
  });

  final String title;
  final String value;
  final String secondary;
  final List<double?>? chart;
  final double? usage;
  final Color accent;

  @override
  Widget build(BuildContext context) => FlareCard(
    padding: const EdgeInsets.all(14),
    child: SizedBox(
      height: 126,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(title.toUpperCase(), style: FlareType.label),
          const SizedBox(height: 9),
          Text(value, style: FlareType.metric),
          const SizedBox(height: 5),
          Text(
            secondary,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: FlareType.metadata.copyWith(
              fontSize: 10.5,
              color: FlareColors.muted,
            ),
          ),
          const Spacer(),
          if (chart != null)
            FlareSparkline(values: chart!, color: accent, height: 38)
          else
            FlareUsageBar(value: usage, color: accent),
        ],
      ),
    ),
  );
}

final class FlareSparkline extends StatelessWidget {
  const FlareSparkline({
    required this.values,
    this.color = FlareColors.accent,
    this.height = 48,
    super.key,
  });
  final List<double?> values;
  final Color color;
  final double height;

  @override
  Widget build(BuildContext context) {
    final spots = <FlSpot>[];
    for (var index = 0; index < values.length; index++) {
      final value = values[index];
      if (value != null) {
        spots.add(FlSpot(index.toDouble(), value.clamp(0, 100).toDouble()));
      }
    }
    if (spots.length < 2) {
      return SizedBox(
        height: height,
        child: CustomPaint(painter: _UnavailableChartPainter()),
      );
    }
    return SizedBox(
      height: height,
      child: LineChart(
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOutCubic,
        LineChartData(
          minY: 0,
          maxY: 100,
          minX: 0,
          maxX: (values.length - 1).toDouble(),
          clipData: const FlClipData.all(),
          borderData: FlBorderData(show: false),
          gridData: const FlGridData(show: false),
          titlesData: const FlTitlesData(show: false),
          lineTouchData: const LineTouchData(enabled: false),
          lineBarsData: <LineChartBarData>[
            LineChartBarData(
              spots: spots,
              isCurved: true,
              curveSmoothness: 0.28,
              color: color,
              barWidth: 2,
              dotData: const FlDotData(show: false),
              belowBarData: BarAreaData(
                show: true,
                gradient: LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: <Color>[
                    color.withValues(alpha: 0.18),
                    color.withValues(alpha: 0),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _UnavailableChartPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = FlareColors.borderStrong
      ..strokeWidth = 1;
    canvas.drawLine(
      Offset(0, size.height * 0.72),
      Offset(size.width, size.height * 0.72),
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

final class FlareUsageBar extends StatelessWidget {
  const FlareUsageBar({
    required this.value,
    this.color = FlareColors.accent,
    super.key,
  });
  final double? value;
  final Color color;
  @override
  Widget build(BuildContext context) => ClipRRect(
    borderRadius: BorderRadius.circular(2),
    child: SizedBox(
      height: 3,
      child: LayoutBuilder(
        builder: (context, constraints) => Stack(
          children: <Widget>[
            Positioned.fill(child: ColoredBox(color: FlareColors.border)),
            AnimatedContainer(
              duration: const Duration(milliseconds: 220),
              curve: Curves.easeOutCubic,
              width:
                  constraints.maxWidth *
                  ((value ?? 0).clamp(0, 100).toDouble() / 100),
              color: color,
            ),
          ],
        ),
      ),
    ),
  );
}
