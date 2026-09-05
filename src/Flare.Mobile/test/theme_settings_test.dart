import 'package:flare_mobile/core/theme/flare_theme.dart';
import 'package:flare_mobile/core/theme/theme_settings.dart';
import 'package:flare_mobile/design/components/flare_controls.dart';
import 'package:flare_mobile/design/components/flare_scaffold.dart';
import 'package:flare_mobile/features/settings/settings_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues(<String, Object>{}));

  test('defaults preserve the existing dark Flare appearance', () async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    final settings = await container.read(themeSettingsProvider.future);

    expect(settings.mode, FlareThemeMode.dark);
    expect(settings.accent, FlareAccent.blue);
  });

  test('theme mode and accent persist across provider containers', () async {
    final first = ProviderContainer();
    await first.read(themeSettingsProvider.future);
    await first
        .read(themeSettingsProvider.notifier)
        .setMode(FlareThemeMode.oled);
    await first
        .read(themeSettingsProvider.notifier)
        .setAccent(FlareAccent.violet);
    first.dispose();

    final second = ProviderContainer();
    addTearDown(second.dispose);
    final restored = await second.read(themeSettingsProvider.future);

    expect(restored.mode, FlareThemeMode.oled);
    expect(restored.accent, FlareAccent.violet);
  });

  test('palettes keep semantic colors separate from selectable accents', () {
    final light = buildFlareTheme(
      brightness: Brightness.light,
      accent: FlareAccent.emerald,
    ).extension<FlarePalette>()!;
    final oled = buildFlareTheme(
      accent: FlareAccent.amber,
      oled: true,
    ).extension<FlarePalette>()!;

    expect(light.background, const Color(0xFFF7F8FA));
    expect(light.surface, Colors.white);
    expect(light.accent, FlareAccent.emerald.color);
    expect(oled.background, Colors.black);
    expect(oled.accent, FlareAccent.amber.color);
    expect(FlareColors.success, isNot(light.accent));
    expect(FlareColors.danger, isNot(oled.accent));
  });

  testWidgets('custom surfaces consume the active light palette', (
    tester,
  ) async {
    final theme = buildFlareTheme(
      brightness: Brightness.light,
      accent: FlareAccent.cyan,
    );
    final palette = theme.extension<FlarePalette>()!;
    await tester.pumpWidget(
      MaterialApp(
        theme: theme,
        home: const FlareScaffold(
          title: 'Theme test',
          body: FlareCard(child: Text('Surface')),
        ),
      ),
    );

    final scaffoldSurface = tester.widget<DecoratedBox>(
      find
          .descendant(
            of: find.byType(Scaffold),
            matching: find.byType(DecoratedBox),
          )
          .first,
    );
    final card = tester.widget<AnimatedContainer>(
      find.byType(AnimatedContainer).first,
    );

    expect(
      (scaffoldSurface.decoration as BoxDecoration).color,
      palette.background,
    );
    expect((card.decoration as BoxDecoration).color, palette.surface);
  });

  testWidgets('settings exposes custom theme and accent selectors', (
    tester,
  ) async {
    PackageInfo.setMockInitialValues(
      appName: 'Flare',
      packageName: 'io.github.itsmangooo.flare',
      version: '1.2.0',
      buildNumber: '6',
      buildSignature: '',
    );
    final container = ProviderContainer();
    addTearDown(container.dispose);
    await container.read(themeSettingsProvider.future);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: buildFlareTheme(),
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.pump(const Duration(seconds: 5));

    await tester.tap(find.text('Dark'));
    await tester.pumpAndSettle();
    expect(find.text('Theme'), findsOneWidget);
    await tester.tap(find.text('Light'));
    await tester.pumpAndSettle();
    expect(
      container.read(themeSettingsProvider).value!.mode,
      FlareThemeMode.light,
    );

    await tester.ensureVisible(find.text('Flare Blue'));
    await tester.tap(find.text('Flare Blue'));
    await tester.pumpAndSettle();
    expect(find.text('Accent'), findsOneWidget);
    await tester.tap(find.text('Violet'));
    await tester.pumpAndSettle();
    expect(
      container.read(themeSettingsProvider).value!.accent,
      FlareAccent.violet,
    );
  });
}
