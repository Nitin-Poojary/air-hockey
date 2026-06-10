import 'package:flutter_dotenv/flutter_dotenv.dart';

/// HTTP and UDP endpoints for the air-hockey backend.
///
/// - Use `http://127.0.0.1:8080` for desktop testing against a local server.
/// - Android emulator: use `http://[IP_ADDRESS]` to reach the host machine.
/// - UDP must target the same host the Go server binds to ([backend] uses 127.0.0.1:8050).
class AppConfig {
  AppConfig({
    this.httpBaseUrl = 'http://[IP_ADDRESS]',
    String? udpHost,
    this.udpPort = 8050,
  }) : udpHost = udpHost ?? Uri.parse(httpBaseUrl).host;

  factory AppConfig.fromEnvironment() {
    final httpBaseUrl = dotenv.get(
      'HTTP_BASE_URL',
      fallback: 'http://127.0.0.1:8080',
    );
    final udpPort =
        int.tryParse(dotenv.get('UDP_PORT', fallback: '8050')) ?? 8050;
    final udpHost = dotenv.get(
      'UDP_HOST',
      fallback: Uri.parse(httpBaseUrl).host,
    );

    return AppConfig(
      httpBaseUrl: httpBaseUrl,
      udpHost: udpHost,
      udpPort: udpPort,
    );
  }

  final String httpBaseUrl;
  final String udpHost;
  final int udpPort;

  String get resolvePlayerUrl => '$httpBaseUrl/players/resolve';
  String playerUrl(String playerId) => '$httpBaseUrl/players/$playerId';
  String get joinUrl => '$httpBaseUrl/join';
  String matchStatusUrl(String playerId) =>
      '$httpBaseUrl/matchStatus?playerID=$playerId';
}
