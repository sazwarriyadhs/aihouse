import os
from datetime import datetime, timezone

from fastapi import FastAPI

app = FastAPI(
    title="AI House Generator AI Service",
    version="0.1.0",
    description="AI-assisted house design service",
)


@app.get("/healthz")
@app.get("/api/v1/health")
async def health():
    return {
        "status": "ok",
        "service": "ai-house-generator-ai",
        "version": app.version,
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host="127.0.0.1",
        port=int(os.getenv("PORT", "8013")),
    )