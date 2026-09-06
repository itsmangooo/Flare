import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_feedback.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';

final class NotificationPreferencesPage extends ConsumerWidget {
  const NotificationPreferencesPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final preferences = ref.watch(notificationPreferencesProvider);
    return FlareScaffold(
      eyebrow: 'ALERTS',
      title: 'Notification delivery',
      leading: const FlareBackButton(),
      body: preferences.when(
        loading: () => const FlareLoading(label: 'Reading preferences'),
        error: (error, _) => FlareErrorState(
          error: error,
          onRetry: () => ref.invalidate(notificationPreferencesProvider),
        ),
        data: (value) => _PreferencesEditor(
          key: ValueKey(value.toJson().toString()),
          initial: value,
        ),
      ),
    );
  }
}

final class _PreferencesEditor extends ConsumerStatefulWidget {
  const _PreferencesEditor({required this.initial, super.key});

  final NotificationPreferencesModel initial;

  @override
  ConsumerState<_PreferencesEditor> createState() => _PreferencesEditorState();
}

final class _PreferencesEditorState extends ConsumerState<_PreferencesEditor> {
  late bool _enabled;
  late AlertSeverity _minimumSeverity;
  late bool _recoveryEnabled;
  late bool _dockerEnabled;
  late bool _coolifyEnabled;
  late bool _cloudflareEnabled;
  late bool _hostEnabled;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    final value = widget.initial;
    _enabled = value.enabled;
    _minimumSeverity = value.minimumSeverity;
    _recoveryEnabled = value.recoveryEnabled;
    _dockerEnabled = value.dockerEnabled;
    _coolifyEnabled = value.coolifyEnabled;
    _cloudflareEnabled = value.cloudflareEnabled;
    _hostEnabled = value.hostEnabled;
  }

  NotificationPreferencesModel get _value => NotificationPreferencesModel(
    enabled: _enabled,
    minimumSeverity: _minimumSeverity,
    recoveryEnabled: _recoveryEnabled,
    dockerEnabled: _dockerEnabled,
    coolifyEnabled: _coolifyEnabled,
    cloudflareEnabled: _cloudflareEnabled,
    hostEnabled: _hostEnabled,
  );

  Future<void> _save() async {
    if (_saving) return;
    setState(() => _saving = true);
    try {
      await ref
          .read(apiClientProvider)
          .putJson('api/v1/alerts/preferences', data: _value.toJson());
      ref.invalidate(notificationPreferencesProvider);
      if (mounted) {
        FlareToast.show(
          context,
          'Notification preferences saved.',
          tone: FlareToastTone.success,
        );
      }
    } on FlareApiException catch (error) {
      if (mounted) {
        FlareToast.show(context, error.message, tone: FlareToastTone.error);
      }
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Notification preferences could not be saved.',
          tone: FlareToastTone.error,
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return ListView(
      padding: const EdgeInsets.only(top: 8, bottom: 30),
      children: <Widget>[
        FlareCard(
          child: _PreferenceSwitch(
            icon: PhosphorIconsRegular.bellRinging,
            title: 'External delivery',
            subtitle: 'Persist alert history even when delivery is disabled.',
            value: _enabled,
            onChanged: (value) => setState(() => _enabled = value),
          ),
        ),
        const SizedBox(height: 18),
        Text(
          'MINIMUM SEVERITY',
          style: FlareType.label.copyWith(color: palette.accent),
        ),
        const SizedBox(height: 9),
        SegmentedButton<AlertSeverity>(
          segments: const <ButtonSegment<AlertSeverity>>[
            ButtonSegment(value: AlertSeverity.info, label: Text('Info')),
            ButtonSegment(value: AlertSeverity.warning, label: Text('Warning')),
            ButtonSegment(
              value: AlertSeverity.critical,
              label: Text('Critical'),
            ),
          ],
          selected: <AlertSeverity>{_minimumSeverity},
          showSelectedIcon: false,
          onSelectionChanged: (selection) =>
              setState(() => _minimumSeverity = selection.single),
        ),
        const SizedBox(height: 18),
        Text(
          'BEHAVIOR',
          style: FlareType.label.copyWith(color: palette.accent),
        ),
        const SizedBox(height: 9),
        FlareCard(
          child: _PreferenceSwitch(
            icon: PhosphorIconsRegular.checkCircle,
            title: 'Recovery notifications',
            subtitle: 'Notify when an active condition becomes healthy.',
            value: _recoveryEnabled,
            onChanged: (value) => setState(() => _recoveryEnabled = value),
          ),
        ),
        const SizedBox(height: 18),
        Text('SOURCES', style: FlareType.label.copyWith(color: palette.accent)),
        const SizedBox(height: 9),
        FlareCard(
          padding: const EdgeInsets.symmetric(horizontal: 14),
          child: Column(
            children: <Widget>[
              _PreferenceSwitch(
                icon: PhosphorIconsRegular.cube,
                title: 'Docker',
                subtitle: 'Containers and Docker daemon',
                value: _dockerEnabled,
                onChanged: (value) => setState(() => _dockerEnabled = value),
              ),
              const FlareDivider(),
              _PreferenceSwitch(
                icon: PhosphorIconsRegular.rocketLaunch,
                title: 'Coolify',
                subtitle: 'Resources and deployments',
                value: _coolifyEnabled,
                onChanged: (value) => setState(() => _coolifyEnabled = value),
              ),
              const FlareDivider(),
              _PreferenceSwitch(
                icon: PhosphorIconsRegular.cloud,
                title: 'Cloudflare',
                subtitle: 'Domains, TLS and tunnels',
                value: _cloudflareEnabled,
                onChanged: (value) =>
                    setState(() => _cloudflareEnabled = value),
              ),
              const FlareDivider(),
              _PreferenceSwitch(
                icon: PhosphorIconsRegular.cpu,
                title: 'Host',
                subtitle: 'CPU and memory thresholds',
                value: _hostEnabled,
                onChanged: (value) => setState(() => _hostEnabled = value),
              ),
            ],
          ),
        ),
        const SizedBox(height: 20),
        FlareButton(
          label: 'Save preferences',
          icon: PhosphorIconsRegular.floppyDisk,
          expand: true,
          tone: FlareButtonTone.primary,
          loading: _saving,
          onPressed: _saving ? null : _save,
        ),
        const SizedBox(height: 8),
        Text(
          'Administrator access is required to save changes.',
          textAlign: TextAlign.center,
          style: FlareType.metadata.copyWith(color: palette.muted),
        ),
      ],
    );
  }
}

final class _PreferenceSwitch extends StatelessWidget {
  const _PreferenceSwitch({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  final PhosphorIconData icon;
  final String title;
  final String subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = context.flare;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        children: <Widget>[
          PhosphorIcon(icon, size: 19, color: palette.textSecondary),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                Text(
                  title,
                  style: FlareType.body.copyWith(
                    color: palette.text,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: FlareType.metadata.copyWith(color: palette.muted),
                ),
              ],
            ),
          ),
          Switch.adaptive(
            value: value,
            activeThumbColor: palette.accent,
            onChanged: onChanged,
          ),
        ],
      ),
    );
  }
}
