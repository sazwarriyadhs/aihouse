import 'package:flutter/material.dart';

import '../core/config/workspace_modules.dart';
import '../core/theme/app_theme.dart';
import '../core/widgets/module_screen.dart';
import '../features/dashboard/dashboard_screen.dart';
import '../features/projects/screens/projects_screen.dart';
import '../features/projects/services/projects_api.dart';

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  final ProjectsApi _projectsApi = ProjectsApi();
  int _selectedIndex = 0;

  void _openModule(WorkspaceModule module) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(builder: (_) => ModuleScreen(module: module)),
    );
  }

  void _selectTab(int index) {
    setState(() => _selectedIndex = index);
  }

  @override
  Widget build(BuildContext context) {
    final pages = <Widget>[
      DashboardScreen(
        onOpenModule: _openModule,
        onOpenProjects: () => _selectTab(1),
      ),
      ProjectsScreen(api: _projectsApi),
      const _SimpleTab(
        title: 'AI Design Studio',
        description: 'AI-assisted home design workspace.',
        icon: Icons.auto_awesome_outlined,
      ),
      const _SimpleTab(
        title: 'Profile',
        description: 'Manage your personal workspace settings.',
        icon: Icons.person_outline,
      ),
    ];

    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        toolbarHeight: 64,
        titleSpacing: 4,
        title: Image.asset(
          'assets/branding/dasboard_logo.png',
          width: 158,
          height: 44,
          fit: BoxFit.contain,
          alignment: Alignment.centerLeft,
        ),
        actions: [
          IconButton(
            tooltip: 'Notifications',
            onPressed: () {},
            icon: const Icon(Icons.notifications_none_rounded),
          ),
          const SizedBox(width: 8),
        ],
      ),
      drawer: _WorkspaceDrawer(
        selectedIndex: _selectedIndex,
        onSelectTab: _selectTab,
        onOpenModule: _openModule,
      ),
      body: IndexedStack(index: _selectedIndex, children: pages),
      bottomNavigationBar: NavigationBar(
        height: 68,
        backgroundColor: AppColors.surface,
        surfaceTintColor: Colors.transparent,
        indicatorColor: AppColors.selected,
        selectedIndex: _selectedIndex,
        onDestinationSelected: _selectTab,
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.home_outlined),
            selectedIcon: Icon(Icons.home, color: AppColors.primary),
            label: 'Home',
          ),
          NavigationDestination(
            icon: Icon(Icons.folder_outlined),
            selectedIcon: Icon(Icons.folder, color: AppColors.primary),
            label: 'Projects',
          ),
          NavigationDestination(
            icon: Icon(Icons.auto_awesome_outlined),
            selectedIcon: Icon(Icons.auto_awesome, color: AppColors.primary),
            label: 'AI Studio',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person, color: AppColors.primary),
            label: 'Profile',
          ),
        ],
      ),
    );
  }
}

class _WorkspaceDrawer extends StatelessWidget {
  const _WorkspaceDrawer({
    required this.selectedIndex,
    required this.onSelectTab,
    required this.onOpenModule,
  });

  final int selectedIndex;
  final ValueChanged<int> onSelectTab;
  final ValueChanged<WorkspaceModule> onOpenModule;

  @override
  Widget build(BuildContext context) {
    return Drawer(
      backgroundColor: AppColors.surface,
      child: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 20),
              child: Image.asset(
                'assets/branding/dasboard_logo.png',
                width: 178,
                height: 55,
                fit: BoxFit.contain,
                alignment: Alignment.centerLeft,
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.surface,
                  border: Border.all(color: AppColors.border),
                  borderRadius: BorderRadius.circular(11),
                ),
                child: const Row(
                  children: [
                    CircleAvatar(
                      radius: 17,
                      backgroundColor: AppColors.selected,
                      child: Text(
                        'A',
                        style: TextStyle(
                          color: AppColors.primary,
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                    ),
                    SizedBox(width: 11),
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'My Workspace',
                          style: TextStyle(
                            color: AppColors.text,
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        Text(
                          'Personal account',
                          style: TextStyle(
                            color: AppColors.muted,
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
            const Padding(
              padding: EdgeInsets.fromLTRB(27, 25, 16, 8),
              child: Text(
                'WORKSPACE',
                style: TextStyle(
                  color: Color(0xFFA0A3AB),
                  fontSize: 10,
                  fontWeight: FontWeight.w800,
                  letterSpacing: 1.2,
                ),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.symmetric(horizontal: 12),
                children: [
                  _DrawerItem(
                    icon: Icons.home_outlined,
                    label: 'Dashboard',
                    selected: selectedIndex == 0,
                    onTap: () {
                      Navigator.pop(context);
                      onSelectTab(0);
                    },
                  ),
                  _DrawerItem(
                    icon: Icons.folder_outlined,
                    label: 'My Projects',
                    selected: selectedIndex == 1,
                    onTap: () {
                      Navigator.pop(context);
                      onSelectTab(1);
                    },
                  ),
                  _DrawerItem(
                    icon: Icons.auto_awesome_outlined,
                    label: 'AI Design Studio',
                    selected: selectedIndex == 2,
                    onTap: () {
                      Navigator.pop(context);
                      onSelectTab(2);
                    },
                  ),
                  const Divider(height: 20),
                  for (final module in workspaceModules)
                    _DrawerItem(
                      icon: module.icon,
                      label: module.title,
                      onTap: () {
                        Navigator.pop(context);
                        onOpenModule(module);
                      },
                    ),
                ],
              ),
            ),
            const Divider(height: 1),
            const Padding(
              padding: EdgeInsets.all(18),
              child: Row(
                children: [
                  Icon(Icons.check_circle, size: 14, color: AppColors.success),
                  SizedBox(width: 8),
                  Text(
                    'All systems operational',
                    style: TextStyle(color: AppColors.muted, fontSize: 10),
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

class _DrawerItem extends StatelessWidget {
  const _DrawerItem({
    required this.icon,
    required this.label,
    required this.onTap,
    this.selected = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      dense: true,
      visualDensity: const VisualDensity(vertical: -1),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      tileColor: selected ? AppColors.selected : Colors.transparent,
      leading: Icon(
        icon,
        size: 19,
        color: selected ? AppColors.primary : AppColors.muted,
      ),
      title: Text(
        label,
        style: TextStyle(
          color: selected ? AppColors.primary : AppColors.text,
          fontSize: 12,
          fontWeight: selected ? FontWeight.w700 : FontWeight.w600,
        ),
      ),
      onTap: onTap,
    );
  }
}

class _SimpleTab extends StatelessWidget {
  const _SimpleTab({
    required this.title,
    required this.description,
    required this.icon,
  });

  final String title;
  final String description;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, color: AppColors.primary, size: 42),
            const SizedBox(height: 15),
            Text(title, style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 8),
            Text(
              description,
              textAlign: TextAlign.center,
              style: const TextStyle(color: AppColors.muted),
            ),
          ],
        ),
      ),
    );
  }
}
