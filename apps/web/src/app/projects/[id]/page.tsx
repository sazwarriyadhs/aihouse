import Link from "next/link";
import Image from "next/image";
import DashboardShell from "@/components/dashboard-shell";
import { notFound } from "next/navigation";
import {
  ArrowLeft,
  ArrowRight,
  CalendarDays,
  Fingerprint,
  Pencil,
  House,
  Layers3,
  MapPin,
  Ruler,
  Sparkles,
  WandSparkles,
} from "lucide-react";
import { getProject } from "@/lib/api/projects";

export const dynamic = "force-dynamic";

function formatDate(value?: string) {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(date);
}

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
      <DashboardShell activeItem="My Projects" breadcrumb="My Projects">
        <div className="dashboard-content">
          <div className="project-detail-wow project-detail-error-wow">
            <Link href="/" className="project-detail-back">
              <ArrowLeft size={17} />
              Back to Dashboard
            </Link>

            <div className="project-detail-error">
              <span className="project-detail-error-icon">
                <House size={26} />
              </span>
              <p className="project-wow-eyebrow">PROJECT WORKSPACE</p>
              <h1>Unable to load project</h1>
              <p>{result.error || "An unexpected error occurred."}</p>
              <Link href="/" className="project-detail-primary-link">
                <ArrowLeft size={16} />
                Return to Dashboard
              </Link>
            </div>
          </div>
        </div>
      </DashboardShell>
    );
  }

  const { project, designs } = result.data;
  const latestDesign = designs[0];
  const model = latestDesign?.canonical_model;
  const site = model?.site as Record<string, unknown> | undefined;
  const land = site?.land as Record<string, unknown> | undefined;
  const building = model?.building as Record<string, unknown> | undefined;
  const styleValue = building?.style;
  const selectedStyle =
    styleValue && typeof styleValue === "object"
      ? (styleValue as Record<string, unknown>)
      : undefined;
  const selectedStyleName =
    typeof selectedStyle?.name === "string" ? selectedStyle.name : "";
  const selectedStyleImage =
    typeof selectedStyle?.image === "string" ? selectedStyle.image : "";

  const location = String(site?.location || "Location not specified");
  const landArea =
    typeof land?.area_m2 === "number"
      ? `${land.area_m2.toLocaleString("en-US")} m²`
      : "Not available";
  const floors =
    typeof building?.floors === "number"
      ? `${building.floors} ${building.floors === 1 ? "floor" : "floors"}`
      : "Not specified";
  const status = project.status || "draft";

  return (
    <DashboardShell activeItem="My Projects" breadcrumb="My Projects">
      <div className="dashboard-content">
        <div className="project-detail-wow">
          <Link href="/" className="project-detail-back">
            <ArrowLeft size={17} />
            Back to My Projects
          </Link>

          <section className="project-detail-hero">
            <div className="project-detail-hero-glow" />

            <div className="project-detail-hero-content">
              <div className="project-detail-kicker">
                <span className="project-detail-kicker-icon">
                  <House size={15} />
                </span>
                YOUR DESIGN JOURNEY
                <span className="project-detail-kicker-line" />
              </div>

              <div className="project-detail-title-row">
                <div>
                  <h1>{project.name}</h1>
                  <p className="project-detail-hero-description">
                    {project.description ||
                      "Your architectural project workspace. Everything starts with a vision."}
                  </p>
                </div>

                <span
                  className={`project-detail-status status-${status.toLowerCase()}`}
                >
                  <span className="project-detail-status-dot" />
                  {status}
                </span>
              </div>

              <div className="project-detail-actions">
                <Link
                  href={`/projects/${project.id}/edit`}
                  className="project-detail-edit-action"
                >
                  <Pencil size={17} />
                  Edit Project
                </Link>

                <Link
                  href={`/projects/${project.id}/design`}
                  className="project-detail-continue-action"
                >
                  <WandSparkles size={17} />
                  Continue Designing
                  <ArrowRight size={16} />
                </Link>
              </div>
              <div className="project-detail-hero-meta">
                <span>
                  <MapPin size={15} />
                  {location}
                </span>
                <span>
                  <CalendarDays size={15} />
                  Created {formatDate(project.created_at)}
                </span>
              </div>
            </div>

            <div className="project-detail-hero-decoration" aria-hidden="true">
              <House size={118} strokeWidth={0.8} />
            </div>
          </section>

          <section className="project-detail-metrics" aria-label="Project metrics">
            <article className="project-detail-metric">
              <span className="project-detail-metric-icon metric-location">
                <MapPin size={19} />
              </span>
              <div>
                <span className="project-detail-metric-label">Project location</span>
                <strong>{location}</strong>
                <small>Site information</small>
              </div>
            </article>

            <article className="project-detail-metric">
              <span className="project-detail-metric-icon metric-area">
                <Ruler size={19} />
              </span>
              <div>
                <span className="project-detail-metric-label">Land area</span>
                <strong>{landArea}</strong>
                <small>Land measurement</small>
              </div>
            </article>

            <article className="project-detail-metric">
              <span className="project-detail-metric-icon metric-floors">
                <Layers3 size={19} />
              </span>
              <div>
                <span className="project-detail-metric-label">Building height</span>
                <strong>{floors}</strong>
                <small>Planned floor count</small>
              </div>
            </article>
          </section>

          <div className="project-detail-section-heading">
            <div>
              <p className="project-wow-eyebrow">DESIGN & DEVELOPMENT</p>
              <h2>Project overview</h2>
            </div>
            <span className="project-detail-section-caption">
              Your project at a glance
            </span>
          </div>

          {selectedStyleName && (
            <section className="project-detail-house-style">
              <div className="project-detail-house-style-heading">
                <span className="project-detail-house-style-icon">
                  <Sparkles size={19} />
                </span>
                <div>
                  <p className="project-wow-eyebrow">ARCHITECTURAL DIRECTION</p>
                  <h3>Selected house style</h3>
                </div>
              </div>

              <div className="project-detail-house-style-content">
                {selectedStyleImage && (
                  <div className="project-detail-house-style-image">
                    <Image
                      src={selectedStyleImage}
                      alt={`${selectedStyleName} house style`}
                      fill
                      sizes="(max-width: 700px) 100vw, 320px"
                    />
                  </div>
                )}
                <div className="project-detail-house-style-copy">
                  <span>PROJECT DESIGN PREFERENCE</span>
                  <h4>{selectedStyleName}</h4>
                  <p>
                    This is the architectural style selected when the project
                    was created.
                  </p>
                  <Link href="/design-studio">Explore other styles</Link>
                </div>
              </div>
            </section>
          )}

          <div className="project-detail-overview-grid">
            <section className="project-detail-design-card">
              <div className="project-detail-card-top">
                <span className="project-detail-design-icon">
                  <Sparkles size={21} />
                </span>
                <span className="project-detail-card-tag">
                  CURRENT DESIGN
                </span>
              </div>

              <div className="project-detail-design-content">
                <p className="project-wow-eyebrow">DESIGN INFORMATION</p>
                <h3>{latestDesign?.name || "Initial design"}</h3>
                <p className="project-detail-design-description">
                  Your current design record and model information.
                </p>
              </div>

              <div className="project-detail-design-stats">
                <div>
                  <span>Version</span>
                  <strong>
                    {latestDesign?.version_number != null
                      ? `v${latestDesign.version_number}`
                      : "—"}
                  </strong>
                </div>
                <div>
                  <span>Design status</span>
                  <strong>{latestDesign?.status || "—"}</strong>
                </div>
              </div>
            </section>

            <section className="project-detail-metadata-card">
              <div className="project-detail-metadata-heading">
                <span className="project-detail-metadata-icon">
                  <Fingerprint size={19} />
                </span>
                <div>
                  <p className="project-wow-eyebrow">PROJECT RECORD</p>
                  <h3>Technical details</h3>
                </div>
              </div>

              <div className="project-detail-metadata-list">
                <div>
                  <span>Project ID</span>
                  <strong className="project-detail-id">{project.id}</strong>
                </div>
                <div>
                  <span>Last updated</span>
                  <strong>{formatDate(project.updated_at)}</strong>
                </div>
                <div className="project-detail-checksum-row">
                  <span>Model checksum</span>
                  <code>
                    {latestDesign?.model_checksum || "Not available"}
                  </code>
                </div>
              </div>
            </section>
          </div>

          <section className="project-detail-next-step">
            <div className="project-detail-next-icon">
              <Sparkles size={20} />
            </div>
            <div>
              <span className="project-wow-eyebrow">BUILT FOR YOUR NEXT STEP</span>
              <h3>Your project, ready to evolve.</h3>
              <p>
                This workspace provides the foundation for your house design.
                Floor planning, 3D visualization, CAD and BOQ/RAB workflows
                will be connected as development progresses.
              </p>
            </div>
          </section>
        </div>
      </div>
    </DashboardShell>
  );
}
