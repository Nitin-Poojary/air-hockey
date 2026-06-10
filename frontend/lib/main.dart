import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

import 'api/air_hockey_api.dart';
import 'app.dart';
import 'config/app_config.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await dotenv.load(fileName: '.env');

  final config = AppConfig.fromEnvironment();
  final api = AirHockeyApi(config);
  runApp(AirHockeyApp(config: config, api: api));
}
