import 'dart:async';
import 'dart:convert';
import 'dart:developer';
import 'dart:io';

import '../config/app_config.dart';
import '../models/game_models.dart';
import '../network/state_interpolator.dart';

typedef UdpGameStateCallback = void Function(GameStatePayload state);
typedef UdpMatchEndCallback =
    void Function(String winnerPlayerId, String winnerName);

class UdpGameSession {
  UdpGameSession({
    required this.config,
    required this.gameId,
    required this.playerId,
    required this.onState,
    this.onAuthoritative,
    required this.onMatchEnd,
    required this.interpolator,
  });

  final AppConfig config;
  final String gameId;
  final String playerId;
  final UdpGameStateCallback onState;
  final void Function(GameStatePayload raw)? onAuthoritative;
  final UdpMatchEndCallback onMatchEnd;
  final StateInterpolator interpolator;

  RawDatagramSocket? _socket;
  StreamSubscription<RawSocketEvent>? _sub;

  double _nowSec() => DateTime.now().microsecondsSinceEpoch / 1e6;

  Future<void> start(double initialX, double initialY) async {
    try {
      final socket = await RawDatagramSocket.bind(InternetAddress.anyIPv4, 0);
      _socket = socket;
      log(
        "UdpGameSession started. Bound to socket on local port ${socket.port}",
      );

      _sub = socket.listen(
        (event) {
          if (event == RawSocketEvent.read) {
            final dg = socket.receive();
            if (dg == null) return;
            final raw = utf8.decode(dg.data);
            final parsed = parseUdpJson(raw);
            if (parsed is GameStatePayload) {
              onAuthoritative?.call(parsed);
              final now = _nowSec();
              interpolator.push(parsed, now);
              final smooth = interpolator.interpolate(now) ?? parsed;
              onState(smooth);
            } else if (parsed is MatchResultUdpPayload) {
              onMatchEnd(parsed.winnerId, parsed.winnerName);
            }
          }
        },
        onError: (err, st) {
          log("UdpGameSession Socket Error: $err\n$st");
        },
        onDone: () {
          log("UdpGameSession Socket closed");
        },
      );

      sendPaddle(initialX, initialY);
    } catch (e, stack) {
      log("UdpGameSession start failed: $e\n$stack");
      rethrow;
    }
  }

  void sendPaddle(double x, double y) {
    final socket = _socket;
    if (socket == null) return;
    try {
      final jsonMap = {'gameID': gameId, 'playerID': playerId, 'x': x, 'y': y};
      final payloadString = jsonEncode(jsonMap);
      final payload = utf8.encode(payloadString);
      final destIP = InternetAddress(config.udpHost);
      final bytesSent = socket.send(payload, destIP, config.udpPort);
      if (bytesSent <= 0) {
        log(
          "UdpGameSession Warning: sent $bytesSent bytes to ${destIP.address}:${config.udpPort}",
        );
      }
    } catch (e) {
      log("UdpGameSession sendPaddle failed: $e");
    }
  }

  Future<void> dispose() async {
    await _sub?.cancel();
    _sub = null;
    _socket?.close();
    _socket = null;
    interpolator.clear();
  }
}
