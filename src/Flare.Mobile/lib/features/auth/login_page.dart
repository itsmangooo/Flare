import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../design/components/flare_brand.dart';
import '../../design/components/flare_controls.dart';

final class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});
  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

final class _LoginPageState extends ConsumerState<LoginPage> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  String _server = 'Flare server';
  String? _error;
  bool _signingIn = false;

  @override
  void initState() {
    super.initState();
    _loadServer();
  }

  Future<void> _loadServer() async {
    final value = await ref.read(sessionStoreProvider).serverUrl;
    if (mounted && value != null) {
      setState(() => _server = Uri.parse(value).host);
    }
  }

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    if (_signingIn) return;
    if (!_email.text.contains('@') || _password.text.isEmpty) {
      setState(() => _error = 'Enter your email and password.');
      return;
    }
    TextInput.finishAutofillContext();
    setState(() {
      _signingIn = true;
      _error = null;
    });
    try {
      await ref.read(apiClientProvider).login(_email.text, _password.text);
      if (!mounted) return;
      ref.invalidate(startupDestinationProvider);
      context.go('/overview');
    } on FlareApiException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } on Object {
      if (mounted) setState(() => _error = 'Flare could not sign you in.');
    } finally {
      if (mounted) setState(() => _signingIn = false);
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
                      const Row(
                        children: <Widget>[
                          FlareMark(size: 40),
                          SizedBox(width: 12),
                          Text(
                            'FLARE',
                            style: TextStyle(
                              color: FlareColors.text,
                              fontWeight: FontWeight.w700,
                              letterSpacing: 3,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 34),
                      Text(
                        'Welcome back',
                        style: FlareType.display.copyWith(fontSize: 32),
                      ),
                      const SizedBox(height: 7),
                      Row(
                        children: <Widget>[
                          Container(
                            width: 7,
                            height: 7,
                            decoration: const BoxDecoration(
                              color: FlareColors.success,
                              shape: BoxShape.circle,
                            ),
                          ),
                          const SizedBox(width: 7),
                          Text(_server, style: FlareType.metadata),
                        ],
                      ),
                      const SizedBox(height: 30),
                      FlareTextField(
                        controller: _email,
                        label: 'Email',
                        hint: 'admin@example.com',
                        keyboardType: TextInputType.emailAddress,
                        textInputAction: TextInputAction.next,
                        autofillHints: const <String>[
                          AutofillHints.username,
                          AutofillHints.email,
                        ],
                        leading: PhosphorIconsRegular.envelope,
                      ),
                      const SizedBox(height: 17),
                      FlareTextField(
                        controller: _password,
                        label: 'Password',
                        hint: 'Your password',
                        obscureText: true,
                        error: _error,
                        textInputAction: TextInputAction.done,
                        autofillHints: const <String>[AutofillHints.password],
                        leading: PhosphorIconsRegular.lock,
                        onSubmitted: (_) => _login(),
                      ),
                      const SizedBox(height: 20),
                      FlareButton(
                        label: _signingIn ? 'Signing in' : 'Sign in',
                        icon: PhosphorIconsRegular.signIn,
                        tone: FlareButtonTone.primary,
                        loading: _signingIn,
                        expand: true,
                        onPressed: _signingIn ? null : _login,
                      ),
                      const SizedBox(height: 14),
                      Align(
                        child: InkWell(
                          onTap: _signingIn
                              ? null
                              : () async {
                                  await ref
                                      .read(sessionStoreProvider)
                                      .clearServer();
                                  if (context.mounted) context.go('/connect');
                                },
                          child: Padding(
                            padding: const EdgeInsets.all(8),
                            child: Text(
                              'Use a different server',
                              style: FlareType.metadata.copyWith(
                                color: FlareColors.textSecondary,
                              ),
                            ),
                          ),
                        ),
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
