import 'package:flutter/material.dart';

import '../config/workspace_modules.dart';
import '../theme/app_theme.dart';

class ModuleScreen extends StatelessWidget {
  const ModuleScreen({super.key, required this.module});

  final WorkspaceModule module;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(title: Text(module.title)),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Container(
            padding: const EdgeInsets.all(22),
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(18),
              border: Border.all(color: AppColors.border),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  width: 54,
                  height: 54,
                  decoration: BoxDecoration(
                    color: AppColors.selected,
                    borderRadius: BorderRadius.circular(15),
                  ),
                  child: Icon(module.icon, color: AppColors.primary, size: 28),
                ),
                const SizedBox(height: 20),
                Text(
                  module.title,
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 8),
                Text(
                  module.description,
                  style: const TextStyle(color: AppColors.muted, height: 1.6),
                ),
                const SizedBox(height: 22),
                const Divider(),
                const SizedBox(height: 12),
                const Text(
                  'Workspace module foundation',
                  style: TextStyle(
                    color: AppColors.text,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                const Text(
                  'This screen is ready for the feature implementation and API integration phase.',
                  style: TextStyle(color: AppColors.muted, height: 1.6),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
