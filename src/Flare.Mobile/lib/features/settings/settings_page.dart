import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../core/theme/theme_settings.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_feedback.dart';
import '../../design/components/flare_scaffold.dart';

final class SettingsPage extends ConsumerStatefulWidget {
  const SettingsPage({super.key});

  @override
  ConsumerState<SettingsPage> createState() => _SettingsPageState();
}

final class _SettingsPageState extends ConsumerState<SettingsPage> {
  String _email = 'Unknown account';
  String _serverUrl = 'Not configured';
  String _connectivity = 'Checking';
  String _mobileVersion = '—';
  ServerInfoModel? _serverInfo;
  bool _checking = false;
  bool _signingOut = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final session = ref.read(sessionStoreProvider);
      final info = await PackageInfo.fromPlatform();
      final values = await Future.wait<String?>(<Future<String?>>[
        session.userEmail,
        session.serverUrl,
      ]);
      if (!mounted) return;
      setState(() {
        _email = values[0] ?? 'Unknown account';
        _serverUrl = values[1] ?? 'Not configured';
        _mobileVersion = '${info.version} (${info.buildNumber})';
      });
      await _reconnect();
    } on Object {
      if (mounted) setState(() => _connectivity = 'Unavailable');
    }
  }

  Future<void> _reconnect() async {
    if (_checking) return;
    setState(() {
      _checking = true;
      _connectivity = 'Checking';
    });
    try {
      final json = await ref
          .read(apiClientProvider)
          .getJson('api/v1/system/info');
      if (mounted) {
        setState(() {
          _serverInfo = ServerInfoModel.fromJson(json);
          _connectivity = 'Connected';
        });
      }
    } on FlareApiException catch (error) {
      if (mounted) {
        setState(() => _connectivity = 'Unreachable');
        FlareToast.show(context, error.message, tone: FlareToastTone.error);
      }
    } on Object {
      if (mounted) {
        setState(() => _connectivity = 'Unreachable');
        FlareToast.show(
          context,
          'Flare server unreachable.',
          tone: FlareToastTone.error,
        );
      }
    } finally {
      if (mounted) setState(() => _checking = false);
    }
  }

  Future<void> _logout() async {
    if (_signingOut) return;
    final confirmed = await FlareConfirmSheet.show(
      context,
      title: 'Log out?',
      subject: _email,
      message: 'The secure session tokens will be removed from this device.',
      confirmLabel: 'Log out',
      destructive: true,
    );
    if (!confirmed || !mounted) return;
    setState(() => _signingOut = true);
    try {
      await ref.read(telemetryServiceProvider).stop();
      await ref.read(apiClientProvider).logout();
    } on Object {
      await ref.read(sessionStoreProvider).clearAuthentication();
    } finally {
      if (mounted) {
        ref.invalidate(startupDestinationProvider);
        context.go('/login');
      }
    }
  }

  Future<void> _changeServer() async {
    final confirmed = await FlareConfirmSheet.show(
      context,
      title: 'Change server?',
      subject: _serverUrl,
      message:
          'This signs out and removes the current server address from this device.',
      confirmLabel: 'Change server',
      destructive: true,
    );
    if (!confirmed || !mounted) return;
    try {
      await ref.read(telemetryServiceProvider).stop();
      await ref.read(sessionStoreProvider).clearServer();
      if (mounted) {
        ref.invalidate(startupDestinationProvider);
        context.go('/connect');
      }
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'The server could not be changed.',
          tone: FlareToastTone.error,
        );
      }
    }
  }

  Future<void> _chooseTheme(FlareThemeSettings settings) async {
    final selected = await FlareBottomSheet.show<FlareThemeMode>(
      context,
      barrierLabel: 'Dismiss theme selection',
      child: _ChoiceSheet<FlareThemeMode>(
        title: 'Theme',
        selected: settings.mode,
        items: const <_Choice<FlareThemeMode>>[
          _Choice(FlareThemeMode.system, 'System', 'Follow Android appearance'),
          _Choice(
            FlareThemeMode.light,
            'Light',
            'Clean neutral light surfaces',
          ),
          _Choice(FlareThemeMode.dark, 'Dark', 'Flare charcoal surfaces'),
          _Choice(FlareThemeMode.oled, 'OLED', 'True-black primary background'),
        ],
      ),
    );
    if (selected == null || !mounted) return;
    try {
      await ref.read(themeSettingsProvider.notifier).setMode(selected);
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Theme preference could not be saved.',
          tone: FlareToastTone.error,
        );
      }
    }
  }

  Future<void> _chooseAccent(FlareThemeSettings settings) async {
    final selected = await FlareBottomSheet.show<FlareAccent>(
      context,
      barrierLabel: 'Dismiss accent selection',
      child: _ChoiceSheet<FlareAccent>(
        title: 'Accent',
        selected: settings.accent,
        items: FlareAccent.values
            .map(
              (accent) => _Choice<FlareAccent>(
                accent,
                accent.label,
                'Navigation, controls and chart emphasis',
                color: accent.color,
              ),
            )
            .toList(growable: false),
      ),
    );
    if (selected == null || !mounted) return;
    try {
      await ref.read(themeSettingsProvider.notifier).setAccent(selected);
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Accent preference could not be saved.',
          tone: FlareToastTone.error,
        );
      }
    }
  }

  Future<void> _chooseAtmosphere(FlareThemeSettings settings) async {
    final selected = await FlareBottomSheet.show<FlareAtmosphere>(
      context,
      barrierLabel: 'Dismiss atmosphere selection',
      child: _ChoiceSheet<FlareAtmosphere>(
        title: 'Atmosphere',
        selected: settings.atmosphere,
        items: FlareAtmosphere.values
            .map(
              (atmosphere) => _Choice<FlareAtmosphere>(
                atmosphere,
                atmosphere.label,
                'Subtle background and glass ambience',
              ),
            )
            .toList(growable: false),
      ),
    );
    if (selected == null || !mounted) return;
    try {
      await ref.read(themeSettingsProvider.notifier).setAtmosphere(selected);
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Atmosphere preference could not be saved.',
          tone: FlareToastTone.error,
        );
      }
    }
  }

  Future<void> _choosePreset(FlareThemeSettings settings) async {
    final applied = await FlareBottomSheet.show<bool>(
      context,
      barrierLabel: 'Dismiss theme presets',
      child: _ThemePresetSheet(original: settings),
    );
    if (applied == true || !mounted) return;
    ref.read(themeSettingsProvider.notifier).preview(settings);
  }

  @override
  Widget build(BuildContext context) {
    final settings =
        ref.watch(themeSettingsProvider).value ?? const FlareThemeSettings();
    final palette = context.flare;
    return FlareScaffold(
      title: 'Settings',
      subtitle: 'Personalize and manage Flare',
      leading: const FlareBackButton(),
      body: ListView(
        padding: const EdgeInsets.only(
          top: FlareSpace.xs,
          bottom: FlareSpace.xxl,
        ),
        children: <Widget>[
          const _SettingsHeader('ACCOUNT'),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(horizontal: FlareSpace.md),
            child: _SettingsRow(
              icon: PhosphorIconsRegular.userCircle,
              title: _email,
              subtitle: 'Administrator session',
              trailing: FlareButton(
                label: 'Log out',
                compact: true,
                tone: FlareButtonTone.danger,
                loading: _signingOut,
                onPressed: _signingOut ? null : _logout,
              ),
            ),
          ),
          const _SettingsHeader('SERVER'),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(horizontal: FlareSpace.md),
            child: Column(
              children: <Widget>[
                _SettingsRow(
                  icon: PhosphorIconsRegular.sparkle,
                  title: 'Theme preset',
                  subtitle: 'Preview a curated Flare look',
                  trailing: PhosphorIcon(
                    PhosphorIconsRegular.caretRight,
                    size: 16,
                    color: palette.muted,
                  ),
                  onTap: () => _choosePreset(settings),
                ),
                const FlareDivider(indent: 50),
                _SettingsRow(
                  icon: PhosphorIconsRegular.hardDrives,
                  title: _serverUrl,
                  subtitle: _connectivity,
                  subtitleColor: _connectivity == 'Connected'
                      ? palette.success
                      : _connectivity == 'Unreachable'
                      ? palette.danger
                      : palette.warning,
                  trailing: FlareButton(
                    label: 'Reconnect',
                    compact: true,
                    loading: _checking,
                    onPressed: _checking ? null : _reconnect,
                  ),
                ),
                const FlareDivider(indent: 50),
                _SettingsRow(
                  icon: PhosphorIconsRegular.arrowsLeftRight,
                  title: 'Change server',
                  subtitle: 'Connect Flare to another API',
                  trailing: PhosphorIcon(
                    PhosphorIconsRegular.caretRight,
                    size: 16,
                    color: palette.muted,
                  ),
                  onTap: _changeServer,
                ),
              ],
            ),
          ),
          const _SettingsHeader('APPEARANCE'),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(horizontal: FlareSpace.md),
            child: Column(
              children: <Widget>[
                _SettingsRow(
                  icon: PhosphorIconsRegular.circleHalfTilt,
                  title: 'Appearance',
                  subtitle: 'System, Light, Dark or OLED',
                  trailing: _TrailingValue(value: _themeLabel(settings.mode)),
                  onTap: () => _chooseTheme(settings),
                ),
                const FlareDivider(indent: 50),
                _SettingsRow(
                  icon: PhosphorIconsRegular.palette,
                  title: 'Accent',
                  subtitle: settings.accent.label,
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: <Widget>[
                      Container(
                        width: 18,
                        height: 18,
                        decoration: BoxDecoration(
                          color: settings.accent.color,
                          shape: BoxShape.circle,
                        ),
                      ),
                      const SizedBox(width: FlareSpace.xs),
                      PhosphorIcon(
                        PhosphorIconsRegular.caretRight,
                        size: 15,
                        color: palette.muted,
                      ),
                    ],
                  ),
                  onTap: () => _chooseAccent(settings),
                ),
                const FlareDivider(indent: 50),
                _SettingsRow(
                  icon: PhosphorIconsRegular.gradient,
                  title: 'Atmosphere',
                  subtitle: settings.atmosphere.label,
                  trailing: _TrailingValue(value: settings.atmosphere.label),
                  onTap: () => _chooseAtmosphere(settings),
                ),
              ],
            ),
          ),
          const _SettingsHeader('INTERACTIONS'),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(horizontal: FlareSpace.md),
            child: _SettingsRow(
              icon: PhosphorIconsRegular.vibrate,
              title: 'Haptics',
              subtitle: 'Subtle feedback for controls and actions',
              trailing: FlareSwitch(
                value: settings.hapticsEnabled,
                semanticLabel: 'Haptics',
                onChanged: (enabled) async {
                  await ref
                      .read(themeSettingsProvider.notifier)
                      .setHapticsEnabled(enabled);
                  if (enabled) await safeSelectionHaptic();
                },
              ),
            ),
          ),
          const _SettingsHeader('NOTIFICATIONS'),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(horizontal: FlareSpace.md),
            child: _SettingsRow(
              icon: PhosphorIconsRegular.bellRinging,
              title: 'Alert delivery',
              subtitle: 'Severity, recoveries and infrastructure sources',
              trailing: PhosphorIcon(
                PhosphorIconsRegular.caretRight,
                size: 16,
                color: palette.muted,
              ),
              onTap: () => context.push('/settings/notifications'),
            ),
          ),
          const _SettingsHeader('ABOUT'),
          FlareGroupedSurface(
            padding: const EdgeInsets.symmetric(
              horizontal: FlareSpace.md,
              vertical: FlareSpace.xxs,
            ),
            child: Column(
              children: <Widget>[
                _ValueRow(label: 'Mobile version', value: _mobileVersion),
                const FlareDivider(),
                _ValueRow(
                  label: 'API version',
                  value: _serverInfo?.apiVersion ?? '—',
                ),
                const FlareDivider(),
                _ValueRow(
                  label: 'Server version',
                  value: _serverInfo?.serverVersion ?? '—',
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

String _themeLabel(FlareThemeMode mode) => switch (mode) {
  FlareThemeMode.system => 'System',
  FlareThemeMode.light => 'Light',
  FlareThemeMode.dark => 'Dark',
  FlareThemeMode.oled => 'OLED',
};

final class _SettingsHeader extends StatelessWidget {
  const _SettingsHeader(this.label);
  final String label;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(
      top: FlareSpace.lg,
      left: FlareSpace.xxs,
      bottom: FlareSpace.xs,
    ),
    child: Text(
      label,
      style: FlareType.caption.copyWith(
        color: context.flare.textSecondary,
        letterSpacing: 1.1,
      ),
    ),
  );
}

final class _SettingsRow extends StatelessWidget {
  const _SettingsRow({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.trailing,
    this.subtitleColor,
    this.onTap,
  });
  final PhosphorIconData icon;
  final String title;
  final String subtitle;
  final Widget trailing;
  final Color? subtitleColor;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final content = Padding(
      padding: const EdgeInsets.symmetric(vertical: FlareSpace.sm),
      child: Row(
        children: <Widget>[
          Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: palette.accent.withValues(alpha: 0.11),
              borderRadius: BorderRadius.circular(FlareRadii.small),
            ),
            alignment: Alignment.center,
            child: PhosphorIcon(icon, size: 18, color: palette.accent),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: FlareType.body.copyWith(
                    fontWeight: FontWeight.w600,
                    color: palette.text,
                  ),
                ),
                const SizedBox(height: 2),
                Row(
                  children: <Widget>[
                    if (subtitleColor != null) ...<Widget>[
                      Container(
                        width: 6,
                        height: 6,
                        decoration: BoxDecoration(
                          color: subtitleColor,
                          shape: BoxShape.circle,
                        ),
                      ),
                      const SizedBox(width: FlareSpace.xs),
                    ],
                    Expanded(
                      child: Text(
                        subtitle,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: FlareType.metadata.copyWith(
                          color: subtitleColor ?? palette.muted,
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(width: FlareSpace.sm),
          trailing,
        ],
      ),
    );
    return onTap == null
        ? content
        : Semantics(
            button: true,
            child: InkWell(onTap: onTap, child: content),
          );
  }
}

final class _ValueRow extends StatelessWidget {
  const _ValueRow({required this.label, required this.value});
  final String label;
  final String value;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12),
    child: Row(
      children: <Widget>[
        Expanded(
          child: Text(
            label,
            style: FlareType.metadata.copyWith(color: context.flare.muted),
          ),
        ),
        Text(
          value,
          style: FlareType.metadata.copyWith(color: context.flare.text),
        ),
      ],
    ),
  );
}

final class _TrailingValue extends StatelessWidget {
  const _TrailingValue({required this.value});
  final String value;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: <Widget>[
      Text(
        value,
        style: FlareType.metadata.copyWith(color: context.flare.textSecondary),
      ),
      const SizedBox(width: FlareSpace.xs),
      PhosphorIcon(
        PhosphorIconsRegular.caretRight,
        size: 15,
        color: context.flare.muted,
      ),
    ],
  );
}

final class _Choice<T> {
  const _Choice(this.value, this.label, this.description, {this.color});
  final T value;
  final String label;
  final String description;
  final Color? color;
}

final class _ChoiceSheet<T> extends StatelessWidget {
  const _ChoiceSheet({
    required this.title,
    required this.selected,
    required this.items,
  });
  final String title;
  final T selected;
  final List<_Choice<T>> items;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(title, style: FlareType.title.copyWith(color: palette.text)),
        const SizedBox(height: FlareSpace.sm),
        ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.sizeOf(context).height * 0.62,
          ),
          child: ListView.separated(
            shrinkWrap: true,
            itemCount: items.length,
            separatorBuilder: (context, index) => const FlareDivider(),
            itemBuilder: (context, index) => InkWell(
              onTap: () => Navigator.pop(context, items[index].value),
              child: ConstrainedBox(
                constraints: const BoxConstraints(minHeight: 56),
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: FlareSpace.sm),
                  child: Row(
                    children: <Widget>[
                      if (items[index].color case final color?) ...<Widget>[
                        Container(
                          width: 20,
                          height: 20,
                          decoration: BoxDecoration(
                            color: color,
                            shape: BoxShape.circle,
                            border: Border.all(color: palette.borderStrong),
                          ),
                        ),
                        const SizedBox(width: FlareSpace.sm),
                      ],
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: <Widget>[
                            Text(
                              items[index].label,
                              style: FlareType.body.copyWith(
                                color: palette.text,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            const SizedBox(height: 2),
                            Text(
                              items[index].description,
                              style: FlareType.metadata.copyWith(
                                color: palette.muted,
                              ),
                            ),
                          ],
                        ),
                      ),
                      if (items[index].value == selected)
                        PhosphorIcon(
                          PhosphorIconsRegular.check,
                          size: 18,
                          color: palette.accent,
                        ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

final class _ThemePresetSheet extends ConsumerStatefulWidget {
  const _ThemePresetSheet({required this.original});
  final FlareThemeSettings original;

  @override
  ConsumerState<_ThemePresetSheet> createState() => _ThemePresetSheetState();
}

final class _ThemePresetSheetState extends ConsumerState<_ThemePresetSheet> {
  FlareThemePreset? _selected;
  bool _saving = false;

  void _preview(FlareThemePreset preset) {
    setState(() => _selected = preset);
    ref
        .read(themeSettingsProvider.notifier)
        .preview(preset.applyTo(widget.original));
  }

  Future<void> _apply() async {
    final selected = _selected;
    if (selected == null || _saving) return;
    setState(() => _saving = true);
    try {
      await ref
          .read(themeSettingsProvider.notifier)
          .save(selected.applyTo(widget.original));
      if (mounted) Navigator.pop(context, true);
    } on Object {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    final height = MediaQuery.sizeOf(context).height * 0.58;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          'Theme presets',
          style: FlareType.title.copyWith(color: palette.text),
        ),
        const SizedBox(height: FlareSpace.xxs),
        Text(
          'Tap to preview live. Apply only when it feels right.',
          style: FlareType.metadata.copyWith(color: palette.muted),
        ),
        const SizedBox(height: FlareSpace.md),
        SizedBox(
          height: height.clamp(320, 520),
          child: GridView.builder(
            itemCount: FlareThemePreset.values.length,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              crossAxisSpacing: FlareSpace.sm,
              mainAxisSpacing: FlareSpace.sm,
              childAspectRatio: 1.42,
            ),
            itemBuilder: (context, index) {
              final preset = FlareThemePreset.values[index];
              final preview = preset.applyTo(widget.original);
              return _PresetPreviewCard(
                preset: preset,
                settings: preview,
                selected: preset == _selected,
                onTap: () => _preview(preset),
              );
            },
          ),
        ),
        const SizedBox(height: FlareSpace.md),
        Row(
          children: <Widget>[
            Expanded(
              child: FlareButton(
                label: 'Cancel',
                expand: true,
                onPressed: () => Navigator.pop(context, false),
              ),
            ),
            const SizedBox(width: FlareSpace.sm),
            Expanded(
              child: FlareButton(
                label: 'Apply',
                expand: true,
                loading: _saving,
                tone: FlareButtonTone.primary,
                onPressed: _selected == null ? null : _apply,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

final class _PresetPreviewCard extends StatelessWidget {
  const _PresetPreviewCard({
    required this.preset,
    required this.settings,
    required this.selected,
    required this.onTap,
  });
  final FlareThemePreset preset;
  final FlareThemeSettings settings;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final baseDark = settings.mode != FlareThemeMode.light;
    final background = baseDark
        ? const Color(0xFF0B0E13)
        : const Color(0xFFF4F6F9);
    final surface = baseDark ? const Color(0xFF171C23) : Colors.white;
    final foreground = baseDark ? Colors.white : const Color(0xFF1A1D22);
    final accent = settings.accent.color;
    final ambient = switch (settings.atmosphere) {
      FlareAtmosphere.none => Colors.transparent,
      FlareAtmosphere.softGradient => accent.withValues(alpha: 0.18),
      FlareAtmosphere.aurora => const Color(0x5530C99A),
      FlareAtmosphere.midnight => const Color(0x554866D4),
      FlareAtmosphere.graphite => const Color(0x40798A98),
    };
    return Semantics(
      button: true,
      selected: selected,
      label: preset.label,
      child: InkWell(
        borderRadius: BorderRadius.circular(FlareRadii.large),
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          padding: const EdgeInsets.all(FlareSpace.sm),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: <Color>[
                Color.alphaBlend(ambient, background),
                background,
              ],
            ),
            borderRadius: BorderRadius.circular(FlareRadii.large),
            border: Border.all(
              width: selected ? 1.5 : 1,
              color: selected ? accent : foreground.withValues(alpha: 0.1),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Row(
                children: <Widget>[
                  Container(
                    width: 24,
                    height: 6,
                    decoration: BoxDecoration(
                      color: accent,
                      borderRadius: BorderRadius.circular(99),
                    ),
                  ),
                  const Spacer(),
                  if (selected)
                    PhosphorIcon(
                      PhosphorIconsRegular.check,
                      size: 14,
                      color: accent,
                    ),
                ],
              ),
              const Spacer(),
              Container(
                height: 22,
                decoration: BoxDecoration(
                  color: surface,
                  borderRadius: BorderRadius.circular(FlareRadii.small),
                  border: Border.all(color: foreground.withValues(alpha: 0.06)),
                ),
              ),
              const SizedBox(height: FlareSpace.xs),
              Text(
                preset.label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: FlareType.metadata.copyWith(
                  color: foreground,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class FlareDividerBlock extends StatelessWidget {
  const FlareDividerBlock({super.key});
  @override
  Widget build(BuildContext context) => const Padding(
    padding: EdgeInsets.only(top: FlareSpace.md),
    child: FlareDivider(),
  );
}
