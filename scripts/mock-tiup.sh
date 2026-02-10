#!/bin/bash

# Mock TiUP commands for testing without real TiUP installation

COMMAND=$1
SUBCOMMAND=$2
ACTION=$3

if [ "$COMMAND" = "cluster" ] && [ "$SUBCOMMAND" = "list" ]; then
    cat << 'EOF'
Name                        User  Version  Path                                                             PrivateKey
----                        ----  -------  ----                                                             ----------
eg3-cicd-proxy              root  v8.1.0   /root/.tiup/storage/cluster/clusters/eg3-cicd-proxy              /root/.tiup/storage/cluster/clusters/eg3-cicd-proxy/ssh/id_rsa
eg3_cicd_graphrag           root  v8.1.0   /root/.tiup/storage/cluster/clusters/eg3_cicd_graphrag           /root/.tiup/storage/cluster/clusters/eg3_cicd_graphrag/ssh/id_rsa
eg3_cicd_ldbc_rw            root  v8.1.0   /root/.tiup/storage/cluster/clusters/eg3_cicd_ldbc_rw            /root/.tiup/storage/cluster/clusters/eg3_cicd_ldbc_rw/ssh/id_rsa
EOF
elif [ "$COMMAND" = "cluster" ] && [ "$SUBCOMMAND" = "display" ]; then
    CLUSTER_NAME=$ACTION
    cat << EOF
Cluster type:       tidb
Cluster name:       $CLUSTER_NAME
Cluster version:    v8.1.0
Deploy user:        root
SSH type:           builtin
Dashboard URL:      http://11.154.160.37:17379/dashboard
Grafana URL:        http://11.154.160.246:80
ID                    Role          Host            Ports        OS/Arch       Status   Data Dir                                     Deploy Dir
--                    ----          ----            -----        -------       ------   --------                                     ----------
11.154.160.246:49193  alertmanager  11.154.160.246  49193/49194  linux/x86_64  Down     /data3/cicd-data-prop-ro/alertmanager-49193  /data3/cicd-deploy-prop-ro/alertmanager-49193
11.154.160.246:80     grafana       11.154.160.246  80           linux/x86_64  Down     -                                            /data3/cicd-deploy-prop-ro/grafana-80
11.154.160.246:17379  pd            11.154.160.246  17379/17380  linux/x86_64  Up       /data3/cicd-data-prop-ro/pd-17379            /data3/cicd-deploy-prop-ro/pd-17379
11.154.160.28:17379   pd            11.154.160.28   17379/17380  linux/x86_64  Up       /data3/cicd-data-prop-ro/pd-17379            /data3/cicd-deploy-prop-ro/pd-17379
11.154.160.37:17379   pd            11.154.160.37   17379/17380  linux/x86_64  Up|L|UI  /data3/cicd-data-prop-ro/pd-17379            /data3/cicd-deploy-prop-ro/pd-17379
11.154.160.246:49190  prometheus    11.154.160.246  49190/12020  linux/x86_64  Down     /data3/cicd-data-prop-ro/prometheus-49190    /data3/cicd-deploy-prop-ro/prometheus-49190
11.154.160.246:16160  tikv          11.154.160.246  16160/16180  linux/x86_64  Up       /data1/cicd-data-prop-ro/tikv-16160          /data1/cicd-deploy-prop-ro/tikv-16160
11.154.160.246:16161  tikv          11.154.160.246  16161/16181  linux/x86_64  Up       /data2/cicd-data-prop-ro/tikv-16161          /data2/cicd-deploy-prop-ro/tikv-16161
11.154.160.246:16162  tikv          11.154.160.246  16162/16182  linux/x86_64  Up       /data3/cicd-data-prop-ro/tikv-16162          /data3/cicd-deploy-prop-ro/tikv-16162
11.154.160.246:16163  tikv          11.154.160.246  16163/16183  linux/x86_64  Up       /data4/cicd-data-prop-ro/tikv-16163          /data4/cicd-deploy-prop-ro/tikv-16163
11.154.160.28:16160   tikv          11.154.160.28   16160/16180  linux/x86_64  Up       /data1/cicd-data-prop-ro/tikv-16160          /data1/cicd-deploy-prop-ro/tikv-16160
11.154.160.28:16161   tikv          11.154.160.28   16161/16181  linux/x86_64  Up       /data2/cicd-data-prop-ro/tikv-16161          /data2/cicd-deploy-prop-ro/tikv-16161
11.154.160.28:16162   tikv          11.154.160.28   16162/16182  linux/x86_64  Up       /data3/cicd-data-prop-ro/tikv-16162          /data3/cicd-deploy-prop-ro/tikv-16162
11.154.160.28:16163   tikv          11.154.160.28   16163/16183  linux/x86_64  N/A      /data4/cicd-data-prop-ro/tikv-16163          /data4/cicd-deploy-prop-ro/tikv-16163
11.154.160.37:16160   tikv          11.154.160.37   16160/16180  linux/x86_64  Up       /data1/cicd-data-prop-ro/tikv-16160          /data1/cicd-deploy-prop-ro/tikv-16160
11.154.160.37:16161   tikv          11.154.160.37   16161/16181  linux/x86_64  Up       /data2/cicd-data-prop-ro/tikv-16161          /data2/cicd-deploy-prop-ro/tikv-16161
11.154.160.37:16162   tikv          11.154.160.37   16162/16182  linux/x86_64  Up       /data3/cicd-data-prop-ro/tikv-16162          /data3/cicd-deploy-prop-ro/tikv-16162
11.154.160.37:16163   tikv          11.154.160.37   16163/16183  linux/x86_64  Up       /data4/cicd-data-prop-ro/tikv-16163          /data4/cicd-deploy-prop-ro/tikv-16163
Total nodes: 18
EOF
else
    echo "Mock TiUP - Unknown command: $*"
    exit 1
fi
