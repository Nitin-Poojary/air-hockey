import 'package:dio/dio.dart';

import '../config/app_config.dart';
import '../models/player_identity.dart';

class PlayerNotFoundException implements Exception {
  const PlayerNotFoundException();
}

class AirHockeyApi {
  AirHockeyApi(this.config, {Dio? dio})
    : _dio =
          dio ??
          Dio(
            BaseOptions(
              connectTimeout: const Duration(seconds: 20),
              receiveTimeout: const Duration(seconds: 30),
              sendTimeout: const Duration(seconds: 10),
            ),
          );

  final AppConfig config;
  final Dio _dio;

  Future<PlayerIdentity> resolvePlayer(String name) async {
    final res = await _dio.post<Map<String, dynamic>>(
      config.resolvePlayerUrl,
      data: {'name': name},
      options: Options(
        contentType: Headers.jsonContentType,
        validateStatus: (s) => s != null && s < 500,
      ),
    );
    if (res.statusCode != 200 || res.data == null) {
      final err = res.data?['error'] as String?;
      throw DioException(
        requestOptions: res.requestOptions,
        response: res,
        message: err ?? 'resolve failed (${res.statusCode})',
      );
    }
    return PlayerIdentity.fromJson(res.data!);
  }

  Future<PlayerIdentity> validatePlayer(String playerId) async {
    final res = await _dio.get<Map<String, dynamic>>(
      config.playerUrl(playerId),
      options: Options(validateStatus: (s) => s != null && s < 500),
    );
    if (res.statusCode == 404) {
      throw const PlayerNotFoundException();
    }
    if (res.statusCode != 200 || res.data == null) {
      final err = res.data?['error'] as String?;
      throw DioException(
        requestOptions: res.requestOptions,
        response: res,
        message: err ?? 'validate failed (${res.statusCode})',
      );
    }
    return PlayerIdentity.fromJson(res.data!);
  }

  Future<void> join(String playerId, [CancelToken? cancelToken]) async {
    await _dio.post<void>(
      config.joinUrl,
      data: {'playerID': playerId},
      cancelToken: cancelToken,
      options: Options(
        contentType: Headers.jsonContentType,
        validateStatus: (s) => s != null && s < 500,
      ),
    );
  }

  /// Long-poll: server blocks until a match is ready or ~90s elapses.
  Future<Response<dynamic>> matchStatusLongPoll(
    String playerId, [
    CancelToken? cancelToken,
  ]) {
    return _dio.get<dynamic>(
      config.matchStatusUrl(playerId),
      cancelToken: cancelToken,
      options: Options(
        receiveTimeout: const Duration(seconds: 95),
        sendTimeout: const Duration(seconds: 30),
      ),
    );
  }

  Future<void> leave(String playerId) async {
    await _dio.post<void>(
      '${config.httpBaseUrl}/leave',
      data: {'playerID': playerId},
      options: Options(
        contentType: Headers.jsonContentType,
        validateStatus: (s) => s != null && s < 500,
      ),
    );
  }

  Future<void> unregister(String playerId) async {
    await _dio.post<void>(
      '${config.httpBaseUrl}/unregister',
      data: {'playerID': playerId},
      options: Options(
        contentType: Headers.jsonContentType,
        validateStatus: (s) => s != null && s < 500,
      ),
    );
  }
}
