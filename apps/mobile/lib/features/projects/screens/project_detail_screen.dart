import 'dart:convert';

import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';
import '../services/projects_api.dart';

class ProjectDetailScreen extends StatefulWidget {
  const ProjectDetailScreen({
    super.key,
    required this.api,
    required this.projectId,
  });

  final ProjectsApi api;
  final String projectId;

  @override
  State<ProjectDetailScreen> createState() => _ProjectDetailScreenState();
}

class _ProjectDetailScreenState extends State<ProjectDetailScreen> {
  late Future<Map<String, dynamic>> _future;

  @override
  void initState() {
    super.initState();
    _future = widget.api.getProject(widget.projectId);
  }

  void _reload() {
    setState(() => _future = widget.api.getProject(widget.projectId));
  }

  Widget _infoCard(String label, String value) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppColors.surface,
        border: Border.all(color: AppColors.border),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(color: AppColors.muted, fontSize: 11),
          ),
          const SizedBox(height: 5),
          Text(
            value,
            style: const TextStyle(
              color: AppColors.text,
              fontWeight: FontWeight.w700,
              fontSize: 14,
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        title: const Text('Project details'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: FutureBuilder<Map<String, dynamic>>(
        future: _future,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }

          if (snapshot.hasError) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(
                      Icons.cloud_off_outlined,
                      size: 40,
                      color: AppColors.muted,
                    ),
                    const SizedBox(height: 12),
                    Text(
                      snapshot.error.toString(),
                      textAlign: TextAlign.center,
                    ),
                    TextButton(
                      onPressed: _reload,
                      child: const Text('Try again'),
                    ),
                  ],
                ),
              ),
            );
          }

          final data = snapshot.data!;
          final project = data['project'] as Map<String, dynamic>? ?? {};
          final designs = data['designs'] as List<dynamic>? ?? [];
          final firstDesign = designs.isNotEmpty && designs.first is Map
              ? Map<String, dynamic>.from(designs.first as Map)
              : <String, dynamic>{};
          final model = firstDesign['canonical_model'];
          final site = model is Map ? model['site'] : null;
          final land = site is Map ? site['land'] : null;
          final building = model is Map ? model['building'] : null;

          final width = land is Map ? land['width_m'] : null;
          final depth = land is Map ? land['depth_m'] : null;
          final area = land is Map ? land['area_m2'] : null;
          final orientation = land is Map ? land['orientation'] : null;
          final floors = building is Map ? building['floors'] : null;
          final location = site is Map ? site['location'] : null;

          return ListView(
            padding: const EdgeInsets.all(20),
            children: [
              Text(
                project['name']?.toString() ?? 'Project',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              Text(
                project['description']?.toString() ?? '',
                style: const TextStyle(color: AppColors.muted, height: 1.5),
              ),
              const SizedBox(height: 18),
              _infoCard('STATUS', project['status']?.toString() ?? 'draft'),
              const SizedBox(height: 12),
              _infoCard('LOCATION', location?.toString() ?? 'Not specified'),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: _infoCard(
                      'LAND AREA',
                      area == null ? '—' : '$area m²',
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: _infoCard('FLOORS', floors?.toString() ?? '—'),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: _infoCard('WIDTH', width == null ? '—' : '$width m'),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: _infoCard('DEPTH', depth == null ? '—' : '$depth m'),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              _infoCard('ORIENTATION', orientation?.toString() ?? '—'),
              const SizedBox(height: 24),
              Text(
                'Design versions',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 10),
              if (designs.isEmpty)
                const Text('No design version available.')
              else
                ...designs.map((item) {
                  final design = Map<String, dynamic>.from(item as Map);
                  return Card(
                    color: AppColors.surface,
                    child: ListTile(
                      leading: const Icon(
                        Icons.architecture,
                        color: AppColors.primary,
                      ),
                      title: Text(design['name']?.toString() ?? 'Design'),
                      subtitle: Text(
                        'Version ${design['version_number'] ?? 1} • ${design['status'] ?? 'draft'}',
                      ),
                    ),
                  );
                }),
              const SizedBox(height: 20),
              Text(
                'Canonical model',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 10),
              Container(
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: AppColors.surface,
                  border: Border.all(color: AppColors.border),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: SelectableText(
                  const JsonEncoder.withIndent('  ').convert(model ?? {}),
                  style: const TextStyle(
                    color: AppColors.text,
                    fontSize: 11,
                    fontFamily: 'monospace',
                    height: 1.5,
                  ),
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
