import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
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

  @override
  Widget build(BuildContext context) => FlareScaffold(
    title: 'Settings',
    leading: const FlareBackButton(),
    body: ListView(
      padding: const EdgeInsets.only(top: 8, bottom: 30),
      children: <Widget>[
        const _SettingsHeader('ACCOUNT'),
        _SettingsRow(
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
        const FlareDivider(),
        const _SettingsHeader('SERVER'),
        _SettingsRow(
          icon: PhosphorIconsRegular.hardDrives,
          title: _serverUrl,
          subtitle: _connectivity,
          subtitleColor: _connectivity == 'Connected'
              ? FlareColors.success
              : _connectivity == 'Unreachable'
              ? FlareColors.danger
              : FlareColors.warning,
          trailing: FlareButton(
            label: 'Reconnect',
            compact: true,
            loading: _checking,
            onPressed: _checking ? null : _reconnect,
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: 10),
          child: FlareButton(
            label: 'Change server',
            icon: PhosphorIconsRegular.arrowsLeftRight,
            expand: true,
            onPressed: _changeServer,
          ),
        ),
        const FlareDividerBlock(),
        const _SettingsHeader('APPEARANCE'),
        const _SettingsRow(
          icon: PhosphorIconsRegular.moon,
          title: 'Flare dark',
          subtitle: 'Purpose-built high-contrast infrastructure theme',
          trailing: FlareStatusBadge(
            label: 'Active',
            tone: FlareStatusTone.success,
            dot: false,
          ),
        ),
        const FlareDividerBlock(),
        const _SettingsHeader('ABOUT'),
        _ValueRow(label: 'Mobile version', value: _mobileVersion),
        _ValueRow(label: 'API version', value: _serverInfo?.apiVersion ?? '—'),
        _ValueRow(
          label: 'Server version',
          value: _serverInfo?.serverVersion ?? '—',
        ),
      ],
    ),
  );
}

final class _SettingsHeader extends StatelessWidget {
  const _SettingsHeader(this.label);
  final String label;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 14, bottom: 9),
    child: Text(
      label,
      style: FlareType.label.copyWith(color: FlareColors.accent),
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
  });
  final PhosphorIconData icon;
  final String title;
  final String subtitle;
  final Widget trailing;
  final Color? subtitleColor;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12),
    child: Row(
      children: <Widget>[
        Container(
          width: 38,
          height: 38,
          decoration: BoxDecoration(
            color: FlareColors.surface,
            borderRadius: BorderRadius.circular(9),
            border: Border.all(color: FlareColors.border),
          ),
          alignment: Alignment.center,
          child: PhosphorIcon(icon, size: 18, color: FlareColors.textSecondary),
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
                style: FlareType.body.copyWith(fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 2),
              Text(
                subtitle,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: FlareType.metadata.copyWith(
                  color: subtitleColor ?? FlareColors.muted,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 10),
        trailing,
      ],
    ),
  );
}

final class _ValueRow extends StatelessWidget {
  const _ValueRow({required this.label, required this.value});
  final String label;
  final String value;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 7),
    child: Row(
      children: <Widget>[
        Expanded(
          child: Text(
            label,
            style: FlareType.metadata.copyWith(color: FlareColors.muted),
          ),
        ),
        Text(
          value,
          style: FlareType.metadata.copyWith(color: FlareColors.text),
        ),
      ],
    ),
  );
}

final class FlareDividerBlock extends StatelessWidget {
  const FlareDividerBlock({super.key});
  @override
  Widget build(BuildContext context) =>
      const Padding(padding: EdgeInsets.only(top: 18), child: FlareDivider());
}
