import 'dart:convert';

class MatchReadyPayload {
  MatchReadyPayload({required this.gameId, required this.state});

  final String gameId;
  final GameStatePayload state;

  factory MatchReadyPayload.fromJson(Map<String, dynamic> json) {
    final id = json['GameID'] as String? ?? json['gameID'] as String?;
    if (id == null) {
      throw const FormatException('Match payload missing GameID');
    }
    final rawState = json['State'] ?? json['state'];
    if (rawState is! Map<String, dynamic>) {
      throw const FormatException('Match payload missing State');
    }
    return MatchReadyPayload(
      gameId: id,
      state: GameStatePayload.fromJson(rawState),
    );
  }
}

class GameStatePayload {
  GameStatePayload({
    required this.player1,
    required this.player2,
    required this.score,
    required this.puck,
    required this.board,
    this.roundEnded = false,
    this.roundWinner = '',
  });

  final PaddlePayload player1;
  final PaddlePayload player2;
  final ScorePayload score;
  final PuckPayload puck;
  final BoardPayload board;
  final bool roundEnded;
  final String roundWinner;

  factory GameStatePayload.fromJson(Map<String, dynamic> json) {
    return GameStatePayload(
      player1: PaddlePayload.fromJson(json['player1'] as Map<String, dynamic>),
      player2: PaddlePayload.fromJson(json['player2'] as Map<String, dynamic>),
      score: ScorePayload.fromJson(json['score'] as Map<String, dynamic>),
      puck: PuckPayload.fromJson(json['puck'] as Map<String, dynamic>),
      board: BoardPayload.fromJson(json['board'] as Map<String, dynamic>),
      roundEnded: json['roundEnded'] as bool? ?? false,
      roundWinner: json['roundWinner'] as String? ?? '',
    );
  }

  GameStatePayload copyWith({
    PaddlePayload? player1,
    PaddlePayload? player2,
    ScorePayload? score,
    PuckPayload? puck,
    BoardPayload? board,
    bool? roundEnded,
    String? roundWinner,
  }) {
    return GameStatePayload(
      player1: player1 ?? this.player1,
      player2: player2 ?? this.player2,
      score: score ?? this.score,
      puck: puck ?? this.puck,
      board: board ?? this.board,
      roundEnded: roundEnded ?? this.roundEnded,
      roundWinner: roundWinner ?? this.roundWinner,
    );
  }
}

class BoardPayload {
  BoardPayload({required this.width, required this.height});

  final double width;
  final double height;

  factory BoardPayload.fromJson(Map<String, dynamic> json) {
    return BoardPayload(
      width: (json['width'] as num).toDouble(),
      height: (json['height'] as num).toDouble(),
    );
  }
}

class ScorePayload {
  ScorePayload({required this.player1, required this.player2});

  final int player1;
  final int player2;

  factory ScorePayload.fromJson(Map<String, dynamic> json) {
    return ScorePayload(
      player1: (json['player1'] as num).toInt(),
      player2: (json['player2'] as num).toInt(),
    );
  }
}

class VectorPayload {
  VectorPayload({required this.vx, required this.vy});

  final double vx;
  final double vy;

  factory VectorPayload.fromJson(Map<String, dynamic> json) {
    return VectorPayload(
      vx: (json['vx'] as num).toDouble(),
      vy: (json['vy'] as num).toDouble(),
    );
  }
}

class PositionPayload {
  PositionPayload({required this.x, required this.y});

  final double x;
  final double y;

  factory PositionPayload.fromJson(Map<String, dynamic> json) {
    return PositionPayload(
      x: (json['x'] as num).toDouble(),
      y: (json['y'] as num).toDouble(),
    );
  }

  PositionPayload lerp(PositionPayload other, double t) {
    return PositionPayload(x: x + (other.x - x) * t, y: y + (other.y - y) * t);
  }
}

class PaddlePayload {
  PaddlePayload({
    required this.playerId,
    required this.position,
    required this.velocity,
    required this.radius,
  });

  final String playerId;
  final PositionPayload position;
  final VectorPayload velocity;
  final double radius;

  factory PaddlePayload.fromJson(Map<String, dynamic> json) {
    return PaddlePayload(
      playerId: json['playerID'] as String,
      position: PositionPayload.fromJson(
        json['position'] as Map<String, dynamic>,
      ),
      velocity: VectorPayload.fromJson(
        json['velocity'] as Map<String, dynamic>,
      ),
      radius: (json['radius'] as num).toDouble(),
    );
  }

  PaddlePayload copyWith({PositionPayload? position}) {
    return PaddlePayload(
      playerId: playerId,
      position: position ?? this.position,
      velocity: velocity,
      radius: radius,
    );
  }
}

class PuckPayload {
  PuckPayload({
    required this.position,
    required this.velocity,
    required this.radius,
  });

  final PositionPayload position;
  final VectorPayload velocity;
  final double radius;

  factory PuckPayload.fromJson(Map<String, dynamic> json) {
    return PuckPayload(
      position: PositionPayload.fromJson(
        json['position'] as Map<String, dynamic>,
      ),
      velocity: VectorPayload.fromJson(
        json['velocity'] as Map<String, dynamic>,
      ),
      radius: (json['radius'] as num).toDouble(),
    );
  }
}

class MatchResultUdpPayload {
  MatchResultUdpPayload({
    required this.gameId,
    required this.winnerId,
    required this.winnerName,
  });

  final String gameId;
  final String winnerId;
  final String winnerName;

  factory MatchResultUdpPayload.fromJson(Map<String, dynamic> json) {
    final winnerId =
        json['winnerID'] as String? ?? json['winner'] as String? ?? '';
    return MatchResultUdpPayload(
      gameId: json['gameID'] as String,
      winnerId: winnerId,
      winnerName: json['winnerName'] as String? ?? '',
    );
  }
}

dynamic parseUdpJson(String raw) {
  final decoded = jsonDecode(raw);
  if (decoded is! Map<String, dynamic>) return null;
  final isMatchResult =
      (decoded.containsKey('winner') || decoded.containsKey('winnerID')) &&
      !decoded.containsKey('player1');
  if (isMatchResult) {
    return MatchResultUdpPayload.fromJson(decoded);
  }
  return GameStatePayload.fromJson(decoded);
}
