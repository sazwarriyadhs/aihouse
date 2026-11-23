import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../core/config/api_config.dart';
import '../models/project.dart';

class ProjectsApiException implements Exception {
  const ProjectsApiException(this.message);

  final String message;

  @override
  String toString() => message;
}

class ProjectsApi {
  ProjectsApi({http.Client? client}) : _client = client ?? http.Client();

  final http.Client _client;

  Uri _uri(String path) => Uri.parse('${ApiConfig.baseUrl}$path');

  Future<Map<String, dynamic>> _request(
    String method,
    String path, {
    Map<String, dynamic>? body,
  }) async {
    try {
      final uri = _uri(path);
      late http.Response response;

      switch (method) {
        case 'POST':
          response = await _client
              .post(
                uri,
                headers: const {'Content-Type': 'application/json'},
                body: jsonEncode(body),
              )
              .timeout(ApiConfig.requestTimeout);
        default:
          response = await _client.get(uri).timeout(ApiConfig.requestTimeout);
      }

      final dynamic decoded = response.body.isEmpty
          ? <String, dynamic>{}
          : jsonDecode(response.body);

      if (response.statusCode < 200 || response.statusCode >= 300) {
        final message = decoded is Map<String, dynamic>
            ? decoded['error']?.toString()
            : null;

        throw ProjectsApiException(
          message ?? 'Server returned HTTP ${response.statusCode}',
        );
      }

      if (decoded is! Map<String, dynamic>) {
        throw const ProjectsApiException('Invalid server response.');
      }

      return decoded;
    } on ProjectsApiException {
      rethrow;
    } on http.ClientException {
      throw const ProjectsApiException(
        'Cannot connect to the API. Check the server and network configuration.',
      );
    } on FormatException {
      throw const ProjectsApiException('Server returned invalid JSON.');
    } on Exception catch (error) {
      throw ProjectsApiException('Request failed: $error');
    }
  }

  Future<List<Project>> listProjects() async {
    final data = await _request('GET', '/api/v1/projects');
    final items = data['items'];

    if (items is! List) {
      throw const ProjectsApiException('Invalid project list response.');
    }

    return items
        .whereType<Map<String, dynamic>>()
        .map(Project.fromJson)
        .toList();
  }

  Future<String> createProject({
    required String name,
    required String description,
    required String location,
    required double landWidth,
    required double landDepth,
    required String orientation,
    required int floors,
  }) async {
    final data = await _request(
      'POST',
      '/api/v1/projects',
      body: {
        'name': name,
        'description': description,
        'location': location,
        'land_width_m': landWidth,
        'land_depth_m': landDepth,
        'orientation': orientation,
        'floors': floors,
      },
    );

    final id = data['project_id']?.toString();

    if (id == null || id.isEmpty) {
      throw const ProjectsApiException(
        'Project created but server did not return its ID.',
      );
    }

    return id;
  }

  Future<Map<String, dynamic>> getProject(String id) {
    return _request('GET', '/api/v1/projects/${Uri.encodeComponent(id)}');
  }

  void dispose() {
    _client.close();
  }
}
