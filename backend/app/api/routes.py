from fastapi import APIRouter, HTTPException
from typing import List, Dict
from app.services.tiup_service import TiUPService
from app.models.cluster import ClusterInfo, ClusterDetail, HostInfo

router = APIRouter()
tiup_service = TiUPService()


@router.get("/clusters", response_model=List[ClusterInfo])
async def get_clusters():
    """Get all TiUP clusters"""
    try:
        return tiup_service.get_all_clusters()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/clusters/{cluster_name}", response_model=ClusterDetail)
async def get_cluster_detail(cluster_name: str):
    """Get detailed information of a specific cluster"""
    try:
        return tiup_service.get_cluster_detail(cluster_name)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/hosts", response_model=Dict[str, HostInfo])
async def get_hosts():
    """Get all physical hosts"""
    try:
        return tiup_service.get_all_hosts()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/hosts/{host_ip}/clusters", response_model=List[str])
async def get_host_clusters(host_ip: str):
    """Get all clusters deployed on a specific host"""
    try:
        hosts = tiup_service.get_all_hosts()
        if host_ip in hosts:
            return hosts[host_ip].clusters
        return []
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
