package docker

import "time"

// Container labels applied to every lab container. They let the backend find
// (and clean up) its own containers even after a crash wiped in-memory state.
const (
	LabelSession = "devopslab.session"
	LabelUser    = "devopslab.user"
	LabelTask    = "devopslab.task"
)

// LabHostname is the hostname students see inside a lab container.
const LabHostname = "lab"

// BaseCapabilities is the capability set every lab container gets after
// CapDrop ALL. It is enough for a root user to manage files, users, services
// and to bind privileged ports inside the container, nothing more.
var BaseCapabilities = []string{
	"CHOWN",            // chown files (useradd, chown)
	"DAC_OVERRIDE",     // root may read/write any file
	"FOWNER",           // chmod files it does not own
	"FSETID",           // keep setgid bits on chmod (shared dirs)
	"SETUID",           // drop to service users (nginx workers, cron jobs)
	"SETGID",           // same, for groups
	"KILL",             // signal processes owned by other users
	"NET_BIND_SERVICE", // nginx listens on :80
	"AUDIT_WRITE",      // PAM-based tools (cron, su) write audit records
}

// ExtraCapabilityAllowlist lists the capabilities a task.yaml may request on
// top of BaseCapabilities. Anything else is rejected: a task file must never be
// able to grant e.g. SYS_ADMIN. Capabilities are only effective inside the
// container's own namespaces (NET_ADMIN only touches its private netns).
var ExtraCapabilityAllowlist = map[string]bool{
	"NET_ADMIN": true, // iptables/ufw inside the container's network namespace
	"NET_RAW":   true, // raw sockets (ping, legacy iptables)
}

// ContainerConfig holds container creation configuration.
type ContainerConfig struct {
	Image       string
	MemoryLimit int64 // bytes; 0 = unlimited
	CPULimit    int64 // nano CPUs; 0 = unlimited
	PidsLimit   int64 // 0 = unlimited

	// Labels identify the owner session; see LabelSession/LabelUser/LabelTask.
	Labels map[string]string

	// ExtraCapabilities are added on top of BaseCapabilities. They must be in
	// ExtraCapabilityAllowlist (enforced by BuildHostConfig).
	ExtraCapabilities []string

	// Network enables the default bridge network. When false the container
	// only has a private loopback interface (NetworkMode "none").
	Network bool
}

// LabContainer is a lab container found on the Docker host by its labels.
type LabContainer struct {
	ID        string
	SessionID string
	UserID    string
	TaskID    string
	State     string // created, running, exited, ...
	Created   time.Time
}
