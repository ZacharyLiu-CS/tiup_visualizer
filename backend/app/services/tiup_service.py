import subprocess
import re
import os
import time
import logging
from typing import List, Dict, Optional
from app.models.cluster import ClusterInfo, ClusterDetail, ComponentInfo, HostInfo, LogFileInfo

logger = logging.getLogger("tiup_visualizer")

# Mapping of component role to its log file names
ROLE_LOG_FILES = {
    'tikv': ['tikv.log', 'tikv_stderr.log'],
    'pd': ['pd.log', 'pd_stderr.log'],
    'tidb': ['tidb.log', 'tidb_stderr.log'],
    'grafana': ['grafana.log'],
    'prometheus': ['prometheus.log'],
    'alertmanager': ['alertmanager.log'],
}

# Default cache TTL in seconds
CACHE_TTL = 30


class _Cache:
    """Simple TTL cache for tiup command results."""

    def __init__(self, ttl: int = CACHE_TTL):
        self._ttl = ttl
        self._store: Dict[str, tuple] = {}  # key -> (value, timestamp)

    def get(self, key: str):
        entry = self._store.get(key)
        if entry is None:
            return None
        value, ts = entry
        if time.time() - ts > self._ttl:
            del self._store[key]
            return None
        return value

    def set(self, key: str, value):
        self._store[key] = (value, time.time())

    def clear(self):
        self._store.clear()


class TiUPService:
    def __init__(self):
        self._cache = _Cache()

    @staticmethod
    def execute_command(command: str, timeout: int = 30) -> str:
        """Execute tiup command and return output"""
        try:
            result = subprocess.run(
                command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            return result.stdout
        except subprocess.TimeoutExpired:
            raise Exception("Command execution timeout")
        except Exception as e:
            raise Exception(f"Command execution failed: {str(e)}")

    @staticmethod
    def get_log_files_for_role(role: str) -> List[str]:
        """Get expected log file names for a component role"""
        return ROLE_LOG_FILES.get(role.lower(), [f'{role.lower()}.log'])

    @staticmethod
    def parse_cluster_list(output: str) -> List[ClusterInfo]:
        """Parse tiup cluster list output"""
        clusters = []
        lines = output.strip().split('\n')
        
        # Skip header lines (Name, ----)
        data_lines = [line for line in lines if line and not line.startswith('Name') and not line.startswith('----')]
        
        for line in data_lines:
            parts = re.split(r'\s{2,}', line.strip())
            if len(parts) >= 5:
                clusters.append(ClusterInfo(
                    name=parts[0],
                    user=parts[1],
                    version=parts[2],
                    path=parts[3],
                    private_key=parts[4]
                ))
        
        return clusters

    @staticmethod
    def parse_cluster_display(output: str, cluster_name: str) -> ClusterDetail:
        """Parse tiup cluster display output"""
        lines = output.strip().split('\n')
        
        cluster_info = {
            'cluster_type': '',
            'cluster_name': cluster_name,
            'cluster_version': '',
            'deploy_user': '',
            'ssh_type': '',
            'dashboard_url': None,
            'grafana_url': None
        }
        
        components = []
        component_section = False
        
        for line in lines:
            line = line.strip()
            
            if not component_section:
                if line.startswith('Cluster type:'):
                    cluster_info['cluster_type'] = line.split(':', 1)[1].strip()
                elif line.startswith('Cluster version:'):
                    cluster_info['cluster_version'] = line.split(':', 1)[1].strip()
                elif line.startswith('Deploy user:'):
                    cluster_info['deploy_user'] = line.split(':', 1)[1].strip()
                elif line.startswith('SSH type:'):
                    cluster_info['ssh_type'] = line.split(':', 1)[1].strip()
                elif line.startswith('Dashboard URL:'):
                    cluster_info['dashboard_url'] = line.split(':', 1)[1].strip()
                elif line.startswith('Grafana URL:'):
                    cluster_info['grafana_url'] = line.split(':', 1)[1].strip()
                elif line.startswith('ID') and 'Role' in line:
                    component_section = True
                    continue
            else:
                if line.startswith('--') or line.startswith('Total nodes:') or not line:
                    continue
                
                parts = re.split(r'\s{2,}', line)
                if len(parts) >= 8:
                    role = parts[1]
                    log_filenames = TiUPService.get_log_files_for_role(role)
                    log_files = [LogFileInfo(filename=f, exists=True) for f in log_filenames]
                    components.append(ComponentInfo(
                        id=parts[0],
                        role=parts[1],
                        host=parts[2],
                        ports=parts[3],
                        os_arch=parts[4],
                        status=parts[5],
                        data_dir=parts[6],
                        deploy_dir=parts[7],
                        log_files=log_files
                    ))
        
        return ClusterDetail(**cluster_info, components=components)

    def get_cluster_detail(self, cluster_name: str) -> ClusterDetail:
        """Get detailed information of a specific cluster (cached)."""
        cache_key = f"detail:{cluster_name}"
        cached = self._cache.get(cache_key)
        if cached is not None:
            return cached
        output = self.execute_command(f"tiup cluster display {cluster_name}")
        detail = self.parse_cluster_display(output, cluster_name)
        self._cache.set(cache_key, detail)
        return detail

    def _get_cluster_list(self) -> List[ClusterInfo]:
        """Get raw cluster list from tiup (cached)."""
        cache_key = "cluster_list"
        cached = self._cache.get(cache_key)
        if cached is not None:
            return cached
        output = self.execute_command("tiup cluster list")
        clusters = self.parse_cluster_list(output)
        self._cache.set(cache_key, clusters)
        return clusters

    def _get_all_details(self) -> Dict[str, ClusterDetail]:
        """Fetch details for all clusters, reusing cache. Returns {name: detail}."""
        clusters = self._get_cluster_list()
        details = {}
        for cluster in clusters:
            try:
                details[cluster.name] = self.get_cluster_detail(cluster.name)
            except Exception as e:
                logger.warning(f"Error getting cluster {cluster.name} detail: {e}")
        return details

    def get_all_clusters(self) -> List[ClusterInfo]:
        """Get all tiup clusters with health status."""
        clusters = self._get_cluster_list()
        # Make copies so we don't mutate the cached list
        result = []
        details = self._get_all_details()
        for cluster in clusters:
            c = ClusterInfo(
                name=cluster.name,
                user=cluster.user,
                version=cluster.version,
                path=cluster.path,
                private_key=cluster.private_key,
            )
            detail = details.get(cluster.name)
            if detail:
                statuses = [comp.status for comp in detail.components]
                if not statuses:
                    c.status = "unknown"
                else:
                    has_up = any("Up" in s for s in statuses)
                    all_up = all("Up" in s for s in statuses)
                    if all_up:
                        c.status = "healthy"
                    elif has_up:
                        c.status = "partial"
                    else:
                        c.status = "unhealthy"
            else:
                c.status = "unknown"
            result.append(c)
        return result

    def get_all_hosts(self) -> Dict[str, HostInfo]:
        """Get all physical hosts and their components."""
        hosts_map: Dict[str, HostInfo] = {}
        details = self._get_all_details()

        for cluster_name, detail in details.items():
            for component in detail.components:
                if component.host not in hosts_map:
                    hosts_map[component.host] = HostInfo(
                        host=component.host,
                        components=[],
                        clusters=[]
                    )
                hosts_map[component.host].components.append(component)
                if cluster_name not in hosts_map[component.host].clusters:
                    hosts_map[component.host].clusters.append(cluster_name)

        return hosts_map

    def get_log_file_path(self, cluster_name: str, component_id: str, filename: str) -> str:
        """Get the local file path for a component's log file."""
        detail = self.get_cluster_detail(cluster_name)
        component = None
        for comp in detail.components:
            if comp.id == component_id:
                component = comp
                break
        if not component:
            raise Exception(f"Component {component_id} not found in cluster {cluster_name}")

        allowed_files = self.get_log_files_for_role(component.role)
        if filename not in allowed_files:
            raise Exception(f"Log file {filename} not allowed for role {component.role}")

        log_path = os.path.join(component.deploy_dir, "log", filename)
        return log_path, component
