import 'package:flutter/material.dart';

import '../../core/config/workspace_modules.dart';
import '../../core/theme/app_theme.dart';

class DashboardScreen extends StatelessWidget {
  const DashboardScreen({
    super.key,
    required this.onOpenModule,
    required this.onOpenProjects,
  });

  final ValueChanged<WorkspaceModule> onOpenModule;
  final VoidCallback onOpenProjects;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth;
        final isWide = width >= 700;
        final columns = width >= 900 ? 3 : 2;
        final horizontalPadding = isWide ? 30.0 : 20.0;

        return ListView(
          padding: EdgeInsets.fromLTRB(
            horizontalPadding,
            18,
            horizontalPadding,
            32,
          ),
          children: [
            _HeroCover(width: width),
            const SizedBox(height: 25),

            const _SectionLabel(text: 'YOUR DESIGN WORKSPACE'),
            const SizedBox(height: 7),

            Text(
              'Build your next home.',
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                fontSize: isWide ? 28 : 24,
                letterSpacing: -0.6,
              ),
            ),
            const SizedBox(height: 7),

            const Text(
              'From land to living. Turn your ideas into a thoughtfully '
              'designed home.',
              style: TextStyle(
                color: AppColors.muted,
                fontSize: 13,
                height: 1.6,
              ),
            ),
            const SizedBox(height: 19),

            _ProjectActionCard(onTap: onOpenProjects),
            const SizedBox(height: 29),

            _SectionHeading(title: 'Design workspace', trailing: '6 MODULES'),
            const SizedBox(height: 13),

            GridView.builder(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: workspaceModules.length,
              gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: columns,
                crossAxisSpacing: 12,
                mainAxisSpacing: 12,
                childAspectRatio: isWide ? 1.3 : 1.12,
              ),
              itemBuilder: (context, index) {
                final module = workspaceModules[index];

                return _ModuleCard(
                  module: module,
                  onTap: () => onOpenModule(module),
                );
              },
            ),

            const SizedBox(height: 29),

            _SectionHeading(
              title: 'Recent projects',
              trailing: 'VIEW ALL',
              onTrailingTap: onOpenProjects,
            ),
            const SizedBox(height: 12),

            _EmptyProjectsCard(onTap: onOpenProjects),
          ],
        );
      },
    );
  }
}

class _HeroCover extends StatelessWidget {
  const _HeroCover({required this.width});

  final double width;

  @override
  Widget build(BuildContext context) {
    final height = width >= 700 ? 270.0 : 205.0;
    final pixelRatio = MediaQuery.devicePixelRatioOf(context);

    final cacheWidth = (width * pixelRatio).round().clamp(1, 1376).toInt();

    final cacheHeight = (height * pixelRatio).round().clamp(1, 768).toInt();

    return ClipRRect(
      borderRadius: BorderRadius.circular(20),
      child: SizedBox(
        width: double.infinity,
        height: height,
        child: Stack(
          fit: StackFit.expand,
          children: [
            Image.asset(
              'assets/branding/head_cover.jpg',
              fit: BoxFit.cover,
              alignment: Alignment.center,
              cacheWidth: cacheWidth,
              cacheHeight: cacheHeight,
              errorBuilder: (context, error, stackTrace) {
                return Container(
                  color: AppColors.selected,
                  child: const Center(
                    child: Icon(
                      Icons.home_work_outlined,
                      color: AppColors.primary,
                      size: 52,
                    ),
                  ),
                );
              },
            ),

            DecoratedBox(
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topCenter,
                  end: Alignment.bottomCenter,
                  colors: [
                    Colors.transparent,
                    AppColors.navy.withValues(alpha: 0.78),
                  ],
                  stops: const [0.25, 1],
                ),
              ),
            ),

            Positioned(
              left: 18,
              right: 18,
              top: 17,
              child: Align(
                alignment: Alignment.centerLeft,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 11,
                    vertical: 7,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.93),
                    borderRadius: BorderRadius.circular(30),
                  ),
                  child: const Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        Icons.auto_awesome,
                        color: AppColors.primary,
                        size: 13,
                      ),
                      SizedBox(width: 6),
                      Text(
                        'AI-POWERED HOME DESIGN',
                        style: TextStyle(
                          color: AppColors.text,
                          fontSize: 9,
                          fontWeight: FontWeight.w800,
                          letterSpacing: 0.7,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),

            Positioned(
              left: 20,
              right: 20,
              bottom: 19,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    'Imagine it. Design it.\nMake it home.',
                    style: TextStyle(
                      color: Colors.white,
                      fontSize: width >= 700 ? 27 : 22,
                      height: 1.13,
                      fontWeight: FontWeight.w800,
                      letterSpacing: -0.5,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(
      text,
      style: const TextStyle(
        color: AppColors.primary,
        fontSize: 10,
        fontWeight: FontWeight.w800,
        letterSpacing: 1.25,
      ),
    );
  }
}

class _SectionHeading extends StatelessWidget {
  const _SectionHeading({
    required this.title,
    this.trailing,
    this.onTrailingTap,
  });

  final String title;
  final String? trailing;
  final VoidCallback? onTrailingTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(
            title,
            style: Theme.of(
              context,
            ).textTheme.titleLarge?.copyWith(fontSize: 17, letterSpacing: -0.3),
          ),
        ),
        if (trailing != null)
          GestureDetector(
            onTap: onTrailingTap,
            child: Text(
              trailing!,
              style: TextStyle(
                color: onTrailingTap == null
                    ? AppColors.muted
                    : AppColors.primary,
                fontSize: 9,
                fontWeight: FontWeight.w800,
                letterSpacing: 0.8,
              ),
            ),
          ),
      ],
    );
  }
}

class _ProjectActionCard extends StatelessWidget {
  const _ProjectActionCard({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(17),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(17),
        child: Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [Color(0xFFFFFFFF), Color(0xFFF0F6FF)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            border: Border.all(color: const Color(0xFFDCE9FF)),
            borderRadius: BorderRadius.circular(17),
          ),
          child: Row(
            children: [
              Container(
                width: 48,
                height: 48,
                decoration: BoxDecoration(
                  color: AppColors.primary,
                  borderRadius: BorderRadius.circular(14),
                  boxShadow: [
                    BoxShadow(
                      color: AppColors.primary.withValues(alpha: 0.2),
                      blurRadius: 12,
                      offset: const Offset(0, 5),
                    ),
                  ],
                ),
                child: const Icon(
                  Icons.add_home_work_outlined,
                  color: Colors.white,
                  size: 24,
                ),
              ),
              const SizedBox(width: 13),
              const Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Start a new project',
                      style: TextStyle(
                        color: AppColors.text,
                        fontWeight: FontWeight.w800,
                        fontSize: 13,
                      ),
                    ),
                    SizedBox(height: 5),
                    Text(
                      'Bring your dream home to life.',
                      style: TextStyle(color: AppColors.muted, fontSize: 11),
                    ),
                  ],
                ),
              ),
              const Icon(
                Icons.arrow_forward_rounded,
                color: AppColors.primary,
                size: 21,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ModuleCard extends StatelessWidget {
  const _ModuleCard({required this.module, required this.onTap});

  final WorkspaceModule module;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(16),
        child: Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: AppColors.surface,
            border: Border.all(color: AppColors.border),
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 41,
                height: 41,
                decoration: BoxDecoration(
                  color: AppColors.selected,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(module.icon, color: AppColors.primary, size: 21),
              ),
              const Spacer(),
              Text(
                module.title,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: AppColors.text,
                  fontWeight: FontWeight.w800,
                  fontSize: 12,
                  height: 1.3,
                ),
              ),
              const SizedBox(height: 7),
              const Row(
                children: [
                  Text(
                    'Open workspace',
                    style: TextStyle(
                      color: AppColors.primary,
                      fontSize: 10,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  SizedBox(width: 4),
                  Icon(
                    Icons.arrow_forward_rounded,
                    color: AppColors.primary,
                    size: 12,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EmptyProjectsCard extends StatelessWidget {
  const _EmptyProjectsCard({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
      decoration: BoxDecoration(
        color: AppColors.surface,
        border: Border.all(color: AppColors.border),
        borderRadius: BorderRadius.circular(17),
      ),
      child: Column(
        children: [
          Container(
            width: 49,
            height: 49,
            decoration: BoxDecoration(
              color: AppColors.selected,
              borderRadius: BorderRadius.circular(15),
            ),
            child: const Icon(
              Icons.home_work_outlined,
              color: AppColors.primary,
              size: 24,
            ),
          ),
          const SizedBox(height: 12),
          const Text(
            'Your next home starts here',
            style: TextStyle(
              color: AppColors.text,
              fontWeight: FontWeight.w800,
              fontSize: 13,
            ),
          ),
          const SizedBox(height: 6),
          const Text(
            'Create a project to start planning your land, '
            'floor plan, 3D design and construction budget.',
            textAlign: TextAlign.center,
            style: TextStyle(
              color: AppColors.muted,
              fontSize: 11,
              height: 1.55,
            ),
          ),
          const SizedBox(height: 13),
          TextButton.icon(
            onPressed: onTap,
            icon: const Icon(Icons.folder_open_outlined, size: 16),
            label: const Text('View my projects'),
          ),
        ],
      ),
    );
  }
}
