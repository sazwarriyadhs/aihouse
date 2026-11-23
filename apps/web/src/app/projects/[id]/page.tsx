import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, House, MapPin, Ruler, Layers3 } from "lucide-react";
import { getProject } from "@/lib/api/projects";

export const dynamic = "force-dynamic";

export default async function ProjectDetailPage({
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
      <main className="project-create-page">
        <div className="project-create-container">
          <Link href="/" className="project-back-link">
            <ArrowLeft size={17} /> Back to Dashboard
          </Link>
          <div className="project-detail-error">
            <h1>Unable to load project</h1>
            <p>{result.error}</p>
            <Link href="/" className="primary-button">Return to Dashboard</Link>
          </div>
        </div>
      </main>
    );
  }

  const { project, designs } = result.data;
  const latestDesign = designs[0];
  const model = latestDesign?.canonical_model;
  const site = model?.site as Record<string, unknown> | undefined;
  const land = site?.land as Record<string, unknown> | undefined;
  const building = model?.building as Record<string, unknown> | undefined;

  return (
    <main className="project-create-page">
      <div className="project-create-container project-detail-container">
        <Link href="/" className="project-back-link">
          <ArrowLeft size={17} /> Back to Dashboard
        </Link>

        <div className="project-detail-heading">
          <span className="project-create-icon"><House size={23} /></span>
          <div>
            <p className="eyebrow">PROJECT WORKSPACE</p>
            <h1>{project.name}</h1>
            <span className={`project-status status-${project.status.toLowerCase()}`}>
              {project.status}
            </span>
          </div>
        </div>

        {project.description && <p className="project-detail-description">{project.description}</p>}

        <section className="project-detail-grid">
          <article className="project-detail-card">
            <MapPin size={20} />
            <span>Location</span>
            <strong>{String(site?.location || "Not specified")}</strong>
          </article>
          <article className="project-detail-card">
            <Ruler size={20} />
            <span>Land area</span>
            <strong>
              {typeof land?.area_m2 === "number" ? `${land.area_m2} m²` : "Not available"}
            </strong>
          </article>
          <article className="project-detail-card">
            <Layers3 size={20} />
            <span>Building floors</span>
            <strong>{typeof building?.floors === "number" ? building.floors : "Not specified"}</strong>
          </article>
        </section>

        <section className="project-detail-panel">
          <p className="eyebrow">DESIGN INFORMATION</p>
          <h2>{latestDesign?.name || "Initial design"}</h2>
          <p>Design version: {latestDesign?.version_number ?? "—"}</p>
          <p>Design status: {latestDesign?.status || "—"}</p>
          <p className="project-checksum">
            Model checksum: {latestDesign?.model_checksum || "Not available"}
          </p>
        </section>

        <p className="project-detail-note">
          This is the initial project workspace. Floor planning, 3D visualization,
          CAD export and BOQ/RAB workflows will be connected in subsequent phases.
        </p>
      </div>
    </main>
  );
}