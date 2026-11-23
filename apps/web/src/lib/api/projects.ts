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
  process.env.API_BASE_URL || "http://127.0.0.1:8098"
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