import pytest
from app.services.tiup_service import TiUPService


def test_parse_cluster_list():
    """Test parsing tiup cluster list output"""
    service = TiUPService()
    
    output = """Name                        User  Version  Path                                                             PrivateKey
----                        ----  -------  ----                                                             ----------
eg3-cicd-proxy              root  v8.1.0   /root/.tiup/storage/cluster/clusters/eg3-cicd-proxy              /root/.tiup/storage/cluster/clusters/eg3-cicd-proxy/ssh/id_rsa
eg3_cicd_graphrag           root  v8.1.0   /root/.tiup/storage/cluster/clusters/eg3_cicd_graphrag           /root/.tiup/storage/cluster/clusters/eg3_cicd_graphrag/ssh/id_rsa
eg3_cicd_ldbc_rw            root  v8.1.0   /root/.tiup/storage/cluster/clusters/eg3_cicd_ldbc_rw            /root/.tiup/storage/cluster/clusters/eg3_cicd_ldbc_rw/ssh/id_rsa"""
    
    clusters = service.parse_cluster_list(output)
    
    assert len(clusters) == 3
    assert clusters[0].name == "eg3-cicd-proxy"
    assert clusters[0].user == "root"
    assert clusters[0].version == "v8.1.0"


def test_parse_cluster_display():
    """Test parsing tiup cluster display output"""
    service = TiUPService()
    
    output = """Cluster type:       tidb
Cluster name:       eg3_cicd_prop_ro
Cluster version:    v8.1.0
Deploy user:        root
SSH type:           builtin
Dashboard URL:      http://11.154.160.37:17379/dashboard
Grafana URL:        http://11.154.160.246:80
ID                    Role          Host            Ports        OS/Arch       Status   Data Dir                                     Deploy Dir
--                    ----          ----            -----        -------       ------   --------                                     ----------
11.154.160.246:16160  tikv          11.154.160.246  16160/16180  linux/x86_64  Up       /data1/cicd-data-prop-ro/tikv-16160          /data1/cicd-deploy-prop-ro/tikv-16160
11.154.160.246:17379  pd            11.154.160.246  17379/17380  linux/x86_64  Up       /data3/cicd-data-prop-ro/pd-17379            /data3/cicd-deploy-prop-ro/pd-17379
Total nodes: 2"""
    
    detail = service.parse_cluster_display(output, "eg3_cicd_prop_ro")
    
    assert detail.cluster_name == "eg3_cicd_prop_ro"
    assert detail.cluster_type == "tidb"
    assert detail.cluster_version == "v8.1.0"
    assert detail.dashboard_url == "http://11.154.160.37:17379/dashboard"
    assert len(detail.components) == 2
    assert detail.components[0].role == "tikv"
    assert detail.components[0].host == "11.154.160.246"
    assert detail.components[1].role == "pd"
