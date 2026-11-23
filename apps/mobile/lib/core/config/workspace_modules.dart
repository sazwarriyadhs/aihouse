import 'package:flutter/material.dart';

class WorkspaceModule {
  const WorkspaceModule({
    required this.title,
    required this.description,
    required this.icon,
  });

  final String title;
  final String description;
  final IconData icon;
}

const workspaceModules = <WorkspaceModule>[
  WorkspaceModule(
    title: 'Floor Planner',
    description: 'Plan spaces, arrange rooms, and define dimensions.',
    icon: Icons.layers_outlined,
  ),
  WorkspaceModule(
    title: '3D Visualization',
    description: 'Explore a conceptual view of your future home.',
    icon: Icons.view_in_ar_outlined,
  ),
  WorkspaceModule(
    title: 'CAD & Drawings',
    description: 'Organize technical drawing outputs and CAD exports.',
    icon: Icons.architecture_outlined,
  ),
  WorkspaceModule(
    title: 'BOQ & RAB',
    description: 'Review quantities and estimate construction costs.',
    icon: Icons.calculate_outlined,
  ),
  WorkspaceModule(
    title: 'Documents',
    description: 'Keep project documents and revisions organized.',
    icon: Icons.description_outlined,
  ),
  WorkspaceModule(
    title: 'Materials Library',
    description: 'Explore material specifications and reference prices.',
    icon: Icons.inventory_2_outlined,
  ),
];
