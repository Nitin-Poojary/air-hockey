import '../models/game_models.dart';

class _StampedState {
  _StampedState(this.timeSec, this.state);

  final double timeSec;
  final GameStatePayload state;
}

/// Buffers recent authoritative snapshots and returns a smoothed state for
/// rendering (opponent paddle + puck). Local paddle is reconciled separately.
class StateInterpolator {
  StateInterpolator({this.bufferSize = 24, this.renderDelaySec = 0.085});

  final int bufferSize;
  final double renderDelaySec;

  final List<_StampedState> _buf = [];

  void push(GameStatePayload state, double nowSec) {
    _buf.add(_StampedState(nowSec, state));
    while (_buf.length > bufferSize) {
      _buf.removeAt(0);
    }
  }

  void clear() => _buf.clear();

  /// Target simulation time (seconds) we want to display.
  double targetTime(double nowSec) => nowSec - renderDelaySec;

  GameStatePayload? interpolate(double nowSec) {
    if (_buf.isEmpty) return null;
    final t = targetTime(nowSec);
    if (t <= _buf.first.timeSec) return _buf.first.state;
    if (t >= _buf.last.timeSec) return _buf.last.state;

    for (var i = 0; i < _buf.length - 1; i++) {
      final a = _buf[i];
      final b = _buf[i + 1];
      if (t >= a.timeSec && t <= b.timeSec) {
        final span = b.timeSec - a.timeSec;
        final u = span <= 1e-6 ? 1.0 : (t - a.timeSec) / span;
        return _lerpState(a.state, b.state, u);
      }
    }
    return _buf.last.state;
  }
}

GameStatePayload _lerpState(GameStatePayload a, GameStatePayload b, double u) {
  return GameStatePayload(
    player1: _lerpPaddle(a.player1, b.player1, u),
    player2: _lerpPaddle(a.player2, b.player2, u),
    score: u < 0.5 ? a.score : b.score,
    puck: _lerpPuck(a.puck, b.puck, u),
    board: a.board,
    roundEnded: b.roundEnded,
    roundWinner: b.roundWinner,
  );
}

PaddlePayload _lerpPaddle(PaddlePayload a, PaddlePayload b, double u) {
  return PaddlePayload(
    playerId: u < 0.5 ? a.playerId : b.playerId,
    position: a.position.lerp(b.position, u),
    velocity: a.velocity,
    radius: a.radius,
  );
}

PuckPayload _lerpPuck(PuckPayload a, PuckPayload b, double u) {
  return PuckPayload(
    position: a.position.lerp(b.position, u),
    velocity: a.velocity,
    radius: a.radius,
  );
}
