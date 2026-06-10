import 'package:dio/dio.dart';
import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../api/air_hockey_api.dart';
import '../models/game_models.dart';

abstract class MatchmakingState extends Equatable {
  const MatchmakingState();

  @override
  List<Object?> get props => [];
}

class MatchmakingIdle extends MatchmakingState {
  const MatchmakingIdle();
}

class MatchmakingSearching extends MatchmakingState {
  const MatchmakingSearching();
}

class MatchmakingTimeout extends MatchmakingState {
  const MatchmakingTimeout();
}

class MatchmakingMatched extends MatchmakingState {
  const MatchmakingMatched(this.payload);

  final MatchReadyPayload payload;

  @override
  List<Object?> get props => [payload];
}

class MatchmakingFailed extends MatchmakingState {
  const MatchmakingFailed(this.message);

  final String message;

  @override
  List<Object?> get props => [message];
}

class MatchmakingCubit extends Cubit<MatchmakingState> {
  MatchmakingCubit(this._api) : super(const MatchmakingIdle());

  final AirHockeyApi _api;
  CancelToken? _cancelToken;

  void reset() {
    _cancelToken?.cancel('Reset matchmaking');
    _cancelToken = null;
    emit(const MatchmakingIdle());
  }

  void cancel() {
    _cancelToken?.cancel('User cancelled matchmaking');
    _cancelToken = null;
    emit(const MatchmakingIdle());
  }

  Future<void> leave(String playerId) async {
    try {
      await _api.leave(playerId);
    } catch (e) {
      // Silently fail - user is leaving anyway
    }
  }

  /// POST /join then long-poll GET /matchStatus until matched or recoverable waiting.
  Future<void> startMatch(String playerId) async {
    if (state is MatchmakingSearching) return;
    _cancelToken = CancelToken();
    emit(const MatchmakingSearching());

    try {
      await _api.join(playerId, _cancelToken);
      final res = await _api.matchStatusLongPoll(playerId, _cancelToken);

      final data = res.data;
      if (data is Map<String, dynamic>) {
        if (data['status'] == 'timeout') {
          emit(const MatchmakingTimeout());
          return;
        }
        if (data['GameID'] != null || data['gameID'] != null) {
          final payload = MatchReadyPayload.fromJson(data);
          emit(MatchmakingMatched(payload));
          return;
        }
        if (data['error'] != null) {
          emit(MatchmakingFailed(data['error'].toString()));
          return;
        }
      }
      emit(const MatchmakingFailed('Unexpected matchStatus response'));
    } on DioException catch (e) {
      if (e.type == DioExceptionType.cancel) {
        emit(const MatchmakingIdle());
        return;
      }
      final code = e.response?.statusCode;
      if (code == 408) {
        emit(const MatchmakingTimeout());
        await Future<void>.delayed(const Duration(milliseconds: 400));
        emit(const MatchmakingSearching());
        await startMatch(playerId);
        return;
      }
      emit(MatchmakingFailed(e.message ?? e.toString()));
    } catch (e) {
      emit(MatchmakingFailed(e.toString()));
    }
  }
}
