import { NextRequest } from "next/server";

const API_BASE_URL = (
  process.env.API_BASE_URL || "http://127.0.0.1:8097"
).replace(/\/+$/, "");

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

type RouteContext = {
  params: Promise<{ id: string }>;
};

async function forwardFloorPlanRequest(
  request: Request,
  id: string,
  method: "GET" | "PUT",
) {
  try {
    const body = method === "PUT" ? await request.text() : undefined;

    const upstream = await fetch(
      `${API_BASE_URL}/api/v1/projects/${encodeURIComponent(id)}/floor-plans`,
      {
        method,
        headers: {
          Accept: "application/json",
          ...(method === "PUT"
            ? { "Content-Type": "application/json" }
            : {}),
        },
        ...(body !== undefined ? { body } : {}),
        cache: "no-store",
      },
    );

    const responseBody = await upstream.arrayBuffer();

    return new Response(responseBody, {
      status: upstream.status,
      headers: {
        "Content-Type":
          upstream.headers.get("content-type") || "application/json",
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return Response.json(
      { error: "Unable to connect to the Projects API" },
      { status: 502 },
    );
  }
}

export async function GET(
  _request: NextRequest,
  context: RouteContext,
) {
  const { id } = await context.params;
  return forwardFloorPlanRequest(_request, id, "GET");
}

export async function PUT(
  request: NextRequest,
  context: RouteContext,
) {
  const { id } = await context.params;
  return forwardFloorPlanRequest(request, id, "PUT");
}