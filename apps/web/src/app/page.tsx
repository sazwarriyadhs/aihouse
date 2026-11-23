import Link from "next/link";
import Image from "next/image";
import { getProjects } from "@/lib/api/projects";
import {
  ArrowRight,
  Bell,
  Box,
  Calculator,
  ChevronDown,
  ChevronRight,
  CircleHelp,
  Clock3,
  Compass,
  FileText,
  FolderKanban,
  House,
  Layers3,
  Map,
  Menu,
  Plus,
  Search,
  Settings2,
  Sparkles,
  WandSparkles,
} from "lucide-react";

const navigation = [
  { label: "Dashboard", icon: House, active: true, href: "/" },
  { label: "My Projects", icon: FolderKanban, href: "#projects" },
  { label: "AI Design Studio", icon: WandSparkles, href: "#design-studio" },
  { label: "Floor Planner", icon: Layers3, href: "/floor-planner" },
  { label: "3D Visualization", icon: Box, href: "/visualization" },
  { label: "CAD & Drawings", icon: Compass, href: "/cad" },
  { label: "BOQ & RAB", icon: Calculator, href: "/boq-rab" },
  { label: "Documents", icon: FileText, href: "/documents" },
];

const quickActions = [
  { title: "AI House Design", description: "Create a home from your land", icon: WandSparkles, href: "#design-studio" },
  { title: "Floor Planner", description: "Plan rooms and dimensions", icon: Layers3, href: "/floor-planner" },
  { title: "3D Visualization", description: "Explore your future home", icon: Box, href: "/visualization" },
  { title: "Quick RAB", description: "Estimate construction costs", icon: Calculator, href: "/boq-rab" },
];

const styles = [
  { name: "Modern Minimalist", detail: "Clean lines · Open spaces", tone: "style-minimal" },
  { name: "Tropical Modern", detail: "Natural light · Ventilation", tone: "style-tropical" },
  { name: "Scandinavian", detail: "Warm · Simple · Functional", tone: "style-scandi" },
  { name: "Japanese", detail: "Calm · Balanced · Natural", tone: "style-japanese" },
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
    <main className="app-shell">
      <aside className="sidebar">
        <Link href="/" className="brand">
          <Image
            src="/branding/dasboard_logo.png"
            alt="AI House Generator"
            width={176}
            height={56}
            priority
            className="brand-logo"
          />
        </Link>

        <div className="workspace-switcher">
          <div className="workspace-avatar">A</div>
          <div className="workspace-copy">
            <strong>My Workspace</strong>
            <span>Personal account</span>
          </div>
          <ChevronDown size={16} />
        </div>

        <div className="nav-label">WORKSPACE</div>
        <nav className="main-nav" aria-label="Main navigation">
          {navigation.map((item) => {
            const Icon = item.icon;
            return (
              <a
                key={item.label}
                href={item.href}
                className={`nav-item ${item.active ? "active" : ""}`}
                aria-current={item.active ? "page" : undefined}
              >
                <Icon size={18} strokeWidth={1.8} />
                <span>{item.label}</span>
                {item.active && <span className="active-indicator" />}
              </a>
            );
          })}
        </nav>

        <div className="nav-label tools-label">PREFERENCES</div>
        <nav className="main-nav">
          <Link href="/materials" className="nav-item">
            <Map size={18} strokeWidth={1.8} />
            <span>Materials Library</span>
          </Link>
          <a href="#settings" className="nav-item">
            <Settings2 size={18} strokeWidth={1.8} />
            <span>Settings</span>
          </a>
          <a href="#help" className="nav-item">
            <CircleHelp size={18} strokeWidth={1.8} />
            <span>Help & Support</span>
          </a>
        </nav>

        <div className="sidebar-bottom">
          <div className="upgrade-card">
            <div className="upgrade-icon"><Sparkles size={17} /></div>
            <strong>Your next home starts here.</strong>
            <p>Turn your land into a thoughtful home design.</p>
            <a href="#design-studio">Explore AI Design <ArrowRight size={14} /></a>
          </div>
          <div className="sidebar-footer">
            <span className="online-dot" />
            All systems operational
          </div>
        </div>
      </aside>

      <section className="main-area">
        <header className="topbar">
          <div className="mobile-brand">
            <Image src="/branding/logo_kotak.png" alt="AHG" width={36} height={36} />
            <strong>AI HOUSE GENERATOR</strong>
          </div>
          <button className="mobile-menu" aria-label="Open navigation"><Menu size={21} /></button>
          <div className="breadcrumb">
            Workspace <ChevronRight size={15} /> <strong>Dashboard</strong>
          </div>
          <div className="topbar-actions">
            <label className="global-search">
              <Search size={17} />
              <input type="search" placeholder="Search projects, designs..." aria-label="Search" />
              <kbd>⌘ K</kbd>
            </label>
            <button className="icon-button notification-button" aria-label="Notifications">
              <Bell size={19} />
              <span />
            </button>
            <div className="topbar-divider" />
            <button className="profile-button">
              <span className="profile-avatar">A</span>
              <span className="profile-info"><strong>My Account</strong><small>Workspace owner</small></span>
              <ChevronDown size={15} />
            </button>
          </div>
        </header>

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
                  <a href="#design-studio" className="text-link">View all <ArrowRight size={15} /></a>
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
                  <a href="#design-studio" className="secondary-button">
                    Start your first design <ArrowRight size={16} />
                  </a>
                </div>
                ) : (
                  <div className="project-list">
                    {projects.map((project) => (
                      <article className="project-list-item" key={project.id}>
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
                      </article>
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
                <a href="#design-studio" className="workflow-cta">Start guided design <ArrowRight size={15} /></a>
              </section>

              <section className="panel styles-panel">
                <div className="panel-heading">
                  <SectionTitle eyebrow="FIND YOUR INSPIRATION" title="Explore Styles" />
                </div>
                <div className="style-list">
                  {styles.map((style) => (
                    <a href="#design-studio" className="style-item" key={style.name}>
                      <span className={`style-preview ${style.tone}`}>
                        <span /><span /><span />
                      </span>
                      <span className="style-copy"><strong>{style.name}</strong><small>{style.detail}</small></span>
                      <ChevronRight size={16} />
                    </a>
                  ))}
                </div>
                <a href="#design-studio" className="all-styles-link">Explore all design styles <ArrowRight size={14} /></a>
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
      </section>
    </main>
  );
}

