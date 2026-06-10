import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/session_cubit.dart';
import '../config/app_config.dart';
import 'home_screen.dart';
import 'name_entry_screen.dart';

class SessionGate extends StatelessWidget {
  const SessionGate({super.key, required this.config});

  final AppConfig config;

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<SessionCubit, SessionState>(
      builder: (context, session) {
        if (session.loading) {
          return const Scaffold(
            body: Center(child: CircularProgressIndicator()),
          );
        }
        if (session.needsName || session.playerId == null) {
          return const NameEntryScreen();
        }
        return HomeScreen(config: config);
      },
    );
  }
}
