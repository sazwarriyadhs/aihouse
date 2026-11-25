import DashboardShell from "@/components/dashboard-shell";
import { notFound } from "next/navigation";
import { getProject } from "@/lib/api/projects";
import EditProjectForm from "./edit-project-form";

export const dynamic = "force-dynamic";

function asRecord(value: unknown): Record<string, unknown> | undefined {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return undefined;
}

function numberOr(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : fallback;
}

export default async function EditProjectPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const result = await getProject(id);

  if (result.error === "Project not found") {
    notFound();
  }

  if (!result.data) {
    return (
      <DashboardShell activeItem="My Projects" breadcrumb="Edit Project">
        <main className="project-create-page">
          <div className="project-create-container">
            <h1>Unable to load project</h1>
            <p>{result.error || "An unexpected error occurred."}</p>
          </div>
        </main>
      </DashboardShell>
    );
  }

  const { project, designs } = result.data;
  const model = asRecord(designs[0]?.canonical_model);
  const site = asRecord(model?.site);
  const land = asRecord(site?.land);
  const building = asRecord(model?.building);
  const style = asRecord(building?.style);

  const initialValues = {
    name: project.name || "",
    description: project.description || "",
    location: typeof site?.location === "string" ? site.location : "",
    land_width_m: numberOr(land?.width_m, 10),
    land_depth_m: numberOr(land?.depth_m, 15),
    orientation:
      typeof land?.orientation === "string" ? land.orientation : "N",
    floors: numberOr(building?.floors, 1),
  };

  const initialStyleId =
    typeof style?.id === "string" ? style.id : "";

  return (
    <EditProjectForm
      project={project}
      initialValues={initialValues}
      initialStyleId={initialStyleId}
    />
  );
}
