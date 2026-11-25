import Link from "next/link";
import Image from "next/image";
import DashboardShell from "@/components/dashboard-shell";
import { getProjects } from "@/lib/api/projects";
import {
  ArrowRight,
  Box,
  Calculator,
  ChevronRight,
  Clock3,
  FolderKanban,
  House,
  Layers3,
  Plus,
  Sparkles,
  WandSparkles,
} from "lucide-react";

const quickActions = [
  { title: "AI House Design", description: "Create a home from your land", icon: WandSparkles, href: "/design-studio" },
  { title: "Floor Planner", description: "Plan rooms and dimensions", icon: Layers3, href: "/floor-planner" },
  { title: "3D Visualization", description: "Explore your future home", icon: Box, href: "/visualization" },
  { title: "Quick RAB", description: "Estimate construction costs", icon: Calculator, href: "/boq-rab" },
];

const styles = [
  { name: "Modern Minimalist", detail: "Clean lines · Open spaces", tone: "style-minimal", image: "/assets/house-types/01_Rumah_Minimalis_Modern.jpg" },
  { name: "Tropical Modern", detail: "Natural light · Ventilation", tone: "style-tropical", image: "/assets/house-types/12_Rumah_Modern_Tropis.jpg" },
  { name: "Scandinavian", detail: "Warm · Simple · Functional", tone: "style-scandi", image: "/assets/house-types/03_Rumah_Scandinavian.jpg" },
  { name: "Japanese", detail: "Calm · Balanced · Natural", tone: "style-japanese", image: "/assets/house-types/06_Rumah_Jepang_Tradisional_Minka.jpg" },
];

function SectionTitle({
  eyebrow,
  title,
  description,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
}) {
  return (
    <div className="section-heading">
      <div>
        {eyebrow && <p className="eyebrow">{eyebrow}</p>}
        <h2>{title}</h2>
        {description && <p className="section-description">{description}</p>}
      </div>
    </div>
  );
}

export default async function DashboardPage() {
  const { items: projects, error: projectsError } = await getProjects();

  const stats = [
    {
      label: "Total Projects",
      value: String(projects.length),
      note: "Projects in your workspace",
      icon: FolderKanban,
    },
    {
      label: "Active Projects",
      value: String(projects.filter((project) => project.status === "active").length),
      note: "Currently active",
      icon: Layers3,
    },
    {
      label: "Archived Projects",
      value: String(projects.filter((project) => project.status === "archived").length),
      note: "Stored project records",
      icon: House,
    },
    {
      label: "Draft Projects",
      value: String(projects.filter((project) => project.status === "draft").length),
      note: "Work in progress",
      icon: Calculator,
    },
  ];

  return (
    <DashboardShell activeItem="Dashboard" breadcrumb="Dashboard">


        <div className="dashboard-content">
          <section className="welcome-row">
            <div>
              <p className="eyebrow">MONDAY, YOUR CREATIVE SPACE</p>
              <h1>Design a home that feels like <span>yours.</span></h1>
              <p className="welcome-description">
                From the first sketch to a detailed construction estimate,
                bring your dream home to life.
              </p>
            </div>
            <Link className="primary-button" href="/projects/new">
              <Plus size={18} /> Create New Project
            </Link>
          </section>

          <section className="stats-grid" aria-label="Project statistics">
            {stats.map((stat) => {
              const Icon = stat.icon;
              return (
                <article className="stat-card" key={stat.label}>
                  <div className="stat-top">
                    <span>{stat.label}</span>
                    <span className="stat-icon"><Icon size={18} /></span>
                  </div>
                  <strong className="stat-value">{stat.value}</strong>
                  <p>{stat.note}</p>
                </article>
              );
            })}
          </section>

          <section className="content-grid">
            <div className="primary-column">
              <section className="panel projects-panel" id="projects">
                <div className="panel-heading">
                  <SectionTitle
                    eyebrow="YOUR WORKSPACE"
                    title="Recent Projects"
                    description="Pick up where you left off."
                  />
                  <Link href="/design-studio" className="text-link">View all <ArrowRight size={15} /></Link>
                </div>

                {projectsError ? (
                  <div className="projects-state projects-error">
                    <h3>Projects temporarily unavailable</h3>
                    <p>
                      We could not load your projects. Please check the API
                      connection and refresh this page.
                    </p>
                  </div>
                ) : projects.length === 0 ? (
                <div className="empty-projects">
                  <div className="empty-illustration">
                    <div className="plan-sheet">
                      <div className="plan-room plan-room-a" />
                      <div className="plan-room plan-room-b" />
                      <div className="plan-room plan-room-c" />
                      <div className="plan-door" />
                      <div className="plan-window" />
                    </div>
                    <span className="illustration-spark"><Sparkles size={17} /></span>
                  </div>
                  <h3>Your first project begins here</h3>
                  <p>No projects yet. Start with your land dimensions and let&apos;s design a home around your life.</p>
                  <Link href="/design-studio" className="secondary-button">
                    Start your first design <ArrowRight size={16} />
                  </Link>
                </div>
                ) : (
                  <div className="project-list">
                    {projects.map((project) => (
                      <Link
                        href={`/projects/${encodeURIComponent(project.id)}`}
                        className="project-list-item"
                        key={project.id}
                        aria-label={`Open project ${project.name}`}
                      >
                        <div className="project-list-main">
                          <div className="project-list-title">
                            <h3>{project.name}</h3>
                            <span className={`project-status status-${project.status.toLowerCase()}`}>
                              {project.status}
                            </span>
                          </div>

                          {project.description && (
                            <p>{project.description}</p>
                          )}

                          <small>
                            Updated{" "}
                            {new Date(project.updated_at).toLocaleDateString("id-ID", {
                              day: "numeric",
                              month: "short",
                              year: "numeric",
                            })}
                          </small>
                        </div>
                      </Link>
                    ))}
                  </div>
                )}
              </section>

              <section className="panel quick-panel" id="design-studio">
                <div className="panel-heading">
                  <SectionTitle
                    eyebrow="GET THINGS DONE"
                    title="Quick Actions"
                    description="Jump straight into your design workflow."
                  />
                </div>
                <div className="quick-grid">
                  {quickActions.map((action) => {
                    const Icon = action.icon;
                    return (
                      <a href={action.href} className="quick-action" key={action.title}>
                        <span className="quick-icon"><Icon size={20} /></span>
                        <span className="quick-copy">
                          <strong>{action.title}</strong>
                          <small>{action.description}</small>
                        </span>
                        <ArrowRight className="quick-arrow" size={16} />
                      </a>
                    );
                  })}
                </div>
              </section>
            </div>

            <aside className="secondary-column">
              <section className="panel getting-started">
                <div className="panel-heading compact-heading">
                  <SectionTitle eyebrow="A SIMPLE WORKFLOW" title="Getting Started" />
                  <span className="step-count">0 / 4</span>
                </div>
                <div className="progress-track"><span /></div>
                <div className="workflow-list">
                  <div className="workflow-step current">
                    <span className="step-number">1</span>
                    <div><strong>Define your land</strong><small>Set the dimensions and location</small></div>
                    <ChevronRight size={16} />
                  </div>
                  <div className="workflow-step">
                    <span className="step-number">2</span>
                    <div><strong>Choose your home style</strong><small>Find a look that feels right</small></div>
                    <ChevronRight size={16} />
                  </div>
                  <div className="workflow-step">
                    <span className="step-number">3</span>
                    <div><strong>Plan your spaces</strong><small>Arrange rooms and floor plans</small></div>
                    <ChevronRight size={16} />
                  </div>
                  <div className="workflow-step">
                    <span className="step-number">4</span>
                    <div><strong>Explore your budget</strong><small>Build your BOQ and RAB</small></div>
                    <ChevronRight size={16} />
                  </div>
                </div>
                <Link href="/design-studio" className="workflow-cta">Start guided design <ArrowRight size={15} /></Link>
              </section>

              <section className="panel styles-panel">
                <div className="panel-heading">
                  <SectionTitle eyebrow="FIND YOUR INSPIRATION" title="Explore Styles" />
                </div>
                <div className="style-list">
                  {styles.map((style) => (
                    <Link href="/design-studio" className="style-item" key={style.name}>
                      <span className={`style-preview ${style.tone}`}>
  <Image
    src={style.image}
    alt={`${style.name} house design`}
    width={176}
    height={120}
    className="style-image"
    sizes="88px"
  />
</span>
                      <span className="style-copy"><strong>{style.name}</strong><small>{style.detail}</small></span>
                      <ChevronRight size={16} />
                    </Link>
                  ))}
                </div>
                <Link href="/design-studio" className="all-styles-link">Explore all design styles <ArrowRight size={14} /></Link>
              </section>

              <div className="help-banner" id="help">
                <div className="help-icon"><Clock3 size={18} /></div>
                <div><strong>Need a little guidance?</strong><p>We&apos;ll help you through every step.</p></div>
                <ArrowRight size={16} />
              </div>
            </aside>
          </section>

          <footer className="dashboard-footer">
            <span>© 2026 AI HOUSE GENERATOR</span>
            <span>From Land to Living.</span>
            <span>Built for better living.</span>
          </footer>
        </div>
      </DashboardShell>
  );
}
