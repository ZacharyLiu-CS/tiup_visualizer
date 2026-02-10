from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    app_name: str = "TiUP Visualizer"
    debug: bool = True
    api_prefix: str = "/api/v1"
    
    # CORS settings
    cors_origins: list = ["http://localhost:5173", "http://localhost:3000"]
    
    class Config:
        env_file = ".env"


settings = Settings()
