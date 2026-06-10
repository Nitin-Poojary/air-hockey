import 'package:air_hockey_frontend/api/air_hockey_api.dart';
import 'package:air_hockey_frontend/app.dart';
import 'package:air_hockey_frontend/config/app_config.dart';
import 'package:air_hockey_frontend/models/player_identity.dart';
import 'package:air_hockey_frontend/storage/player_storage.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeApi extends AirHockeyApi {
  _FakeApi() : super(AppConfig());

  @override
  Future<PlayerIdentity> validatePlayer(String playerId) async {
    return const PlayerIdentity(
      playerId: 'test-player-id',
      displayName: 'Test Player',
    );
  }
}

class _FakeStorage extends PlayerStorage {
  @override
  Future<CachedPlayer?> read() async {
    return const CachedPlayer(
      playerId: 'test-player-id',
      displayName: 'Test Player',
    );
  }

  @override
  Future<void> save(String playerId, String displayName) async {}

  @override
  Future<void> clear() async {}
}

void main() {
  testWidgets('App builds with cached player', (WidgetTester tester) async {
    final config = AppConfig();
    await tester.pumpWidget(
      AirHockeyApp(
        config: config,
        api: _FakeApi(),
        playerStorage: _FakeStorage(),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.byType(MaterialApp), findsOneWidget);
    expect(find.text('Air Hockey'), findsOneWidget);
    expect(find.text('Test Player'), findsOneWidget);
  });
}
