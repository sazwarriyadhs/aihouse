import Image from "next/image";
import Link from "next/link";
import type { ReactNode } from "react";
import {
  ArrowRight, Bell, Box, Calculator, ChevronDown, ChevronRight,
  CircleHelp, Compass, FileText, FolderKanban, House, Layers3,
  Map, Menu, Search, Settings2, Sparkles, WandSparkles,
} from "lucide-react";

type DashboardShellProps = {
  children: ReactNode;
  activeItem: string;
  breadcrumb: string;
};

const navigation = [
  { label: "Dashboard", icon: House, href: "/" },
  { label: "My Projects", icon: FolderKanban, href: "/#projects" },
  { label: "AI Design Studio", icon: WandSparkles, href: "/#design-studio" },
  { label: "Floor Planner", icon: Layers3, href: "/floor-planner" },
  { label: "3D Visualization", icon: Box, href: "/visualization" },
  { label: "CAD & Drawings", icon: Compass, href: "/cad" },
  { label: "BOQ & RAB", icon: Calculator, href: "/boq-rab" },
  { label: "Documents", icon: FileText, href: "/documents" },
];

export default function DashboardShell({
  children,
  activeItem,
  breadcrumb,
}: DashboardShellProps) {
  return (
    <main className="app-shell">
      <aside className="sidebar">
        <Link href="/" className="brand">
          <Image src="/branding/dasboard_logo.png" alt="AI House Generator"
            width={176} height={56} priority className="brand-logo" />
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
            const active = item.label === activeItem;
            return (
              <Link key={item.label} href={item.href}
                className={`nav-item ${active ? "active" : ""}`}
                aria-current={active ? "page" : undefined}>
                <Icon size={18} strokeWidth={1.8} />
                <span>{item.label}</span>
                {active && <span className="active-indicator" />}
              </Link>
            );
          })}
        </nav>

        <div className="nav-label tools-label">PREFERENCES</div>
        <nav className="main-nav">
          <Link href="/materials" className="nav-item">
            <Map size={18} strokeWidth={1.8} />
            <span>Materials Library</span>
          </Link>
          <Link href="/#settings" className="nav-item">
            <Settings2 size={18} strokeWidth={1.8} />
            <span>Settings</span>
          </Link>
          <Link href="/#help" className="nav-item">
            <CircleHelp size={18} strokeWidth={1.8} />
            <span>Help & Support</span>
          </Link>
        </nav>

        <div className="sidebar-bottom">
          <div className="upgrade-card">
            <div className="upgrade-icon"><Sparkles size={17} /></div>
            <strong>Your next home starts here.</strong>
            <p>Turn your land into a thoughtful home design.</p>
            <Link href="/#design-studio">
              Explore AI Design <ArrowRight size={14} />
            </Link>
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
            <Image src="/branding/logo_kotak.png" alt="AHG"
              width={36} height={36} />
            <strong>AI HOUSE GENERATOR</strong>
          </div>
          <button className="mobile-menu" aria-label="Open navigation">
            <Menu size={21} />
          </button>
          <div className="breadcrumb">
            Workspace <ChevronRight size={15} /> <strong>{breadcrumb}</strong>
          </div>
          <div className="topbar-actions">
            <label className="global-search">
              <Search size={17} />
              <input type="search" placeholder="Search projects, designs..."
                aria-label="Search" />
              <kbd>⌘ K</kbd>
            </label>
            <button className="icon-button notification-button"
              aria-label="Notifications">
              <Bell size={19} />
              <span />
            </button>
            <div className="topbar-divider" />
            <button className="profile-button">
              <span className="profile-avatar">A</span>
              <span className="profile-info">
                <strong>My Account</strong>
                <small>Workspace owner</small>
              </span>
              <ChevronDown size={15} />
            </button>
          </div>
        </header>
        {children}
      </section>
    </main>
  );
}