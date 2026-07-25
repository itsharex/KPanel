package contract

import "time"

// Problem is the stable error envelope used by the public and local APIs.
type Problem struct {
	Type        string            `json:"type,omitempty"`
	Title       string            `json:"title"`
	Status      int               `json:"status"`
	Code        string            `json:"code"`
	Detail      string            `json:"detail,omitempty"`
	RequestID   string            `json:"requestId"`
	Retryable   bool              `json:"retryable"`
	FieldErrors map[string]string `json:"fieldErrors,omitempty"`
}

type PageResult[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type AgentHealth struct {
	Status          string    `json:"status"`
	Version         string    `json:"version"`
	ProtocolVersion string    `json:"protocolVersion"`
	ReadOnly        bool      `json:"readOnly"`
	Reasons         []string  `json:"reasons,omitempty"`
	CheckedAt       time.Time `json:"checkedAt"`
}

type Capability struct {
	ID      string   `json:"id"`
	Enabled bool     `json:"enabled"`
	Reason  string   `json:"reason,omitempty"`
	Methods []string `json:"methods,omitempty"`
}

type SystemSummary struct {
	Hostname      string         `json:"hostname"`
	OS            string         `json:"os"`
	Kernel        string         `json:"kernel"`
	Architecture  string         `json:"architecture"`
	UptimeSeconds uint64         `json:"uptimeSeconds"`
	Load          LoadSummary    `json:"load"`
	CPU           CPUSummary     `json:"cpu"`
	Memory        MemorySummary  `json:"memory"`
	Disks         []DiskSummary  `json:"disks"`
	Network       NetworkSummary `json:"network"`
	CollectedAt   time.Time      `json:"collectedAt"`
}

type LoadSummary struct {
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
}

type CPUSummary struct {
	Model        string  `json:"model,omitempty"`
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usagePercent"`
}

type MemorySummary struct {
	TotalBytes     uint64  `json:"totalBytes"`
	AvailableBytes uint64  `json:"availableBytes"`
	UsedBytes      uint64  `json:"usedBytes"`
	UsagePercent   float64 `json:"usagePercent"`
	SwapTotalBytes uint64  `json:"swapTotalBytes"`
	SwapUsedBytes  uint64  `json:"swapUsedBytes"`
}

type DiskSummary struct {
	Device       string  `json:"device"`
	MountPoint   string  `json:"mountPoint"`
	FileSystem   string  `json:"fileSystem"`
	TotalBytes   uint64  `json:"totalBytes"`
	UsedBytes    uint64  `json:"usedBytes"`
	UsagePercent float64 `json:"usagePercent"`
}

type NetworkSummary struct {
	ReceivedBytes  uint64 `json:"receivedBytes"`
	SentBytes      uint64 `json:"sentBytes"`
	TCPConnections int    `json:"tcpConnections"`
	UDPConnections int    `json:"udpConnections"`
}

type Origin string

const (
	OriginWeb        Origin = "web"
	OriginCLI        Origin = "cli"
	OriginDiscovered Origin = "discovered"
	OriginExternal   Origin = "external"
)

type Consistency string

const (
	ConsistencyInSync      Consistency = "in_sync"
	ConsistencyDrifted     Consistency = "drifted"
	ConsistencyAmbiguous   Consistency = "ambiguous"
	ConsistencyConflicted  Consistency = "conflicted"
	ConsistencyUnsupported Consistency = "unsupported"
	ConsistencyReadOnly    Consistency = "read_only"
)

type SiteKind string

const (
	SiteStatic       SiteKind = "static"
	SiteReverseProxy SiteKind = "reverse_proxy"
	SitePHP          SiteKind = "php"
	SiteWordPress    SiteKind = "wordpress"
	SiteRedirect     SiteKind = "redirect"
	SiteUnknown      SiteKind = "unknown"
)

type SiteSummary struct {
	ID              string      `json:"id"`
	PrimaryDomain   string      `json:"primaryDomain"`
	Domains         []string    `json:"domains"`
	Kind            SiteKind    `json:"kind"`
	Enabled         bool        `json:"enabled"`
	Health          string      `json:"health"`
	TLS             TLSStatus   `json:"tls"`
	Target          string      `json:"target,omitempty"`
	DocumentRoot    string      `json:"documentRoot,omitempty"`
	Origin          Origin      `json:"origin"`
	Consistency     Consistency `json:"consistency"`
	ResourceVersion string      `json:"resourceVersion"`
	AllowedActions  []string    `json:"allowedActions"`
	Artifacts       []Artifact  `json:"artifacts,omitempty"`
	Warnings        []string    `json:"warnings,omitempty"`
	ReconciledAt    time.Time   `json:"reconciledAt"`
}

type TLSStatus struct {
	Enabled   bool       `json:"enabled"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	Source    string     `json:"source,omitempty"`
}

type Artifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
	Hash string `json:"hash,omitempty"`
}

type DockerSummary struct {
	Available     bool      `json:"available"`
	ServerVersion string    `json:"serverVersion,omitempty"`
	Containers    int       `json:"containers"`
	Running       int       `json:"running"`
	Paused        int       `json:"paused"`
	Stopped       int       `json:"stopped"`
	Images        int       `json:"images"`
	CollectedAt   time.Time `json:"collectedAt"`
}

type ContainerSummary struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Image             string            `json:"image"`
	State             string            `json:"state"`
	Status            string            `json:"status"`
	Health            string            `json:"health,omitempty"`
	Ports             []PortBinding     `json:"ports"`
	Mounts            []Mount           `json:"mounts"`
	Networks          []string          `json:"networks"`
	ComposeProject    string            `json:"composeProject,omitempty"`
	Ownership         string            `json:"ownership"`
	OwnershipEvidence []string          `json:"ownershipEvidence,omitempty"`
	ResourceVersion   string            `json:"resourceVersion"`
	AllowedActions    []string          `json:"allowedActions"`
	Labels            map[string]string `json:"-"`
}

type PortBinding struct {
	PrivatePort uint16 `json:"privatePort"`
	PublicPort  uint16 `json:"publicPort,omitempty"`
	IP          string `json:"ip,omitempty"`
	Type        string `json:"type"`
}

type Mount struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"readOnly"`
}

type JobState string

const (
	JobQueued               JobState = "queued"
	JobRunning              JobState = "running"
	JobSucceeded            JobState = "succeeded"
	JobFailedRolledBack     JobState = "failed_rolled_back"
	JobFailedNeedsAttention JobState = "failed_needs_attention"
	JobInterrupted          JobState = "interrupted"
	JobCancelled            JobState = "cancelled"
)

type Job struct {
	ID          string     `json:"id"`
	Action      string     `json:"action"`
	Origin      Origin     `json:"origin"`
	State       JobState   `json:"state"`
	Progress    int        `json:"progress,omitempty"`
	Stage       string     `json:"stage,omitempty"`
	TargetKind  string     `json:"targetKind,omitempty"`
	TargetID    string     `json:"targetId,omitempty"`
	TargetLabel string     `json:"targetLabel,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
	Error       *Problem   `json:"error,omitempty"`
}

type AuditEvent struct {
	ID         string         `json:"id"`
	OccurredAt time.Time      `json:"occurredAt"`
	ActorType  string         `json:"actorType"`
	ActorID    string         `json:"actorId,omitempty"`
	SourceIP   string         `json:"sourceIp,omitempty"`
	Action     string         `json:"action"`
	TargetKind string         `json:"targetKind,omitempty"`
	TargetID   string         `json:"targetId,omitempty"`
	Result     string         `json:"result"`
	RequestID  string         `json:"requestId"`
	Change     map[string]any `json:"change,omitempty"`
}
