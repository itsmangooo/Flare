import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:phosphor_icons/phosphor_icons.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/flare_theme.dart';
import '../../core/utils/formatters.dart';
import '../../design/components/flare_controls.dart';
import '../../design/components/flare_feedback.dart';
import '../../design/components/flare_scaffold.dart';
import '../../design/components/flare_state.dart';
import 'deployments_page.dart';

final class DeploymentDetailPage extends ConsumerStatefulWidget {
  const DeploymentDetailPage({required this.uuid, super.key});
  final String uuid;
  @override
  ConsumerState<DeploymentDetailPage> createState() =>
      _DeploymentDetailPageState();
}

final class _DeploymentDetailPageState
    extends ConsumerState<DeploymentDetailPage> {
  DeploymentModel? _deployment;
  Object? _error;
  bool _loading = true;
  bool _redeploying = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final json = await ref
          .read(apiClientProvider)
          .getJson(
            'api/v1/coolify/deployments/${Uri.encodeComponent(widget.uuid)}',
          );
      if (mounted) setState(() => _deployment = DeploymentModel.fromJson(json));
    } on Object catch (error) {
      if (mounted) setState(() => _error = error);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _redeploy() async {
    final deployment = _deployment;
    if (deployment == null || deployment.resourceUuid.isEmpty || _redeploying) {
      return;
    }
    final confirmed = await FlareConfirmSheet.show(
      context,
      title: 'Redeploy application?',
      subject: deployment.resourceName,
      message:
          'Coolify will queue a new deployment from the configured source.',
      confirmLabel: 'Redeploy',
    );
    if (!confirmed || !mounted || _redeploying) return;
    setState(() => _redeploying = true);
    await safeHaptic();
    try {
      await ref
          .read(apiClientProvider)
          .postJson(
            'api/v1/coolify/applications/${Uri.encodeComponent(deployment.resourceUuid)}/redeploy',
          );
      if (!mounted) return;
      ref.invalidate(deploymentsProvider);
      FlareToast.show(
        context,
        'Redeployment queued.',
        tone: FlareToastTone.success,
      );
    } on FlareApiException catch (error) {
      if (mounted) {
        FlareToast.show(context, error.message, tone: FlareToastTone.error);
      }
    } on Object {
      if (mounted) {
        FlareToast.show(
          context,
          'Flare could not queue this deployment.',
          tone: FlareToastTone.error,
        );
      }
    } finally {
      if (mounted) setState(() => _redeploying = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final deployment = _deployment;
    final palette = context.flare;
    final status = deploymentStatus(deployment?.status, palette);
    return FlareScaffold(
      eyebrow: 'DEPLOYMENT',
      title: deployment?.resourceName ?? 'Deployment',
      subtitle: deployment?.commitMessage ?? 'Release details and output',
      leading: const FlareBackButton(),
      actions: <Widget>[
        if (deployment != null)
          FlareStatusBadge(label: status.$3, tone: status.$1),
      ],
      body: _loading && deployment == null
          ? const FlareLoading(label: 'Reading deployment')
          : _error != null && deployment == null
          ? FlareErrorState(error: _error!, onRetry: _load)
          : deployment == null
          ? const FlareEmptyState(
              title: 'Deployment unavailable',
              message: 'Coolify no longer reports this deployment.',
            )
          : ListView(
              padding: const EdgeInsets.only(top: 7, bottom: 28),
              children: <Widget>[
                if (deployment.resourceUuid.isNotEmpty)
                  Align(
                    alignment: Alignment.centerLeft,
                    child: FlareButton(
                      label: 'Redeploy',
                      icon: PhosphorIconsRegular.arrowClockwise,
                      tone: FlareButtonTone.primary,
                      loading: _redeploying,
                      onPressed: _redeploying ? null : _redeploy,
                    ),
                  ),
                const SizedBox(height: 24),
                const FlareSectionHeader(title: 'Timeline'),
                const SizedBox(height: 10),
                FlareCard(
                  padding: const EdgeInsets.symmetric(horizontal: 15),
                  child: Column(
                    children: <Widget>[
                      _TimelineRow(
                        label: 'Started',
                        value: formatDateTime(deployment.startedAt),
                        color: palette.info,
                      ),
                      const FlareDivider(indent: 6),
                      _TimelineRow(
                        label: 'Finished',
                        value: formatDateTime(deployment.finishedAt),
                        color: status.$2,
                      ),
                      const FlareDivider(indent: 6),
                      _TimelineRow(
                        label: 'Duration',
                        value: formatDuration(deployment.duration),
                        color: palette.textSecondary,
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 24),
                const FlareSectionHeader(title: 'Source'),
                const SizedBox(height: 10),
                FlareCard(
                  padding: const EdgeInsets.symmetric(horizontal: 15),
                  child: Column(
                    children: <Widget>[
                      _DetailRow(
                        label: 'Branch',
                        value: deployment.branch ?? 'Unavailable',
                      ),
                      const FlareDivider(),
                      _DetailRow(
                        label: 'Commit',
                        value: deployment.commit ?? 'Unavailable',
                        mono: true,
                      ),
                      if (deployment.commitMessage != null) ...<Widget>[
                        const FlareDivider(),
                        _DetailRow(
                          label: 'Message',
                          value: deployment.commitMessage!,
                        ),
                      ],
                    ],
                  ),
                ),
                const SizedBox(height: 24),
                const FlareSectionHeader(title: 'Deployment logs'),
                const SizedBox(height: 10),
                Container(
                  constraints: const BoxConstraints(
                    minHeight: 170,
                    maxHeight: 440,
                  ),
                  padding: const EdgeInsets.all(13),
                  decoration: BoxDecoration(
                    color: const Color(0xFF05070A),
                    borderRadius: BorderRadius.circular(FlareRadii.small),
                    border: Border.all(color: palette.borderStrong),
                  ),
                  child: SingleChildScrollView(
                    child: SelectableText(
                      deployment.logs?.trim().isNotEmpty == true
                          ? deployment.logs!
                          : 'No deployment log output is available.',
                      style: FlareType.mono,
                    ),
                  ),
                ),
              ],
            ),
    );
  }
}

final class _TimelineRow extends StatelessWidget {
  const _TimelineRow({
    required this.label,
    required this.value,
    required this.color,
  });
  final String label;
  final String value;
  final Color color;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 13),
    child: Row(
      children: <Widget>[
        Container(
          width: 7,
          height: 7,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        const SizedBox(width: 10),
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

final class _DetailRow extends StatelessWidget {
  const _DetailRow({
    required this.label,
    required this.value,
    this.mono = false,
  });
  final String label;
  final String value;
  final bool mono;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 13),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        SizedBox(
          width: 74,
          child: Text(
            label,
            style: FlareType.metadata.copyWith(color: context.flare.muted),
          ),
        ),
        Expanded(
          child: Text(
            value,
            textAlign: TextAlign.right,
            style: mono
                ? FlareType.mono.copyWith(color: context.flare.textSecondary)
                : FlareType.metadata.copyWith(color: context.flare.text),
          ),
        ),
      ],
    ),
  );
}
