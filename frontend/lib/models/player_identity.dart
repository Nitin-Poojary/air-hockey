class PlayerIdentity {
  const PlayerIdentity({required this.playerId, required this.displayName});

  final String playerId;
  final String displayName;

  factory PlayerIdentity.fromJson(Map<String, dynamic> json) {
    final id = json['playerID'] as String?;
    if (id == null || id.isEmpty) {
      throw const FormatException('PlayerIdentity: missing playerID');
    }
    return PlayerIdentity(
      playerId: id,
      displayName: json['displayName'] as String? ?? '',
    );
  }
}
