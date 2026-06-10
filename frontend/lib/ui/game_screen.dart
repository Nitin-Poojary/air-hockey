import 'package:flame/game.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/matchmaking_cubit.dart';
import '../config/app_config.dart';
import '../game/air_hockey_game.dart';
import '../models/game_models.dart';
import '../network/state_interpolator.dart';

class GameScreen extends StatefulWidget {
  const GameScreen({
    super.key,
    required this.config,
    required this.playerId,
    required this.match,
  });

  final AppConfig config;
  final String playerId;
  final MatchReadyPayload match;

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  late final AirHockeyGame _game;

  @override
  void initState() {
    super.initState();
    _game = AirHockeyGame(
      match: widget.match,
      playerId: widget.playerId,
      config: widget.config,
      interpolator: StateInterpolator(),
      onWinner: _onWinner,
    );
  }

  void _onWinner(String winnerId, String winnerName) {
    if (!mounted) return;
    final youWon = winnerId == widget.playerId;
    final opponentLabel = winnerName.isNotEmpty ? winnerName : 'Opponent';
    showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => AlertDialog(
        title: Text(
          youWon ? '🏆 You win!' : '$opponentLabel wins!',
        ),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              Navigator.of(context).pop();
            },
            child: const Text('Back to menu'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, result) async {
        if (didPop) return;
        final shouldLeave = await _showLeaveDialog();
        if (shouldLeave && mounted) {
          Navigator.of(context).pop();
        }
      },
      child: SafeArea(
        child: Scaffold(
          // GameWidget handles all input routing internally via Flame's event system.
          // No Listener or GestureDetector needed — DraggablePaddle gets drag events
          // directly because it has the DragCallbacks mixin.
          body: GameWidget(game: _game),
        ),
      ),
    );
  }

  Future<bool> _showLeaveDialog() async {
    return await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: const Text('Leave Match'),
            content: const Text('Are you sure you want to leave the match?'),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(false),
                child: const Text('Cancel'),
              ),
              TextButton(
                onPressed: () async {
                  await context.read<MatchmakingCubit>().leave(widget.playerId);
                  if (mounted) {
                    Navigator.of(ctx).pop(true);
                  }
                },
                child: const Text('Leave'),
              ),
            ],
          ),
        ) ??
        false;
  }

  @override
  void dispose() {
    _game.pauseEngine();
    super.dispose();
  }
}
