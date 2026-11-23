"use client";
import Image from "next/image";

import { useMemo, useState } from "react";
import Link from "next/link";
import {
  ArrowLeft, ArrowRight, Bell, Box, Calculator, Check,
  ChevronRight, CircleHelp, Download, DraftingCompass,
  Eye, FileText, FolderKanban, Grid3X3, House,
  Layers3, LibraryBig, Menu, Plus, RotateCcw,
  Ruler, Search, Settings2, Trash2, WandSparkles
} from "lucide-react";

type Room = {
  id: number;
  name: string;
  width: number;
  depth: number;
  color: string;
};

const modules = {
  "floor-planner": {
    title: "Floor Planner",
    subtitle: "Plan spaces, arrange rooms, and define dimensions.",
    icon: Layers3,
    eyebrow: "DESIGN WORKSPACE",
  },
  visualization: {
    title: "3D Visualization",
    subtitle: "Explore a conceptual view of your future home.",
    icon: Box,
    eyebrow: "VISUALIZATION STUDIO",
  },
  cad: {
    title: "CAD & Drawings",
    subtitle: "Organize technical drawing outputs and CAD exports.",
    icon: DraftingCompass,
    eyebrow: "TECHNICAL DOCUMENTATION",
  },
  "boq-rab": {
    title: "BOQ & RAB",
    subtitle: "Review quantities and estimate construction costs.",
    icon: Calculator,
    eyebrow: "COST ESTIMATION",
  },
  documents: {
    title: "Documents",
    subtitle: "Keep project documents and revisions organized.",
    icon: FileText,
    eyebrow: "PROJECT DOCUMENTS",
  },
  materials: {
    title: "Materials Library",
    subtitle: "Explore material specifications and reference prices.",
    icon: LibraryBig,
    eyebrow: "REFERENCE LIBRARY",
  },
} as const;

const initialRooms: Room[] = [
  { id: 1, name: "Living Room", width: 5, depth: 4, color: "#DCE9F8" },
  { id: 2, name: "Bedroom 1", width: 4, depth: 3.5, color: "#E9E3F7" },
  { id: 3, name: "Kitchen", width: 3, depth: 3, color: "#F7EBD6" },
  { id: 4, name: "Bathroom", width: 2, depth: 2, color: "#DDEFEA" },
];

const materials = [
  { name: "Concrete K-250", category: "Structure", unit: "m³", price: 1100000 },
  { name: "Reinforcing Steel", category: "Structure", unit: "kg", price: 14500 },
  { name: "Red Brick", category: "Wall", unit: "pcs", price: 1100 },
  { name: "Lightweight Brick", category: "Wall", unit: "m³", price: 750000 },
  { name: "Ceramic Floor Tile", category: "Finishing", unit: "m²", price: 95000 },
  { name: "Interior Wall Paint", category: "Finishing", unit: "L", price: 65000 },
  { name: "Roof Tile", category: "Roofing", unit: "pcs", price: 4500 },
  { name: "Aluminium Window", category: "Opening", unit: "m²", price: 850000 },
];

const rupiah = (value: number) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value);

function Sidebar({ active }: { active: string }) {
  const links = [
    { label: "Dashboard", href: "/", icon: House },
    { label: "My Projects", href: "/#projects", icon: FolderKanban },
    { label: "AI Design Studio", href: "/#design-studio", icon: WandSparkles },
    { label: "Floor Planner", href: "/floor-planner", icon: Layers3 },
    { label: "3D Visualization", href: "/visualization", icon: Box },
    { label: "CAD & Drawings", href: "/cad", icon: DraftingCompass },
    { label: "BOQ & RAB", href: "/boq-rab", icon: Calculator },
    { label: "Documents", href: "/documents", icon: FileText },
  ];

  return (
    <aside className="module-sidebar">
      <Link href="/" className="module-brand">
        <Image
          src="/branding/dasboard_logo.png"
          alt="AI House Generator"
          width={176}
          height={56}
          priority
          className="brand-logo"
        />
      </Link>

      <div className="module-workspace">
        <span className="module-avatar">A</span>
        <span><strong>My Workspace</strong><small>Personal account</small></span>
      </div>

      <p className="module-nav-label">WORKSPACE</p>
      <nav className="module-nav">
        {links.map((item) => {
          const Icon = item.icon;
          const selected = item.href === `/${active}`;
          return (
            <Link
              key={item.label}
              href={item.href}
              className={`module-nav-item ${selected ? "active" : ""}`}
            >
              <Icon size={17} />
              <span>{item.label}</span>
              {selected && <span className="module-nav-indicator" />}
            </Link>
          );
        })}
      </nav>

      <p className="module-nav-label module-preferences">PREFERENCES</p>
      <nav className="module-nav">
        <Link href="/materials" className={`module-nav-item ${active === "materials" ? "active" : ""}`}>
          <LibraryBig size={17} /><span>Materials Library</span>
        </Link>
        <Link href="/#settings" className="module-nav-item">
          <Settings2 size={17} /><span>Settings</span>
        </Link>
        <Link href="/#help" className="module-nav-item">
          <CircleHelp size={17} /><span>Help & Support</span>
        </Link>
      </nav>

      <div className="module-sidebar-bottom">
        <div className="module-promo">
          <WandSparkles size={19} />
          <strong>Your next home starts here.</strong>
          <p>From land to living. Bring your ideas to life.</p>
          <Link href="/">Back to Dashboard <ArrowRight size={13} /></Link>
        </div>
        <small className="module-system-status">
          <span /> Workspace preview
        </small>
      </div>
    </aside>
  );
}

function FloorPlanner() {
  const [rooms, setRooms] = useState(initialRooms);
  const [selected, setSelected] = useState(1);

  const selectedRoom = rooms.find((room) => room.id === selected);
  const totalArea = rooms.reduce((sum, room) => sum + room.width * room.depth, 0);

  function addRoom() {
    const id = Math.max(0, ...rooms.map((room) => room.id)) + 1;
    const room = {
      id,
      name: `New Room ${id}`,
      width: 3,
      depth: 3,
      color: "#E5ECF5",
    };
    setRooms((current) => [...current, room]);
    setSelected(id);
  }

  function updateRoom(field: "name" | "width" | "depth", value: string) {
    setRooms((current) =>
      current.map((room) =>
        room.id === selected
          ? {
              ...room,
              [field]: field === "name" ? value : Math.max(0.5, Number(value) || 0.5),
            }
          : room
      )
    );
  }

  function removeRoom() {
    setRooms((current) => current.filter((room) => room.id !== selected));
    setSelected(rooms.find((room) => room.id !== selected)?.id ?? 0);
  }

  return (
    <>
      <div className="module-metrics">
        <article><span>Total Rooms</span><strong>{rooms.length}</strong><small>Current layout</small></article>
        <article><span>Estimated Floor Area</span><strong>{totalArea.toFixed(2)} m²</strong><small>Sum of room areas</small></article>
        <article><span>Selected Room</span><strong>{selectedRoom?.name ?? "None"}</strong><small>Editable properties</small></article>
      </div>

      <div className="planner-layout">
        <section className="module-panel planner-canvas-panel">
          <div className="module-panel-heading">
            <div><h3>2D Floor Plan</h3><p>Conceptual room arrangement · Not to construction scale</p></div>
            <span className="module-tag"><Grid3X3 size={13} /> Grid preview</span>
          </div>

          <div className="floor-canvas">
            <svg viewBox="0 0 620 420" role="img" aria-label="Conceptual room layout">
              <defs>
                <pattern id="floorGrid" width="20" height="20" patternUnits="userSpaceOnUse">
                  <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#E8EDF4" strokeWidth="1" />
                </pattern>
              </defs>
              <rect width="620" height="420" fill="url(#floorGrid)" />
              {rooms.map((room, index) => {
                const columns = 2;
                const col = index % columns;
                const row = Math.floor(index / columns);
                const x = 65 + col * 255;
                const y = 45 + row * 150;
                const w = Math.max(70, Math.min(220, room.width * 36));
                const h = Math.max(55, Math.min(125, room.depth * 25));
                return (
                  <g key={room.id} onClick={() => setSelected(room.id)} style={{ cursor: "pointer" }}>
                    <rect
                      x={x} y={y} width={w} height={h}
                      rx="3"
                      fill={room.color}
                      stroke={selected === room.id ? "#1070FF" : "#8B9CB2"}
                      strokeWidth={selected === room.id ? 3 : 1.5}
                    />
                    <text x={x + 10} y={y + 23} fontSize="12" fontWeight="700" fill="#001030">
                      {room.name.slice(0, 22)}
                    </text>
                    <text x={x + 10} y={y + 43} fontSize="10" fill="#53657C">
                      {room.width} × {room.depth} m
                    </text>
                  </g>
                );
              })}
              {rooms.length === 0 && (
                <text x="310" y="205" textAnchor="middle" fontSize="14" fill="#718096">
                  Add a room to begin
                </text>
              )}
            </svg>
          </div>

          <div className="module-canvas-footer">
            <span><Grid3X3 size={14} /> Conceptual layout</span>
            <span><Ruler size={14} /> Units: meters</span>
            <button className="module-quiet-button" onClick={() => { setRooms(initialRooms); setSelected(1); }}>
              <RotateCcw size={14} /> Reset demo
            </button>
          </div>
        </section>

        <aside className="module-panel room-panel">
          <div className="module-panel-heading">
            <div><h3>Room List</h3><p>Select a room to edit</p></div>
            <button className="module-square-button" onClick={addRoom} aria-label="Add room"><Plus size={17} /></button>
          </div>

          <div className="room-list">
            {rooms.map((room) => (
              <button
                key={room.id}
                className={`room-list-item ${selected === room.id ? "selected" : ""}`}
                onClick={() => setSelected(room.id)}
              >
                <span className="room-color" style={{ background: room.color }} />
                <span><strong>{room.name}</strong><small>{(room.width * room.depth).toFixed(2)} m²</small></span>
                <ChevronRight size={15} />
              </button>
            ))}
          </div>

          {selectedRoom && (
            <div className="room-properties">
              <h4>Room Properties</h4>
              <label>Room name
                <input value={selectedRoom.name} onChange={(e) => updateRoom("name", e.target.value)} />
              </label>
              <div className="room-dimensions">
                <label>Width (m)
                  <input type="number" min="0.5" step="0.5" value={selectedRoom.width}
                    onChange={(e) => updateRoom("width", e.target.value)} />
                </label>
                <label>Depth (m)
                  <input type="number" min="0.5" step="0.5" value={selectedRoom.depth}
                    onChange={(e) => updateRoom("depth", e.target.value)} />
                </label>
              </div>
              <button className="module-danger-button" onClick={removeRoom}>
                <Trash2 size={14} /> Remove room
              </button>
            </div>
          )}
        </aside>
      </div>
    </>
  );
}

function Visualization() {
  const [view, setView] = useState<"exterior" | "interior">("exterior");
  const [rotation, setRotation] = useState(25);
  const [showGrid, setShowGrid] = useState(true);

  return (
    <div className="visualization-layout">
      <section className="module-panel visual-panel">
        <div className="module-panel-heading">
          <div><h3>3D Model Preview</h3><p>Illustrative massing model · Not a construction model</p></div>
          <span className="module-tag"><Eye size={13} /> Preview</span>
        </div>
        <div className={`visual-stage ${showGrid ? "with-grid" : ""}`}>
          <div className={`house-model ${view}`}>
            <div className="house-roof" />
            <div className="house-wall front"><span className="house-window" /><span className="house-door" /></div>
            <div className="house-wall side"><span className="house-side-window" /></div>
            <div className="house-base" />
          </div>
          <span className="visual-stage-label">CONCEPTUAL 3D MODEL</span>
        </div>
        <div className="visual-controls">
          <div className="visual-tabs">
            <button className={view === "exterior" ? "active" : ""} onClick={() => setView("exterior")}>Exterior</button>
            <button className={view === "interior" ? "active" : ""} onClick={() => setView("interior")}>Interior concept</button>
          </div>
          <label className="rotation-control">View angle
            <input type="range" min="0" max="60" value={rotation} onChange={(e) => setRotation(Number(e.target.value))} />
            <span>{rotation}°</span>
          </label>
          <label className="grid-toggle"><input type="checkbox" checked={showGrid} onChange={(e) => setShowGrid(e.target.checked)} /> Show grid</label>
        </div>
      </section>
      <aside className="module-panel visual-info">
        <h3>Scene Settings</h3>
        <p>Preview controls</p>
        <div className="info-row"><span>Model status</span><strong>Illustrative</strong></div>
        <div className="info-row"><span>View mode</span><strong>{view === "exterior" ? "Exterior" : "Interior concept"}</strong></div>
        <div className="info-row"><span>Camera angle</span><strong>{rotation}°</strong></div>
        <div className="info-row"><span>Grid</span><strong>{showGrid ? "Visible" : "Hidden"}</strong></div>
        <div className="module-notice">The production 3D viewer will load geometry from the Canonical Design Model and GLB export worker.</div>
      </aside>
    </div>
  );
}

function CADModule() {
  const [format, setFormat] = useState("FCStd");
  const [prepared, setPrepared] = useState(false);
  const drawings = [
    { name: "Architectural Floor Plan", type: "2D drawing", status: "Concept only" },
    { name: "Building Elevations", type: "Elevation", status: "Not generated" },
    { name: "Building Sections", type: "Section", status: "Not generated" },
    { name: "3D CAD Model", type: "3D model", status: "Not generated" },
  ];

  return (
    <section className="module-panel">
      <div className="module-panel-heading">
        <div><h3>Drawing Register</h3><p>Technical output register for this design workspace</p></div>
        <span className="module-tag">CAD workspace</span>
      </div>
      <div className="module-table-wrap">
        <table className="module-table">
          <thead><tr><th>Drawing / Output</th><th>Category</th><th>Status</th></tr></thead>
          <tbody>{drawings.map((item) => (
            <tr key={item.name}>
              <td><strong>{item.name}</strong></td>
              <td>{item.type}</td>
              <td><span className="status-pill">{item.status}</span></td>
            </tr>
          ))}</tbody>
        </table>
      </div>
      <div className="cad-export-box">
        <div><h4>Export Configuration</h4><p>Choose a target format for a future CAD worker export.</p></div>
        <label>Output format
          <select value={format} onChange={(e) => { setFormat(e.target.value); setPrepared(false); }}>
            <option value="FCStd">FreeCAD project (.FCStd)</option>
            <option value="DXF">Drawing exchange (.DXF)</option>
            <option value="STEP">3D exchange (.STEP)</option>
            <option value="IFC">Building information model (.IFC)</option>
          </select>
        </label>
        <button className="module-primary-button" onClick={() => setPrepared(true)}>
          <Download size={16} /> Prepare export
        </button>
      </div>
      {prepared && <div className="module-notice"><Check size={15} /> Export configuration selected: {format}. File generation is not connected yet; no file has been created.</div>}
    </section>
  );
}

function BOQModule() {
  const [items, setItems] = useState([
    { id: 1, name: "Concrete K-250", unit: "m³", quantity: 5, price: 1100000 },
    { id: 2, name: "Reinforcing Steel", unit: "kg", quantity: 650, price: 14500 },
    { id: 3, name: "Red Brick", unit: "pcs", quantity: 3500, price: 1100 },
  ]);

  const total = items.reduce((sum, item) => sum + item.quantity * item.price, 0);

  function updateItem(id: number, field: "quantity" | "price", value: string) {
    setItems((current) => current.map((item) => item.id === id
      ? { ...item, [field]: Math.max(0, Number(value) || 0) }
      : item
    ));
  }

  function addItem() {
    const id = Math.max(0, ...items.map((item) => item.id)) + 1;
    setItems((current) => [...current, {
      id, name: `New item ${id}`, unit: "unit", quantity: 1, price: 0,
    }]);
  }

  return (
    <>
      <div className="module-metrics">
        <article><span>BOQ Items</span><strong>{items.length}</strong><small>Editable demo items</small></article>
        <article><span>Estimated Subtotal</span><strong>{rupiah(total)}</strong><small>Quantity × unit price</small></article>
        <article><span>Calculation Mode</span><strong>Quick Estimate</strong><small>Before taxes and overheads</small></article>
      </div>
      <section className="module-panel">
        <div className="module-panel-heading">
          <div><h3>Bill of Quantities</h3><p>Adjust quantity and unit price to recalculate the subtotal</p></div>
          <button className="module-secondary-button" onClick={addItem}><Plus size={15} /> Add item</button>
        </div>
        <div className="module-table-wrap">
          <table className="module-table boq-table">
            <thead><tr><th>Work / Material</th><th>Unit</th><th>Quantity</th><th>Unit Price (IDR)</th><th>Subtotal</th><th></th></tr></thead>
            <tbody>{items.map((item) => (
              <tr key={item.id}>
                <td><strong>{item.name}</strong></td>
                <td>{item.unit}</td>
                <td><input aria-label={`Quantity ${item.name}`} type="number" min="0" value={item.quantity} onChange={(e) => updateItem(item.id, "quantity", e.target.value)} /></td>
                <td><input aria-label={`Price ${item.name}`} type="number" min="0" value={item.price} onChange={(e) => updateItem(item.id, "price", e.target.value)} /></td>
                <td><strong>{rupiah(item.quantity * item.price)}</strong></td>
                <td><button className="module-delete-icon" aria-label={`Remove ${item.name}`} onClick={() => setItems((current) => current.filter((x) => x.id !== item.id))}><Trash2 size={15} /></button></td>
              </tr>
            ))}</tbody>
          </table>
        </div>
        <div className="boq-total"><span>Estimated subtotal</span><strong>{rupiah(total)}</strong></div>
        <div className="module-notice">Demonstration calculation only. Production RAB requires validated quantities, regional AHSP, price sources, effective dates, overhead, taxes, and review. Missing prices must not be treated as zero.</div>
      </section>
    </>
  );
}

function DocumentsModule() {
  const docs = [
    { name: "Project Design Brief", category: "Design", detail: "Project requirements and design intent" },
    { name: "Floor Plan Drawing", category: "Drawing", detail: "Conceptual 2D floor plan" },
    { name: "BOQ & RAB Summary", category: "Cost estimate", detail: "Preliminary quantity and cost estimate" },
    { name: "Material Specifications", category: "Reference", detail: "Material selections and specifications" },
  ];
  const [filter, setFilter] = useState("All");
  const categories = ["All", "Design", "Drawing", "Cost estimate", "Reference"];
  const filtered = docs.filter((doc) => filter === "All" || doc.category === filter);

  return (
    <section className="module-panel">
      <div className="module-panel-heading">
        <div><h3>Project Document Register</h3><p>Organize project outputs and document revisions</p></div>
        <span className="module-tag">{filtered.length} entries</span>
      </div>
      <div className="document-filters">
        {categories.map((category) => (
          <button key={category} className={filter === category ? "active" : ""} onClick={() => setFilter(category)}>{category}</button>
        ))}
      </div>
      <div className="document-list">
        {filtered.map((doc) => (
          <article className="document-row" key={doc.name}>
            <span className="document-icon"><FileText size={20} /></span>
            <span className="document-copy"><strong>{doc.name}</strong><small>{doc.detail}</small></span>
            <span className="status-pill">Demo entry</span>
          </article>
        ))}
      </div>
      <div className="module-notice">These are sample document-register entries. No project files have been uploaded or generated.</div>
    </section>
  );
}

function MaterialsModule() {
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("All");
  const categories = ["All", ...Array.from(new Set(materials.map((item) => item.category)))];

  const filtered = useMemo(() => materials.filter((item) =>
    (category === "All" || item.category === category) &&
    item.name.toLowerCase().includes(query.toLowerCase())
  ), [category, query]);

  return (
    <section className="module-panel">
      <div className="module-panel-heading">
        <div><h3>Material Catalogue</h3><p>Reference catalogue · Indicative demo prices, not supplier quotations</p></div>
        <span className="module-tag">{filtered.length} materials</span>
      </div>
      <div className="material-tools">
        <label className="material-search"><Search size={17} /><input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search materials..." /></label>
        <select value={category} onChange={(e) => setCategory(e.target.value)}>
          {categories.map((item) => <option key={item}>{item}</option>)}
        </select>
      </div>
      <div className="material-grid">
        {filtered.map((item) => (
          <article className="material-card" key={item.name}>
            <span className="material-card-icon"><LibraryBig size={20} /></span>
            <span className="material-category">{item.category}</span>
            <h4>{item.name}</h4>
            <p>Unit: {item.unit}</p>
            <strong>{rupiah(item.price)} <small>/ {item.unit}</small></strong>
          </article>
        ))}
        {filtered.length === 0 && <p className="module-empty">No materials match your search.</p>}
      </div>
      <div className="module-notice">Prices are illustrative placeholders. Production material prices must record region, supplier/source, date, and validity period.</div>
    </section>
  );
}

export default function WorkspaceModule({ moduleKey }: { moduleKey: string }) {
  const config = modules[moduleKey as keyof typeof modules];
  const [mobileOpen, setMobileOpen] = useState(false);

  if (!config) return null;
  const Icon = config.icon;

  return (
    <main className="module-app-shell">
      <div className={mobileOpen ? "module-mobile-overlay open" : "module-mobile-overlay"} onClick={() => setMobileOpen(false)} />
      <div className={mobileOpen ? "module-sidebar-wrap open" : "module-sidebar-wrap"}>
        <Sidebar active={moduleKey} />
      </div>

      <section className="module-main">
        <header className="module-topbar">
          <button className="module-mobile-menu" onClick={() => setMobileOpen(!mobileOpen)} aria-label="Toggle navigation"><Menu size={20} /></button>
          <div className="module-breadcrumb"><Link href="/">Workspace</Link><ChevronRight size={14} /><strong>{config.title}</strong></div>
          <div className="module-topbar-right"><span className="module-workspace-indicator">Personal Workspace</span><button className="module-top-icon" aria-label="Notifications"><Bell size={18} /></button><span className="module-profile-avatar">A</span></div>
        </header>

        <div className="module-content">
          <div className="module-heading">
            <div>
              <p className="module-eyebrow">{config.eyebrow}</p>
              <div className="module-title-line"><span className="module-title-icon"><Icon size={22} /></span><h1>{config.title}</h1></div>
              <p className="module-subtitle">{config.subtitle}</p>
            </div>
            <Link href="/" className="module-back-link"><ArrowLeft size={15} /> Dashboard</Link>
          </div>

          {moduleKey === "floor-planner" && <FloorPlanner />}
          {moduleKey === "visualization" && <Visualization />}
          {moduleKey === "cad" && <CADModule />}
          {moduleKey === "boq-rab" && <BOQModule />}
          {moduleKey === "documents" && <DocumentsModule />}
          {moduleKey === "materials" && <MaterialsModule />}

          <footer className="module-footer">
            <span>AI HOUSE GENERATOR</span>
            <span>From Land to Living.</span>
            <span>Workspace module preview</span>
          </footer>
        </div>
      </section>
    </main>
  );
}
