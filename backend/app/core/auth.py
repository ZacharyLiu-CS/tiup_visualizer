import jwt
import logging
from datetime import datetime, timedelta, timezone
from typing import Optional, Tuple
from fastapi import Depends, HTTPException, status, Query
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from pydantic import BaseModel
from app.core.config import AUTH_USERNAME, AUTH_PASSWORD, AUTH_SECRET_KEY, AUTH_TOKEN_EXPIRE_HOURS

logger = logging.getLogger("tiup_visualizer")

ALGORITHM = "HS256"

security = HTTPBearer(auto_error=False)


class LoginRequest(BaseModel):
    username: str
    password: str


class TokenResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    expires_in: int


class UserInfo(BaseModel):
    username: str


def create_access_token(username: str) -> Tuple[str, int]:
    """Create a JWT access token. Returns (token, expires_in_seconds)."""
    expires_delta = timedelta(hours=AUTH_TOKEN_EXPIRE_HOURS)
    expire = datetime.now(timezone.utc) + expires_delta
    payload = {
        "sub": username,
        "exp": expire,
        "iat": datetime.now(timezone.utc),
    }
    token = jwt.encode(payload, AUTH_SECRET_KEY, algorithm=ALGORITHM)
    return token, int(expires_delta.total_seconds())


def verify_token(token: str) -> Optional[str]:
    """Verify a JWT token and return the username, or None if invalid."""
    try:
        payload = jwt.decode(token, AUTH_SECRET_KEY, algorithms=[ALGORITHM])
        username: str = payload.get("sub")
        if username is None:
            return None
        return username
    except jwt.ExpiredSignatureError:
        logger.warning("Token expired")
        return None
    except jwt.InvalidTokenError as e:
        logger.warning(f"Invalid token: {e}")
        return None


def authenticate_user(username: str, password: str) -> bool:
    """Authenticate user against configured credentials."""
    return username == AUTH_USERNAME and password == AUTH_PASSWORD


async def get_current_user(
    credentials: Optional[HTTPAuthorizationCredentials] = Depends(security),
    token: Optional[str] = Query(default=None),
) -> str:
    """FastAPI dependency that extracts and validates the JWT token.
    Supports both Authorization header and token query parameter."""
    # Try header first
    if credentials is not None:
        username = verify_token(credentials.credentials)
        if username:
            return username
    # Fallback to query parameter (for direct URL access like log downloads)
    if token:
        username = verify_token(token)
        if username:
            return username
    raise HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Not authenticated",
        headers={"WWW-Authenticate": "Bearer"},
    )


def verify_ws_token(token: Optional[str]) -> Optional[str]:
    """Verify token for WebSocket connections (passed as query param)."""
    if not token:
        return None
    return verify_token(token)
