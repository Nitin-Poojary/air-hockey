import 'dart:async';
import 'dart:math' as math;

import 'package:air_hockey_frontend/udp/udp_session_io.dart';
import 'package:flame/components.dart';
import 'package:flame/events.dart';
import 'package:flame/game.dart';
import 'package:flame/text.dart';
import 'package:flutter/material.dart';

import '../config/app_config.dart';
import '../models/game_models.dart';
import '../network/state_interpolator.dart';

class DraggablePaddle extends CircleComponent with DragCallbacks {
  DraggablePaddle({
    required double radius,
    required Color color,
    required this.onDrag,
    required this.clampToZone,
    this.canDrag,
  }) : super(radius: radius, anchor: Anchor.center);

  final void Function(Vector2 serverPos) onDrag;
  final Vector2 Function(Vector2) clampToZone;
  final bool Function()? canDrag;

  @override
  bool onDragStart(DragStartEvent event) {
    if (canDrag != null && !canDrag!()) return false;
    event.continuePropagation = false;
    super.onDragStart(event);
    return true;
  }

  @override
  bool onDragUpdate(DragUpdateEvent event) {
    if (canDrag != null && !canDrag!()) return false;
    position += event.localDelta;
    position = clampToZone(position);
    onDrag(position.clone());
    return true;
  }
}

class AirHockeyGame extends FlameGame with DragCallbacks {
  AirHockeyGame({
    required this.match,
    required this.playerId,
    required this.config,
    required this.interpolator,
    required this.onWinner,
  }) : super(world: World());

  final MatchReadyPayload match;
  final String playerId;
  final AppConfig config;
  final StateInterpolator interpolator;
  final void Function(String winnerId, String winnerName) onWinner;

  late final bool _isPlayer1;
  late final CircleComponent _puck;
  late final DraggablePaddle _paddleMe;
  late final CircleComponent _paddleOpponent;
  late final TextComponent _scoreText;

  GameStatePayload? _display;
  GameStatePayload? _lastAuth;
  Vector2 _localTarget = Vector2.zero();

  UdpGameSession? _udp;
  double _sendAccum = 0;
  static const _sendInterval = 1 / 60;
  bool _ended = false;
  var _isRoundIdle = false;
  late CircleComponent _puckScaleAnim;

  void resumeAfterRound() {
    _ended = false;
    _puck.scale = Vector2.all(1.0);
  }

  void startRoundIdle() {
    _isRoundIdle = true;
    _puckScaleAnim.scale = Vector2.zero();
    _puck.scale = Vector2.zero();
    _puckScaleAnim.position = Vector2(
      _display!.board.width / 2,
      _display!.board.height / 2,
    );
  }

  Future<void> _updatePuckScaleAnim(double dt) async {
    if (!_isRoundIdle) {
      _puckScaleAnim.scale = Vector2.zero();
      return;
    }

    const animDuration = 1.5;
    const maxScale = 1.0;
    final currentScale = _puckScaleAnim.scale.x;
    if (currentScale < maxScale) {
      final newScale = (currentScale + dt / animDuration).clamp(0.0, maxScale);
      _puckScaleAnim.scale = Vector2.all(newScale);
    } else {
      _isRoundIdle = false;
      _puckScaleAnim.scale = Vector2.zero();
      // Show actual puck again
      _puck.scale = Vector2.all(1.0);
      await Future.delayed(const Duration(milliseconds: 500));
      resumeAfterRound();
    }
  }

  @override
  Future<void> onLoad() async {
    await super.onLoad();

    _isPlayer1 = match.state.player1.playerId == playerId;
    _display = match.state;
    _localTarget = Vector2(
      _myPaddle(match.state).position.x,
      _myPaddle(match.state).position.y,
    );

    final board = match.state.board;
    camera.viewfinder.visibleGameSize = Vector2(board.width, board.height);
    camera.viewfinder.position = Vector2(board.width / 2, board.height / 2);

    // Flip the entire view for Player1 so their paddle appears at screen bottom.
    // World objects still live in server coords — only rendering is flipped.
    // Flame's drag system resolves deltas through the camera, so dragging
    // still produces correct server-space deltas even after the flip.
    if (_isPlayer1) {
      camera.viewfinder.angle = math.pi;
    }

    final boardBg = RectangleComponent(
      size: Vector2(board.width, board.height),
      paint: Paint()..color = const Color(0xFF0d1b2a),
    )..priority = -100;

    final markings = BoardMarkings(size: Vector2(board.width, board.height))
      ..priority = -90;

    _puck = CircleComponent(
      radius: match.state.puck.radius,
      anchor: Anchor.center,
      paint: Paint()..color = Colors.white,
    );

    _puckScaleAnim = CircleComponent(
      radius: match.state.puck.radius,
      anchor: Anchor.center,
      paint: Paint()..color = Colors.white.withValues(alpha: 0.5),
      scale: Vector2.zero(),
    );

    _paddleMe = DraggablePaddle(
      radius: _myPaddle(match.state).radius,
      color: const Color(0xFFe63946),
      onDrag: (serverPos) => _localTarget = serverPos,
      clampToZone: (pos) => _clampToZone(pos, _display!.board),
      canDrag: () => !_isRoundIdle,
    );

    _paddleOpponent = CircleComponent(
      radius: _theirPaddle(match.state).radius,
      anchor: Anchor.center,
      paint: Paint()..color = const Color(0xFF457b9d),
    );

    _scoreText = TextComponent(
      text: '0 — 0',
      position: Vector2(16, 12),
      priority: 50,
      textRenderer: TextPaint(
        style: const TextStyle(
          color: Colors.white,
          fontSize: 22,
          fontWeight: FontWeight.bold,
        ),
      ),
    );
    camera.viewport.add(_scoreText);
    _scoreText.position = Vector2(16, 12);

    world.addAll([
      boardBg,
      markings,
      _puck,
      _puckScaleAnim,
      _paddleMe,
      _paddleOpponent,
    ]);

    _udp = UdpGameSession(
      config: config,
      gameId: match.gameId,
      playerId: playerId,
      interpolator: interpolator,
      onAuthoritative: (raw) => _lastAuth = raw,
      onState: (smooth) => _display = _mergeDisplay(smooth),
      onMatchEnd: (winnerId, winnerName) {
        if (_ended) return;
        _ended = true;
        onWinner(winnerId, winnerName);
      },
    );
    double initialX, initialY;
    if (_isPlayer1) {
      initialX = match.state.player1.position.x;
      initialY = match.state.player1.position.y;
    } else {
      initialX = match.state.player2.position.x;
      initialY = match.state.player2.position.y;
    }
    await _udp!.start(initialX, initialY);

    _syncFromState(_display!);
    _updateScoreHud();
  }

  PaddlePayload _myPaddle(GameStatePayload s) =>
      _isPlayer1 ? s.player1 : s.player2;

  PaddlePayload _theirPaddle(GameStatePayload s) =>
      _isPlayer1 ? s.player2 : s.player1;

  GameStatePayload _mergeDisplay(GameStatePayload server) {
    // During round idle, trust server completely — no local prediction
    if (_isRoundIdle) {
      return server;
    }

    final basis = _lastAuth ?? server;
    final mine = _myPaddle(basis);
    final corrected = _correctLocal(
      _localTarget.x,
      _localTarget.y,
      mine.position.x,
      mine.position.y,
    );
    final myPaddle = mine.copyWith(
      position: PositionPayload(x: corrected.x, y: corrected.y),
    );
    return _isPlayer1
        ? server.copyWith(player1: myPaddle)
        : server.copyWith(player2: myPaddle);
  }

  Vector2 _correctLocal(double lx, double ly, double ax, double ay) {
    const snap = 55.0;
    const soft = 0.18;
    final dx = lx - ax;
    final dy = ly - ay;
    if (math.sqrt(dx * dx + dy * dy) > snap) return Vector2(ax, ay);
    return Vector2(ax + dx * (1 - soft), ay + dy * (1 - soft));
  }

  void _syncFromState(GameStatePayload s) {
    _puck.position = Vector2(s.puck.position.x, s.puck.position.y);

    // Always sync paddle during round idle — ignore drag state
    if (_isRoundIdle || !_paddleMe.isDragged) {
      final me = _myPaddle(s).position;
      _paddleMe.position = Vector2(me.x, me.y);
      _localTarget = Vector2(me.x, me.y); // ← also sync localTarget
    }

    final them = _theirPaddle(s).position;
    _paddleOpponent.position = Vector2(them.x, them.y);
  }

  void _updateScoreHud() {
    final s = _display;
    if (s == null) return;
    final mine = _isPlayer1 ? s.score.player1 : s.score.player2;
    final theirs = _isPlayer1 ? s.score.player2 : s.score.player1;
    _scoreText.text = '$mine — $theirs';
  }

  Vector2 _clampToZone(Vector2 pos, BoardPayload board) {
    final r = _myPaddle(_display!).radius;
    final w = board.width;
    final h = board.height;
    final x = pos.x.clamp(r, w - r);
    final double y = _isPlayer1
        ? pos.y.clamp(r, h * 0.48)
        : pos.y.clamp(h * 0.52, h - r);
    return Vector2(x, y);
  }

  @override
  void update(double dt) {
    super.update(dt);
    if (_display == null || _ended) return;

    // Check for round ended in state
    if (_display!.roundEnded && !_isRoundIdle) {
      interpolator.clear();
      startRoundIdle();
    }

    _updatePuckScaleAnim(dt);

    if (_isRoundIdle) {
      _syncFromState(_display!);
      _updateScoreHud();
      return;
    }

    _sendAccum += dt;
    if (_sendAccum >= _sendInterval) {
      _sendAccum -= _sendInterval;
      _udp?.sendPaddle(_localTarget.x, _localTarget.y);
    }

    if (_lastAuth != null) {
      final mine = _myPaddle(_lastAuth!);
      _localTarget = _correctLocal(
        _localTarget.x,
        _localTarget.y,
        mine.position.x,
        mine.position.y,
      );
    }

    _syncFromState(_display!);
    _updateScoreHud();
  }

  @override
  void onRemove() {
    unawaited(_udp?.dispose());
    super.onRemove();
  }
}

class BoardMarkings extends PositionComponent {
  BoardMarkings({required super.size})
    : super(anchor: Anchor.topLeft, position: Vector2.zero());

  @override
  void render(Canvas canvas) {
    final w = size.x;
    final h = size.y;
    const goalLeft = 175.0;
    const goalRight = 375.0;
    const tick = 12.0;

    final mid = Paint()
      ..color = const Color(0x55FFFFFF)
      ..strokeWidth = 2;
    canvas.drawLine(Offset(0, h / 2), Offset(w, h / 2), mid);

    final mouth = Paint()
      ..color = const Color(0x88A8DADC)
      ..strokeWidth = 3
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(
      const Offset(goalLeft, 0),
      const Offset(goalRight, 0),
      mouth,
    );
    canvas.drawLine(Offset(goalLeft, h), Offset(goalRight, h), mouth);

    final post = Paint()
      ..color = const Color(0x66A8DADC)
      ..strokeWidth = 2;
    canvas.drawLine(const Offset(goalLeft, 0), Offset(goalLeft, tick), post);
    canvas.drawLine(const Offset(goalRight, 0), Offset(goalRight, tick), post);
    canvas.drawLine(Offset(goalLeft, h), Offset(goalLeft, h - tick), post);
    canvas.drawLine(Offset(goalRight, h), Offset(goalRight, h - tick), post);
  }
}
