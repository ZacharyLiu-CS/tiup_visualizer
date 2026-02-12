from fastapi import APIRouter, HTTPException, Depends, Query
from fastapi.responses import StreamingResponse, PlainTextResponse
from typing import List, Dict, Optional
import subprocess
import os
import glob
from app.services.tiup_service import TiUPService
from app.models.cluster import ClusterInfo, ClusterDetail, HostInfo
from app.core.auth import get_current_user
from app.core.config import LOG_DIR

router = APIRouter()
tiup_service = TiUPService()


def _resolve_log_dir() -> str:
    """Resolve the backend log directory to an absolute path."""
    log_dir = LOG_DIR
    if not os.path.isabs(log_dir):
        log_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), log_dir)
    return log_dir


@router.get("/overview")
async def get_overview(current_user: str = Depends(get_current_user)):
    """Get clusters and hosts in a single call to avoid redundant tiup commands."""
    try:
        clusters = tiup_service.get_all_clusters()
        hosts = tiup_service.get_all_hosts()
        return {"clusters": clusters, "hosts": hosts}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/clusters", response_model=List[ClusterInfo])
async def get_clusters(current_user: str = Depends(get_current_user)):
    """Get all TiUP clusters"""
    try:
        return tiup_service.get_all_clusters()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/clusters/{cluster_name}", response_model=ClusterDetail)
async def get_cluster_detail(cluster_name: str, current_user: str = Depends(get_current_user)):
    """Get detailed information of a specific cluster"""
    try:
        return tiup_service.get_cluster_detail(cluster_name)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/hosts", response_model=Dict[str, HostInfo])
async def get_hosts(current_user: str = Depends(get_current_user)):
    """Get all physical hosts"""
    try:
        return tiup_service.get_all_hosts()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/hosts/{host_ip}/clusters", response_model=List[str])
async def get_host_clusters(host_ip: str, current_user: str = Depends(get_current_user)):
    """Get all clusters deployed on a specific host"""
    try:
        hosts = tiup_service.get_all_hosts()
        if host_ip in hosts:
            return hosts[host_ip].clusters
        return []
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/logs/{cluster_name}/{component_id}/{filename}")
async def get_log_file(cluster_name: str, component_id: str, filename: str, action: str = "view", token: Optional[str] = Query(default=None), current_user: str = Depends(get_current_user)):
    """Get a log file for a component. action=view returns text, action=download returns file download.
    Supports both Bearer token in header and token as query parameter (for direct browser access)."""
    try:
        log_path, component = tiup_service.get_log_file_path(cluster_name, component_id, filename)
        host = component.host

        # Try to read the log file. If the host is the local machine, read directly.
        # Otherwise, use SSH to read from remote host.
        local_hostname = subprocess.run(
            "hostname -I", shell=True, capture_output=True, text=True, timeout=5
        ).stdout.strip().split()

        if host in local_hostname or host == "127.0.0.1" or host == "localhost":
            # Local file
            if not os.path.exists(log_path):
                raise HTTPException(status_code=404, detail=f"Log file not found: {log_path}")
            if action == "download":
                def iterfile():
                    with open(log_path, "rb") as f:
                        while chunk := f.read(8192):
                            yield chunk
                return StreamingResponse(
                    iterfile(),
                    media_type="application/octet-stream",
                    headers={"Content-Disposition": f"attachment; filename={filename}"}
                )
            else:
                with open(log_path, "r", errors="replace") as f:
                    content = f.read()
                return PlainTextResponse(content)
        else:
            # Remote file via SSH
            detail = tiup_service.get_cluster_detail(cluster_name)
            deploy_user = detail.deploy_user
            ssh_cmd = f"ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 {deploy_user}@{host} 'cat {log_path}'"
            result = subprocess.run(
                ssh_cmd, shell=True, capture_output=True, text=True, timeout=30
            )
            if result.returncode != 0:
                raise HTTPException(
                    status_code=404,
                    detail=f"Failed to read log file from {host}:{log_path} - {result.stderr.strip()}"
                )
            content = result.stdout
            if action == "download":
                return StreamingResponse(
                    iter([content.encode("utf-8")]),
                    media_type="application/octet-stream",
                    headers={"Content-Disposition": f"attachment; filename={filename}"}
                )
            else:
                return PlainTextResponse(content)
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/server-logs")
async def list_server_logs(current_user: str = Depends(get_current_user)):
    """List all log files of the backend service."""
    try:
        log_dir = _resolve_log_dir()
        if not os.path.isdir(log_dir):
            return {"files": []}
        files = []
        for entry in sorted(os.listdir(log_dir)):
            filepath = os.path.join(log_dir, entry)
            if os.path.isfile(filepath) and entry.endswith(".log"):
                stat = os.stat(filepath)
                files.append({
                    "filename": entry,
                    "size": stat.st_size,
                })
        return {"files": files}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/server-logs/{filename}")
async def get_server_log(filename: str, action: str = "view", token: Optional[str] = Query(default=None), current_user: str = Depends(get_current_user)):
    """View or download a backend service log file.
    Supports both Bearer token in header and token as query parameter (for direct browser access)."""
    try:
        if ".." in filename or "/" in filename or "\\" in filename:
            raise HTTPException(status_code=400, detail="Invalid filename")
        log_dir = _resolve_log_dir()
        log_path = os.path.join(log_dir, filename)
        if not os.path.isfile(log_path):
            raise HTTPException(status_code=404, detail=f"Log file not found: {filename}")
        if action == "download":
            def iterfile():
                with open(log_path, "rb") as f:
                    while chunk := f.read(8192):
                        yield chunk
            return StreamingResponse(
                iterfile(),
                media_type="application/octet-stream",
                headers={"Content-Disposition": f"attachment; filename={filename}"}
            )
        else:
            with open(log_path, "r", errors="replace") as f:
                content = f.read()
            return PlainTextResponse(content)
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
