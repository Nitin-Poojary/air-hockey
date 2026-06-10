import 'package:shared_preferences/shared_preferences.dart';

class CachedPlayer {
  const CachedPlayer({required this.playerId, required this.displayName});

  final String playerId;
  final String displayName;
}

class PlayerStorage {
  static const _keyPlayerId = 'player_id';
  static const _keyDisplayName = 'display_name';

  Future<CachedPlayer?> read() async {
    final prefs = await SharedPreferences.getInstance();
    final playerId = prefs.getString(_keyPlayerId);
    final displayName = prefs.getString(_keyDisplayName);
    if (playerId == null || playerId.isEmpty) return null;
    return CachedPlayer(
      playerId: playerId,
      displayName: displayName ?? '',
    );
  }

  Future<void> save(String playerId, String displayName) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyPlayerId, playerId);
    await prefs.setString(_keyDisplayName, displayName);
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyPlayerId);
    await prefs.remove(_keyDisplayName);
  }
}
