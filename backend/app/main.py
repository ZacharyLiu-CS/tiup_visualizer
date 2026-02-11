from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
from app.core.config import settings
from app.api.routes import router
import os

app = FastAPI(
    title=settings.app_name,
    debug=settings.debug,
    root_path=settings.root_path,
)

# Trusted proxy support: correctly handle X-Forwarded-For, X-Forwarded-Proto
# When behind Nginx, uvicorn should be started with --proxy-headers (default on)
# and --forwarded-allow-ips to trust the proxy

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include API routes
app.include_router(router, prefix=settings.api_prefix)


@app.get("/health")
async def health_check():
    return {"status": "healthy"}


# Serve static files (for production build)
static_dir = os.path.join(os.path.dirname(os.path.dirname(__file__)), "static")
if os.path.exists(static_dir):
    app.mount("/assets", StaticFiles(directory=os.path.join(static_dir, "assets")), name="assets")
    
    @app.get("/")
    async def serve_spa():
        return FileResponse(os.path.join(static_dir, "index.html"))
else:
    @app.get("/")
    async def root():
        return {"message": "TiUP Visualizer API - Frontend not built yet"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host="0.0.0.0", port=8000, reload=True)
