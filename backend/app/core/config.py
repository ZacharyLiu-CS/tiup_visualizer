import os
import yaml
import logging
from logging.handlers import RotatingFileHandler
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


def _find_config_file() -> str:
    """Search for config.yaml in multiple locations."""
    candidates = [
        os.environ.get("TIUP_VISUALIZER_CONFIG", ""),
        os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "config.yaml"),
        "/etc/tiup-visualizer/config.yaml",
    ]
    for path in candidates:
        if path and os.path.isfile(path):
            return path
    return ""


def load_yaml_config() -> dict:
    """Load config.yaml and return as dict."""
    path = _find_config_file()
    if not path:
        return {}
    with open(path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f) or {}


_yaml_cfg = load_yaml_config()

# --- Auth config ---
_auth_cfg = _yaml_cfg.get("auth", {})
AUTH_USERNAME: str = _auth_cfg.get("username", "admin")
AUTH_PASSWORD: str = _auth_cfg.get("password", "easygraph")
AUTH_SECRET_KEY: str = _auth_cfg.get("secret_key", "tiup-visualizer-secret-key-change-me-in-production")
AUTH_TOKEN_EXPIRE_HOURS: int = _auth_cfg.get("token_expire_hours", 24)

# --- Logging config ---
_log_cfg = _yaml_cfg.get("logging", {})
LOG_DIR: str = _log_cfg.get("log_dir", "./logs")
LOG_LEVEL: str = _log_cfg.get("log_level", "INFO")
LOG_MAX_FILE_SIZE_MB: int = _log_cfg.get("max_file_size_mb", 10)
LOG_BACKUP_COUNT: int = _log_cfg.get("backup_count", 5)


def setup_logging():
    """Configure application logging based on config.yaml."""
    log_dir = LOG_DIR
    if not os.path.isabs(log_dir):
        log_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), log_dir)
    os.makedirs(log_dir, exist_ok=True)

    log_file = os.path.join(log_dir, "tiup-visualizer.log")
    level = getattr(logging, LOG_LEVEL.upper(), logging.INFO)

    formatter = logging.Formatter(
        "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )

    file_handler = RotatingFileHandler(
        log_file,
        maxBytes=LOG_MAX_FILE_SIZE_MB * 1024 * 1024,
        backupCount=LOG_BACKUP_COUNT,
        encoding="utf-8",
    )
    file_handler.setLevel(level)
    file_handler.setFormatter(formatter)

    console_handler = logging.StreamHandler()
    console_handler.setLevel(level)
    console_handler.setFormatter(formatter)

    root_logger = logging.getLogger()
    root_logger.setLevel(level)
    # Remove existing handlers to avoid duplicates
    root_logger.handlers.clear()
    root_logger.addHandler(file_handler)
    root_logger.addHandler(console_handler)

    logging.getLogger("uvicorn.access").handlers.clear()
    logging.getLogger("uvicorn.error").handlers.clear()
    logging.getLogger("uvicorn.access").addHandler(file_handler)
    logging.getLogger("uvicorn.error").addHandler(file_handler)

    return logging.getLogger("tiup_visualizer")
