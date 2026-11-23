import { notFound } from "next/navigation";
import WorkspaceModule from "../../features/workspace/workspace-module";

const allowedModules = new Set([
  "floor-planner",
  "visualization",
  "cad",
  "boq-rab",
  "documents",
  "materials",
]);

export default async function ModulePage({
  params,
}: {
  params: Promise<{ module: string }>;
}) {
  const { module: moduleSlug } = await params;

  if (!allowedModules.has(moduleSlug)) {
    notFound();
  }

  return <WorkspaceModule moduleKey={moduleSlug} />;
}