import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/matchmaking_cubit.dart';
import '../bloc/session_cubit.dart';
import '../config/app_config.dart';
import 'game_screen.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key, required this.config});

  final AppConfig config;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Air Hockey')),
      body: BlocListener<MatchmakingCubit, MatchmakingState>(
        listenWhen: (p, c) => c is MatchmakingMatched || c is MatchmakingFailed,
        listener: (context, state) {
          if (state is MatchmakingMatched) {
            final playerId = context.read<SessionCubit>().state.playerId;
            if (playerId == null) return;
            Navigator.of(context)
                .push<void>(
                  MaterialPageRoute(
                    builder: (_) => GameScreen(
                      config: config,
                      playerId: playerId,
                      match: state.payload,
                    ),
                  ),
                )
                .then((_) {
                  if (context.mounted) {
                    context.read<MatchmakingCubit>().reset();
                  }
                });
          } else if (state is MatchmakingTimeout) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: const Text('Timeout'),
                backgroundColor: Theme.of(context).colorScheme.error,
              ),
            );
          } else if (state is MatchmakingFailed) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text(state.message),
                backgroundColor: Theme.of(context).colorScheme.error,
              ),
            );
          }
        },
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: BlocBuilder<SessionCubit, SessionState>(
            builder: (context, session) {
              if (session.error != null && session.playerId != null) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'Could not reach server: ${session.error}',
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: () =>
                          context.read<SessionCubit>().initSession(),
                      child: const Text('Retry'),
                    ),
                  ],
                );
              }
              final id = session.playerId;
              final name = session.displayName;
              return Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    'Player',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 8),
                  if (name != null && name.isNotEmpty)
                    Text(
                      name,
                      style: Theme.of(context).textTheme.headlineSmall,
                    ),
                  const SizedBox(height: 8),
                  SelectableText(
                    id ?? '—',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 32),
                  BlocBuilder<MatchmakingCubit, MatchmakingState>(
                    builder: (context, match) {
                      final busy = match is MatchmakingSearching;
                      return FilledButton(
                        onPressed: id == null
                            ? null
                            : busy
                            ? () => context.read<MatchmakingCubit>().cancel()
                            : () => context.read<MatchmakingCubit>().startMatch(
                                id,
                              ),
                        child: Text(busy ? 'Cancel' : 'Start match'),
                      );
                    },
                  ),
                  const SizedBox(height: 16),
                ],
              );
            },
          ),
        ),
      ),
    );
  }
}
