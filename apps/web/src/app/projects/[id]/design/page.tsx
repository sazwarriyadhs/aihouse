import Link from "next/link";
import DashboardShell from "@/components/dashboard-shell";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  ArrowRight,
  Box,
  CheckCircle2,
  Compass,
  Layers3,
  Pencil,
  Ruler,
  Sparkles,
} from "lucide-react";
import { getProject } from "@/lib/api/projects";
import FloorPlanEditor from "@/components/floor-planning/floor-plan-editor";

export const dynamic = "force-dynamic";

export default async function DesignWorkspacePage({
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
      <DashboardShell activeItem="My Projects" breadcrumb="Design Workspace">
        <main className="project-create-page">
          <div className="project-create-container">
            <h1>Unable to load design workspace</h1>
            <p>{result.error || "An unexpected error occurred."}</p>
            <Link href="/">Back to My Projects</Link>
          </div>
        </main>
      </DashboardShell>
    );
  }

  const { project, designs } = result.data;
  const latestDesign = designs[0];
  const model = latestDesign?.canonical_model;
  const site = model?.site as Record<string, unknown> | undefined;
  const land = site?.land as Record<string, unknown> | undefined;
  const building = model?.building as Record<string, unknown> | undefined;

  const width = typeof land?.width_m === "number" ? land.width_m : null;
  const depth = typeof land?.depth_m === "number" ? land.depth_m : null;
  const area = typeof land?.area_m2 === "number" ? land.area_m2 : null;
  const floors = typeof building?.floors === "number" ? building.floors : null;

  return (
    <DashboardShell activeItem="My Projects" breadcrumb="Design Workspace">
      <main className="project-create-page">
        <div className="project-create-container">
          <Link href={`/projects/${project.id}`} className="project-back-link">
            <ArrowLeft size={17} />
            Back to Project
          </Link>

          <div className="project-create-heading">
            <span className="project-create-icon">
              <Sparkles size={23} />
            </span>
            <p className="eyebrow">DESIGN WORKSPACE</p>
            <h1>Continue building your vision.</h1>
            <p>
              Your project information and current design version are ready
              for the next stage of development.
            </p>
          </div>

          <section className="project-form-section">
            <div className="project-form-section-heading">
              <span>01</span>
              <div>
                <h2>{project.name}</h2>
                <p>{project.description || "No project description provided."}</p>
              </div>
            </div>

            <div className="project-detail-metadata-list">
              <div>
                <span>Location</span>
                <strong>{String(site?.location || "Not specified")}</strong>
              </div>
              <div>
                <span>Land dimensions</span>
                <strong>
                  {width !== null && depth !== null
                    ? `${width} m × ${depth} m`
                    : "Not available"}
                </strong>
              </div>
              <div>
                <span>Land area</span>
                <strong>{area !== null ? `${area} m²` : "Not available"}</strong>
              </div>
              <div>
                <span>Planned floors</span>
                <strong>{floors !== null ? floors : "Not specified"}</strong>
              </div>
              <div>
                <span>Current design version</span>
                <strong>
                  {latestDesign?.version_number != null
                    ? `v${latestDesign.version_number}`
                    : "No design version"}
                </strong>
              </div>
            </div>

            <div className="project-form-actions">
              <Link
                href={`/projects/${project.id}/edit`}
                className="primary-button"
              >
                <Pencil size={17} />
                Edit Project
                <ArrowRight size={17} />
              </Link>
            </div>
          </section>

          <FloorPlanEditor
            projectId={project.id}
            landWidth={width}
            landDepth={depth}
            floors={floors}
          />

          <div className="project-create-heading">
            <p className="eyebrow">DESIGN DEVELOPMENT PIPELINE</p>
            <h2>Next stages of your design</h2>
            <p>
              The following capabilities are planned for integration.
              They are not represented as operational tools yet.
            </p>
          </div>

          <div className="project-detail-overview-grid">
            <section className="project-detail-metadata-card">
              <div className="project-detail-metadata-heading">
                <span className="project-detail-metadata-icon">
                  <Ruler size={19} />
                </span>
                <div>
                  <p className="project-wow-eyebrow">PLANNED MODULE</p>
                  <h3>Floor Planning</h3>
                </div>
              </div>
              <p>Room layout, circulation, and floor plan development.</p>
              <span className="project-detail-card-tag">Coming soon</span>
            </section>

            <section className="project-detail-metadata-card">
              <div className="project-detail-metadata-heading">
                <span className="project-detail-metadata-icon">
                  <Box size={19} />
                </span>
                <div>
                  <p className="project-wow-eyebrow">PLANNED MODULE</p>
                  <h3>3D Visualization</h3>
                </div>
              </div>
              <p>Three-dimensional building visualization and exploration.</p>
              <span className="project-detail-card-tag">Coming soon</span>
            </section>

            <section className="project-detail-metadata-card">
              <div className="project-detail-metadata-heading">
                <span className="project-detail-metadata-icon">
                  <Compass size={19} />
                </span>
                <div>
                  <p className="project-wow-eyebrow">PLANNED MODULE</p>
                  <h3>CAD Development</h3>
                </div>
              </div>
              <p>Technical drawing and CAD document workflows.</p>
              <span className="project-detail-card-tag">Coming soon</span>
            </section>

            <section className="project-detail-metadata-card">
              <div className="project-detail-metadata-heading">
                <span className="project-detail-metadata-icon">
                  <Layers3 size={19} />
                </span>
                <div>
                  <p className="project-wow-eyebrow">PLANNED MODULE</p>
                  <h3>BOQ / RAB</h3>
                </div>
              </div>
              <p>Quantity takeoff and project cost estimation.</p>
              <span className="project-detail-card-tag">Coming soon</span>
            </section>
          </div>

          <section className="project-detail-next-step">
            <div className="project-detail-next-icon">
              <CheckCircle2 size={20} />
            </div>
            <div>
              <span className="project-wow-eyebrow">PROJECT CONTINUITY</span>
              <h3>Your design history is preserved.</h3>
              <p>
                Editing project information creates a new immutable design
                version. Previous versions remain part of the project history.
              </p>
              <Link href={`/projects/${project.id}`}>
                View project details
              </Link>
            </div>
          </section>
        </div>
      </main>
    </DashboardShell>
  );
}
