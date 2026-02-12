from pydantic import BaseModel
from typing import List, Optional


class ClusterInfo(BaseModel):
    name: str
    user: str
    version: str
    path: str
    private_key: str
    status: str = "unknown"  # healthy, partial, unhealthy, unknown


class ComponentInfo(BaseModel):
    id: str
    role: str
    host: str
    ports: str
    os_arch: str
    status: str
    data_dir: str
    deploy_dir: str


class ClusterDetail(BaseModel):
    cluster_type: str
    cluster_name: str
    cluster_version: str
    deploy_user: str
    ssh_type: str
    dashboard_url: Optional[str] = None
    grafana_url: Optional[str] = None
    components: List[ComponentInfo]


class HostInfo(BaseModel):
    host: str
    components: List[ComponentInfo]
    clusters: List[str]
