package docker

// ContainerConfig holds container creation configuration
type ContainerConfig struct {
	Image       string
	MemoryLimit int64
	CPULimit    int64
	PidsLimit   int64
}
