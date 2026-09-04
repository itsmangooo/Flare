import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_brand.dart';
import '../../design/components/flare_controls.dart';

final class ConnectPage extends ConsumerStatefulWidget {
  const ConnectPage({super.key});
  @override
  ConsumerState<ConnectPage> createState() => _ConnectPageState();
}

final class _ConnectPageState extends ConsumerState<ConnectPage> {
  final _server = TextEditingController(
    text: kDebugMode ? 'https://flare.vjecni.dev' : '',
  );
  bool _connecting = false;
  String? _error;

  @override
  void dispose() {
    _server.dispose();
    super.dispose();
  }

  Future<void> _connect() async {
    if (_connecting) return;
    TextInput.finishAutofillContext();
    setState(() {
      _connecting = true;
      _error = null;
    });
    try {
      final result = await ref.read(apiClientProvider).connect(_server.text);
      if (!mounted) return;
      if (result.connected) {
        ref.invalidate(startupDestinationProvider);
        context.go('/login');
      } else {
        setState(() => _error = result.message);
      }
    } on Object {
      if (mounted) setState(() => _error = 'Flare server unreachable.');
    } finally {
      if (mounted) setState(() => _connecting = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    backgroundColor: FlareColors.background,
    body: SafeArea(
      child: LayoutBuilder(
        builder: (context, constraints) => SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: constraints.maxHeight - 48),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 460),
                child: AutofillGroup(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: <Widget>[
                      const FlareMark(size: 68)
                          .animate()
                          .fadeIn(duration: 220.ms)
                          .slideY(begin: 0.08, end: 0),
                      const SizedBox(height: 28),
                      Text(
                        'CONNECT',
                        style: FlareType.label.copyWith(
                          color: FlareColors.accent,
                          letterSpacing: 2,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Connect to Flare',
                        style: FlareType.display.copyWith(fontSize: 32),
                      ),
                      const SizedBox(height: 9),
                      Text(
                        'Enter the public HTTPS address of your self-hosted Flare API.',
                        style: FlareType.body.copyWith(
                          color: FlareColors.textSecondary,
                        ),
                      ),
                      const SizedBox(height: 30),
                      FlareTextField(
                        controller: _server,
                        label: 'Server URL',
                        hint: 'https://flare.example.com',
                        error: _error,
                        keyboardType: TextInputType.url,
                        textInputAction: TextInputAction.done,
                        autofillHints: const <String>[AutofillHints.url],
                        leading: PhosphorIconsRegular.globe,
                        onSubmitted: (_) => _connect(),
                      ),
                      const SizedBox(height: 18),
                      FlareButton(
                        label: _connecting ? 'Connecting' : 'Connect',
                        icon: PhosphorIconsRegular.arrowRight,
                        tone: FlareButtonTone.primary,
                        loading: _connecting,
                        expand: true,
                        onPressed: _connecting ? null : _connect,
                      ),
                      const SizedBox(height: 14),
                      Row(
                        children: <Widget>[
                          const PhosphorIcon(
                            PhosphorIconsRegular.lockKey,
                            size: 14,
                            color: FlareColors.muted,
                          ),
                          const SizedBox(width: 7),
                          Expanded(
                            child: Text(
                              'TLS certificates are always validated.',
                              style: FlareType.metadata.copyWith(
                                color: FlareColors.muted,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );
}
