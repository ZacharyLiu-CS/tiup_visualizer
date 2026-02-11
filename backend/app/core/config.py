from pydantic_settings import BaseSettings
from typing import List


class Settings(BaseSettings):
    app_name: str = "TiUP Visualizer"
    debug: bool = True
    api_prefix: str = "/api/v1"

    # Root path for reverse proxy (e.g. "/app" if behind a sub-path proxy)
    root_path: str = ""

    # CORS settings - comma-separated origins or JSON list
    cors_origins: list = ["http://localhost:5173", "http://localhost:3000"]

    class Config:
        env_file = ".env"


settings = Settings()
