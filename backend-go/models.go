package main

// ClusterInfo represents a cluster summary from `tiup cluster list`.
type ClusterInfo struct {
	Name       string `json:"name"`
	User       string `json:"user"`
	Version    string `json:"version"`
	Path       string `json:"path"`
	PrivateKey string `json:"private_key"`
	Status     string `json:"status"` // healthy, partial, unhealthy, unknown
}

// LogFileInfo represents a log file descriptor.
type LogFileInfo struct {
	Filename string `json:"filename"`
	Exists   bool   `json:"exists"`
}

// ComponentInfo represents a single component instance.
type ComponentInfo struct {
	ID        string        `json:"id"`
	Role      string        `json:"role"`
	Host      string        `json:"host"`
	Ports     string        `json:"ports"`
	OSArch    string        `json:"os_arch"`
	Status    string        `json:"status"`
	DataDir   string        `json:"data_dir"`
	DeployDir string        `json:"deploy_dir"`
	LogFiles  []LogFileInfo `json:"log_files"`
}

// ClusterDetail represents the full detail of a cluster from `tiup cluster display`.
type ClusterDetail struct {
	ClusterType    string          `json:"cluster_type"`
	ClusterName    string          `json:"cluster_name"`
	ClusterVersion string          `json:"cluster_version"`
	DeployUser     string          `json:"deploy_user"`
	SSHType        string          `json:"ssh_type"`
	DashboardURL   *string         `json:"dashboard_url"`
	GrafanaURL     *string         `json:"grafana_url"`
	Components     []ComponentInfo `json:"components"`
}

// HostInfo aggregates components by physical host.
type HostInfo struct {
	Host       string          `json:"host"`
	Components []ComponentInfo `json:"components"`
	Clusters   []string        `json:"clusters"`
}

// --- Auth models ---

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type UserInfoResponse struct {
	Username string `json:"username"`
}

type ErrorResponse struct {
	Detail string `json:"detail"`
}
