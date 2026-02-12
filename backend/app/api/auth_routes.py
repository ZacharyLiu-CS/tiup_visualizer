import logging
from fastapi import APIRouter, HTTPException, status
from app.core.auth import (
    LoginRequest, TokenResponse, UserInfo,
    authenticate_user, create_access_token, verify_token,
)

logger = logging.getLogger("tiup_visualizer")

router = APIRouter()


@router.post("/auth/login", response_model=TokenResponse)
async def login(request: LoginRequest):
    """Authenticate user and return JWT token."""
    if not authenticate_user(request.username, request.password):
        logger.warning(f"Failed login attempt for user: {request.username}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid username or password",
        )
    token, expires_in = create_access_token(request.username)
    logger.info(f"User '{request.username}' logged in successfully")
    return TokenResponse(access_token=token, expires_in=expires_in)


@router.get("/auth/verify", response_model=UserInfo)
async def verify(token: str):
    """Verify if a token is still valid."""
    username = verify_token(token)
    if username is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token",
        )
    return UserInfo(username=username)
