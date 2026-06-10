import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/session_cubit.dart';

class NameEntryScreen extends StatefulWidget {
  const NameEntryScreen({super.key});

  @override
  State<NameEntryScreen> createState() => _NameEntryScreenState();
}

class _NameEntryScreenState extends State<NameEntryScreen> {
  final _controller = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    if (!_formKey.currentState!.validate()) return;
    context.read<SessionCubit>().resolveWithName(_controller.text);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Choose your name')),
      body: BlocBuilder<SessionCubit, SessionState>(
        builder: (context, session) {
          return Padding(
            padding: const EdgeInsets.all(24),
            child: Form(
              key: _formKey,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    'Enter a display name to play. After reinstall, the same name recovers your player.',
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 24),
                  TextFormField(
                    controller: _controller,
                    enabled: !session.loading,
                    textInputAction: TextInputAction.done,
                    onFieldSubmitted: (_) => _submit(),
                    decoration: const InputDecoration(
                      labelText: 'Display name',
                      hintText: 'e.g. Nitin',
                      border: OutlineInputBorder(),
                    ),
                    validator: (value) {
                      final trimmed = value?.trim() ?? '';
                      if (trimmed.length < 2) {
                        return 'At least 2 characters';
                      }
                      if (trimmed.length > 20) {
                        return 'At most 20 characters';
                      }
                      final valid = RegExp(r'^[a-zA-Z0-9 _-]+$');
                      if (!valid.hasMatch(trimmed)) {
                        return 'Letters, numbers, spaces, - and _ only';
                      }
                      return null;
                    },
                  ),
                  if (session.error != null) ...[
                    const SizedBox(height: 16),
                    Text(
                      session.error!,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ],
                  const SizedBox(height: 24),
                  FilledButton(
                    onPressed: session.loading ? null : _submit,
                    child: session.loading
                        ? const SizedBox(
                            height: 22,
                            width: 22,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Text('Continue'),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
