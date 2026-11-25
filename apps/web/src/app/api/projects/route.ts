const API_BASE_URL = (
  process.env.API_BASE_URL || "http://127.0.0.1:8097"
).replace(/\/+$/, "");

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

async function forwardRequest(
  request: Request,
  method: "POST" | "PUT",
  endpoint: string,
) {
  try {
    const body = await request.text();

    const upstream = await fetch(`${API_BASE_URL}${endpoint}`, {
      method,
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body,
      cache: "no-store",
    });

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

export async function POST(request: Request) {
  return forwardRequest(request, "POST", "/api/v1/projects");
}
