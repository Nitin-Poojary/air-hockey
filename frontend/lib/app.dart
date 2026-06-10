import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'api/air_hockey_api.dart';
import 'bloc/matchmaking_cubit.dart';
import 'bloc/session_cubit.dart';
import 'config/app_config.dart';
import 'storage/player_storage.dart';
import 'ui/session_gate.dart';

class AirHockeyApp extends StatelessWidget {
  const AirHockeyApp({
    super.key,
    required this.config,
    required this.api,
    this.playerStorage,
  });

  final AppConfig config;
  final AirHockeyApi api;
  final PlayerStorage? playerStorage;

  @override
  Widget build(BuildContext context) {
    final storage = playerStorage ?? PlayerStorage();
    return MultiBlocProvider(
      providers: [
        BlocProvider(
          create: (_) => SessionCubit(api, storage)..initSession(),
        ),
        BlocProvider(
          create: (_) => MatchmakingCubit(api),
        ),
      ],
      child: MaterialApp(
        title: 'Air Hockey',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
          useMaterial3: true,
        ),
        home: SessionGate(config: config),
      ),
    );
  }
}
