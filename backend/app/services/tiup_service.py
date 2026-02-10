import subprocess
import re
from typing import List, Dict
from app.models.cluster import ClusterInfo, ClusterDetail, ComponentInfo, HostInfo


class TiUPService:
    @staticmethod
    def execute_command(command: str) -> str:
        """Execute tiup command and return output"""
        try:
            result = subprocess.run(
                command,
                shell=True,
                capture_output=True,
                text=True,
                timeout=30
            )
            return result.stdout
        except subprocess.TimeoutExpired:
            raise Exception("Command execution timeout")
        except Exception as e:
            raise Exception(f"Command execution failed: {str(e)}")

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
                    components.append(ComponentInfo(
                        id=parts[0],
                        role=parts[1],
                        host=parts[2],
                        ports=parts[3],
                        os_arch=parts[4],
                        status=parts[5],
                        data_dir=parts[6],
                        deploy_dir=parts[7]
                    ))
        
        return ClusterDetail(**cluster_info, components=components)

    def get_all_clusters(self) -> List[ClusterInfo]:
        """Get all tiup clusters"""
        output = self.execute_command("tiup cluster list")
        return self.parse_cluster_list(output)

    def get_cluster_detail(self, cluster_name: str) -> ClusterDetail:
        """Get detailed information of a specific cluster"""
        output = self.execute_command(f"tiup cluster display {cluster_name}")
        return self.parse_cluster_display(output, cluster_name)

    def get_all_hosts(self) -> Dict[str, HostInfo]:
        """Get all physical hosts and their components"""
        hosts_map: Dict[str, HostInfo] = {}
        clusters = self.get_all_clusters()
        
        for cluster in clusters:
            try:
                detail = self.get_cluster_detail(cluster.name)
                for component in detail.components:
                    if component.host not in hosts_map:
                        hosts_map[component.host] = HostInfo(
                            host=component.host,
                            components=[],
                            clusters=[]
                        )
                    
                    hosts_map[component.host].components.append(component)
                    if cluster.name not in hosts_map[component.host].clusters:
                        hosts_map[component.host].clusters.append(cluster.name)
            except Exception as e:
                print(f"Error getting cluster {cluster.name} detail: {str(e)}")
                continue
        
        return hosts_map
