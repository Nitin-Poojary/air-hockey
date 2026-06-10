import 'package:dio/dio.dart';
import 'package:equatable/equatable.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../api/air_hockey_api.dart';
import '../storage/player_storage.dart';

class SessionState extends Equatable {
  const SessionState({
    this.playerId,
    this.displayName,
    this.loading = false,
    this.error,
    this.needsName = false,
  });

  final String? playerId;
  final String? displayName;
  final bool loading;
  final String? error;
  final bool needsName;

  SessionState copyWith({
    String? playerId,
    String? displayName,
    bool? loading,
    String? error,
    bool? needsName,
    bool clearError = false,
  }) {
    return SessionState(
      playerId: playerId ?? this.playerId,
      displayName: displayName ?? this.displayName,
      loading: loading ?? this.loading,
      error: clearError ? null : (error ?? this.error),
      needsName: needsName ?? this.needsName,
    );
  }

  @override
  List<Object?> get props => [playerId, displayName, loading, error, needsName];
}

class SessionCubit extends Cubit<SessionState> {
  SessionCubit(this._api, this._storage) : super(const SessionState(loading: true));

  final AirHockeyApi _api;
  final PlayerStorage _storage;

  /// Cold start: load cache, validate if present, otherwise ask for name.
  Future<void> initSession() async {
    emit(state.copyWith(loading: true, clearError: true));
    try {
      final cached = await _storage.read();
      if (cached == null) {
        emit(const SessionState(needsName: true, loading: false));
        return;
      }

      try {
        final identity = await _api.validatePlayer(cached.playerId);
        await _storage.save(identity.playerId, identity.displayName);
        emit(
          SessionState(
            playerId: identity.playerId,
            displayName: identity.displayName,
            loading: false,
          ),
        );
      } on PlayerNotFoundException {
        await _storage.clear();
        emit(const SessionState(needsName: true, loading: false));
      } on DioException {
        // Server unreachable: play with cached identity until validate succeeds.
        emit(
          SessionState(
            playerId: cached.playerId,
            displayName: cached.displayName,
            loading: false,
          ),
        );
      }
    } catch (e) {
      emit(
        SessionState(
          playerId: state.playerId,
          displayName: state.displayName,
          loading: false,
          error: e.toString(),
          needsName: state.playerId == null,
        ),
      );
    }
  }

  Future<void> resolveWithName(String name) async {
    final trimmed = name.trim();
    if (trimmed.length < 2 || trimmed.length > 20) {
      emit(
        state.copyWith(
          error: 'Name must be between 2 and 20 characters',
          clearError: false,
        ),
      );
      return;
    }

    emit(state.copyWith(loading: true, clearError: true));
    try {
      final identity = await _api.resolvePlayer(trimmed);
      await _storage.save(identity.playerId, identity.displayName);
      emit(
        SessionState(
          playerId: identity.playerId,
          displayName: identity.displayName,
          loading: false,
        ),
      );
    } on DioException catch (e) {
      emit(
        SessionState(
          needsName: true,
          loading: false,
          error: e.message ?? e.toString(),
        ),
      );
    } catch (e) {
      emit(
        SessionState(
          needsName: true,
          loading: false,
          error: e.toString(),
        ),
      );
    }
  }

  /// Optional explicit quit — not called on app pause/background.
  Future<void> unregister() async {
    if (state.playerId == null) return;
    try {
      await _api.unregister(state.playerId!);
    } catch (_) {
      // App is closing; best-effort only.
    }
  }
}
