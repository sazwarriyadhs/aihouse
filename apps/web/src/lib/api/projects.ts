export interface Project {
  id: string;
  name: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectInput {
  name: string;
  description: string;
  location: string;
  land_width_m: number;
  land_depth_m: number;
  orientation: string;
  floors: number;
  design_style_id?: string;
  design_style_name?: string;
  design_style_image?: string;
}

export interface CreatedProject {
  project_id: string;
  design_id: string;
  version_id: string;
  canonical_model: Record<string, unknown>;
}

export interface ProjectDetail {
  project: Project;
  designs: Array<{
    id: string;
    name: string;
    current_version: number;
    status: string;
    version_id: string;
    version_number: number;
    canonical_model: Record<string, unknown>;
    model_checksum: string;
  }>;
}

interface ProjectsListResponse {
  items: Project[];
  count: number;
}

const API_BASE_URL = (
  process.env.API_BASE_URL || "http://127.0.0.1:8097"
).replace(/\/+$/, "");

export async function getProjects(): Promise<{
  items: Project[];
  error: string | null;
}> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/v1/projects`, {
      headers: { Accept: "application/json" },
      cache: "no-store",
    });

    if (!response.ok) {
      return {
        items: [],
        error: `Projects API returned HTTP ${response.status}`,
      };
    }

    const payload = (await response.json()) as ProjectsListResponse;

    if (!payload || !Array.isArray(payload.items)) {
      return { items: [], error: "Invalid Projects API response" };
    }

    return { items: payload.items, error: null };
  } catch {
    return { items: [], error: "Unable to connect to the Projects API" };
  }
}

export async function createProject(
  input: CreateProjectInput,
): Promise<{ data: CreatedProject | null; error: string | null }> {
  try {
    const response = await fetch("/api/projects", {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify(input),
      cache: "no-store",
    });

    const payload = await response.json().catch(() => null);

    if (!response.ok) {
      return {
        data: null,
        error:
          typeof payload?.error === "string"
            ? payload.error
            : `Project creation failed (HTTP ${response.status})`,
      };
    }

    if (
      !payload ||
      typeof payload.project_id !== "string" ||
      typeof payload.design_id !== "string"
    ) {
      return { data: null, error: "Invalid create-project response" };
    }

    return { data: payload as CreatedProject, error: null };
  } catch {
    return { data: null, error: "Unable to connect to the Projects API" };
  }
}
export async function getProject(
  id: string,
): Promise<{ data: ProjectDetail | null; error: string | null }> {
  try {
    const response = await fetch(
      `${API_BASE_URL}/api/v1/projects/${encodeURIComponent(id)}`,
      {
        headers: { Accept: "application/json" },
        cache: "no-store",
      },
    );

    if (!response.ok) {
      return {
        data: null,
        error:
          response.status === 404
            ? "Project not found"
            : `Project API returned HTTP ${response.status}`,
      };
    }

    const payload = (await response.json()) as ProjectDetail;

    if (!payload?.project || !Array.isArray(payload.designs)) {
      return { data: null, error: "Invalid project detail response" };
    }

    return { data: payload, error: null };
  } catch {
    return { data: null, error: "Unable to connect to the Projects API" };
  }
}

export async function updateProject(
  id: string,
  input: CreateProjectInput,
): Promise<{ data: CreatedProject | null; error: string | null }> {
  try {
    const response = await fetch(
      `/api/projects/${encodeURIComponent(id)}`,
      {
        method: "PUT",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(input),
        cache: "no-store",
      },
    );

    const payload = await response.json().catch(() => null);

    if (!response.ok) {
      return {
        data: null,
        error:
          typeof payload?.error === "string"
            ? payload.error
            : `Project update failed (HTTP ${response.status})`,
      };
    }

    if (
      !payload ||
      typeof payload.project_id !== "string" ||
      typeof payload.design_id !== "string" ||
      typeof payload.version_id !== "string"
    ) {
      return { data: null, error: "Invalid update-project response" };
    }

    return { data: payload as CreatedProject, error: null };
  } catch {
    return { data: null, error: "Unable to connect to the Projects API" };
  }
}

// Floor Planning API contracts.
export interface FloorPlanRoom {
  id: string;
  name: string;
  type: string;
  x_m: number;
  y_m: number;
  width_m: number;
  depth_m: number;
}

export interface FloorPlanWall {
  id: string;
  x1_m: number;
  y1_m: number;
  x2_m: number;
  y2_m: number;
  thickness_m: number;
}

export interface FloorPlanOpening {
  id: string;
  wall_id: string;
  offset_m: number;
  width_m: number;
  height_m: number;
  sill_height_m?: number;
}

export interface FloorPlan {
  floor_number: number;
  name: string;
  rooms: FloorPlanRoom[];
  walls: FloorPlanWall[];
  doors: FloorPlanOpening[];
  windows: FloorPlanOpening[];
}

export interface FloorPlansResponse {
  project_id: string;
  version_id: string;
  version_number: number;
  model_checksum: string;
  floor_plans: FloorPlan[];
}

export interface SavedFloorPlansResponse {
  project_id: string;
  design_id: string;
  version_id: string;
  version_number: number;
  model_checksum: string;
  floor_plans: FloorPlan[];
}

async function readFloorPlanError(
  response: Response,
  fallback: string,
): Promise<string> {
  const payload = await response.json().catch(() => null);

  return typeof payload?.error === "string"
    ? payload.error
    : `${fallback} (HTTP ${response.status})`;
}

export async function getFloorPlans(
  id: string,
): Promise<{ data: FloorPlansResponse | null; error: string | null }> {
  try {
    const response = await fetch(
      `/api/projects/${encodeURIComponent(id)}/floor-plans`,
      {
        headers: { Accept: "application/json" },
        cache: "no-store",
      },
    );

    if (!response.ok) {
      return {
        data: null,
        error: await readFloorPlanError(response, "Could not load floor plans"),
      };
    }

    const payload = (await response.json()) as FloorPlansResponse;

    if (
      !payload ||
      typeof payload.project_id !== "string" ||
      !Array.isArray(payload.floor_plans) ||
      typeof payload.version_number !== "number"
    ) {
      return { data: null, error: "Invalid floor plans response" };
    }

    return { data: payload, error: null };
  } catch {
    return { data: null, error: "Unable to connect to the Floor Plans API" };
  }
}

export async function saveFloorPlans(
  id: string,
  floorPlans: FloorPlan[],
): Promise<{
  data: SavedFloorPlansResponse | null;
  error: string | null;
}> {
  try {
    const response = await fetch(
      `/api/projects/${encodeURIComponent(id)}/floor-plans`,
      {
        method: "PUT",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ floor_plans: floorPlans }),
        cache: "no-store",
      },
    );

    if (!response.ok) {
      return {
        data: null,
        error: await readFloorPlanError(response, "Could not save floor plans"),
      };
    }

    const payload = (await response.json()) as SavedFloorPlansResponse;

    if (
      !payload ||
      typeof payload.project_id !== "string" ||
      typeof payload.design_id !== "string" ||
      typeof payload.version_id !== "string" ||
      typeof payload.version_number !== "number" ||
      !Array.isArray(payload.floor_plans)
    ) {
      return { data: null, error: "Invalid save floor plans response" };
    }

    return { data: payload, error: null };
  } catch {
    return { data: null, error: "Unable to connect to the Floor Plans API" };
  }
}