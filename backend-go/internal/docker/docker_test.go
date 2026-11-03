package docker

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/moby/moby/api/types/container"
)

func TestBuildHostConfigDefaults(t *testing.T) {
	hc, err := BuildHostConfig(&ContainerConfig{
		Image:       "devopslab-lab:latest",
		MemoryLimit: 512 << 20,
		CPULimit:    500_000_000,
		PidsLimit:   256,
	})
	if err != nil {
		t.Fatal(err)
	}

	if hc.Privileged {
		t.Error("container must not be privileged")
	}
	if len(hc.CapDrop) != 1 || hc.CapDrop[0] != "ALL" {
		t.Errorf("CapDrop = %v, want [ALL]", hc.CapDrop)
	}
	if strings.Join(hc.CapAdd, ",") != strings.Join(BaseCapabilities, ",") {
		t.Errorf("CapAdd = %v, want base set %v", hc.CapAdd, BaseCapabilities)
	}
	for _, c := range hc.CapAdd {
		if c == "SYS_ADMIN" || c == "NET_ADMIN" || c == "NET_RAW" || c == "MKNOD" || c == "SYS_PTRACE" {
			t.Errorf("unexpected capability %s in base set", c)
		}
	}
	if hc.NetworkMode != "none" {
		t.Errorf("NetworkMode = %q, want none", hc.NetworkMode)
	}
	if len(hc.SecurityOpt) != 1 || hc.SecurityOpt[0] != "no-new-privileges:true" {
		t.Errorf("SecurityOpt = %v", hc.SecurityOpt)
	}
	if hc.Memory != 512<<20 || hc.MemorySwap != 512<<20 {
		t.Errorf("Memory/MemorySwap = %d/%d", hc.Memory, hc.MemorySwap)
	}
	if hc.NanoCPUs != 500_000_000 {
		t.Errorf("NanoCPUs = %d", hc.NanoCPUs)
	}
	if hc.PidsLimit == nil || *hc.PidsLimit != 256 {
		t.Errorf("PidsLimit = %v", hc.PidsLimit)
	}
	if len(hc.Binds) != 0 || len(hc.Mounts) != 0 {
		t.Error("lab containers must not have host mounts")
	}
	if _, ok := hc.Tmpfs["/tmp"]; !ok {
		t.Error("expected tmpfs on /tmp")
	}
	if hc.Init == nil || !*hc.Init {
		t.Error("expected an init process")
	}
	if hc.ReadonlyRootfs {
		t.Error("rootfs must stay writable for students")
	}
	var nofile *container.Ulimit
	for _, u := range hc.Ulimits {
		if u.Name == "nofile" {
			nofile = u
		}
	}
	if nofile == nil || nofile.Soft <= 0 || nofile.Hard < nofile.Soft {
		t.Errorf("nofile ulimit = %+v", nofile)
	}
	if hc.LogConfig.Type != "none" {
		t.Errorf("LogConfig.Type = %q, want none", hc.LogConfig.Type)
	}
}

func TestBuildHostConfigNetworkAndExtraCaps(t *testing.T) {
	hc, err := BuildHostConfig(&ContainerConfig{Network: true, ExtraCapabilities: []string{"net_admin", "CAP_NET_ADMIN", "NET_ADMIN"}})
	if err != nil {
		t.Fatal(err)
	}
	if hc.NetworkMode != "bridge" {
		t.Errorf("NetworkMode = %q, want bridge", hc.NetworkMode)
	}
	got := strings.Join(hc.CapAdd, ",")
	if !strings.HasSuffix(got, ",NET_ADMIN") || strings.Count(got, "NET_ADMIN") != 1 {
		t.Errorf("CapAdd = %s, want base + NET_ADMIN (deduplicated)", got)
	}
	if hc.Runtime != "" || hc.StorageOpt != nil {
		t.Errorf("runtime/storage must be unset by default: %q %v", hc.Runtime, hc.StorageOpt)
	}
	if hc.PidsLimit != nil {
		t.Error("PidsLimit 0 should mean unset")
	}
	if hc.MemorySwap != 0 {
		t.Error("MemorySwap must be 0 when no memory limit is set")
	}
}

func TestBuildCapAddRejectsDangerousCaps(t *testing.T) {
	for _, c := range []string{"SYS_ADMIN", "ALL", "SYS_MODULE", "SYS_PTRACE", "DAC_READ_SEARCH", "NET_RAW", "CAP_NET_RAW", "bogus"} {
		if _, err := BuildCapAdd([]string{c}); err == nil {
			t.Errorf("BuildCapAdd(%s) should fail", c)
		}
		if _, err := BuildHostConfig(&ContainerConfig{ExtraCapabilities: []string{c}}); err == nil {
			t.Errorf("BuildHostConfig with %s should fail", c)
		}
	}
}

func TestBuildHostConfigRuntimeAndStorage(t *testing.T) {
	hc, err := BuildHostConfig(&ContainerConfig{Runtime: "runsc", StorageSize: "2G"})
	if err != nil {
		t.Fatal(err)
	}
	if hc.Runtime != "runsc" {
		t.Errorf("Runtime = %q, want runsc", hc.Runtime)
	}
	if hc.StorageOpt["size"] != "2G" || len(hc.StorageOpt) != 1 {
		t.Errorf("StorageOpt = %v, want size=2G", hc.StorageOpt)
	}
}

func TestIsStorageOptUnsupported(t *testing.T) {
	yes := []string{
		"Error response from daemon: --storage-opt is supported only for overlay over xfs with 'pquota' mount option",
		"Error response from daemon: Storage Opt size not supported",
		"storage option size is not supported",
	}
	for _, m := range yes {
		if !IsStorageOptUnsupported(errors.New(m)) {
			t.Errorf("not detected: %q", m)
		}
	}
	if IsStorageOptUnsupported(errors.New("No such image: x")) || IsStorageOptUnsupported(nil) {
		t.Error("false positive")
	}
}

func TestBuildContainerConfig(t *testing.T) {
	labels := map[string]string{LabelSession: "abc", LabelUser: "7", LabelTask: "nginx-health"}
	cfg := BuildContainerConfig(&ContainerConfig{Image: "img", Labels: labels})
	if cfg.Hostname != LabHostname {
		t.Errorf("Hostname = %q", cfg.Hostname)
	}
	if cfg.Labels[LabelSession] != "abc" || cfg.Labels[LabelUser] != "7" || cfg.Labels[LabelTask] != "nginx-health" {
		t.Errorf("Labels = %v", cfg.Labels)
	}
	labels[LabelSession] = "mutated"
	if cfg.Labels[LabelSession] != "abc" {
		t.Error("labels must be copied")
	}
	if cfg.User != "" && cfg.User != "root" {
		t.Errorf("User = %q", cfg.User)
	}
}

func TestResolveHost(t *testing.T) {
	cases := []struct{ arg, env, want string }{
		{"/var/run/docker.sock", "", "unix:///var/run/docker.sock"},
		{"/var/run/docker.sock", "tcp://proxy:2375", ""}, // keep DOCKER_HOST from FromEnv
		{"tcp://proxy:2375", "", "tcp://proxy:2375"},
		{"unix:///custom.sock", "tcp://other:2375", "unix:///custom.sock"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := resolveHost(c.arg, c.env); got != c.want {
			t.Errorf("resolveHost(%q, %q) = %q, want %q", c.arg, c.env, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("short", 2048); got != "short" {
		t.Errorf("got %q", got)
	}
	long := strings.Repeat("a", 5000)
	got := Truncate(long, 2048)
	if len(got) > 2048 {
		t.Errorf("len = %d, want <= 2048", len(got))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Error("missing truncation marker")
	}
	// Never split a multi-byte rune.
	multi := strings.Repeat("ш", 3000)
	got = Truncate(multi, 2048)
	if !utf8.ValidString(got) || len(got) > 2048 {
		t.Errorf("invalid truncation: valid=%v len=%d", utf8.ValidString(got), len(got))
	}
	if Truncate("abc", 0) != "" {
		t.Error("max 0 should give empty string")
	}
}

func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{max: 10}
	n, err := b.Write([]byte("0123456789abcdef"))
	if err != nil || n != 16 {
		t.Fatalf("Write = %d, %v; must report full length to keep draining", n, err)
	}
	_, _ = b.Write([]byte("more"))
	if b.String() != "0123456789" {
		t.Errorf("buffer = %q", b.String())
	}
}

func TestWrapWithTimeout(t *testing.T) {
	argv := wrapWithTimeout("echo hi", 10)
	got := strings.Join(argv, " ")
	want := "/usr/bin/timeout -k 2 10 /usr/bin/env -i PATH=" + checkPath +
		" HOME=/root LANG=C.UTF-8 TERM=dumb /bin/bash --noprofile --norc -c echo hi"
	if got != want {
		t.Errorf("argv = %q\nwant %q", got, want)
	}
	// The command must be a single argv element (no shell re-splitting) and
	// no login/interactive flag may sneak in.
	if argv[len(argv)-1] != "echo hi" {
		t.Errorf("command is not the last argv element: %q", argv)
	}
	for _, a := range argv[:len(argv)-1] {
		if a == "-l" || a == "-lc" || a == "--login" {
			t.Errorf("unexpected flag %q in %q", a, argv)
		}
	}
}

func TestUsageFromStats(t *testing.T) {
	var s container.StatsResponse
	s.MemoryStats.Usage = 300
	s.MemoryStats.Limit = 1000
	s.MemoryStats.Stats = map[string]uint64{"inactive_file": 100}
	s.CPUStats.CPUUsage.TotalUsage = 2_000
	s.PreCPUStats.CPUUsage.TotalUsage = 1_000
	s.CPUStats.SystemUsage = 20_000
	s.PreCPUStats.SystemUsage = 10_000
	s.CPUStats.OnlineCPUs = 4

	u := usageFromStats(&s)
	if u.MemBytes != 200 || u.MemLimitBytes != 1000 {
		t.Errorf("mem = %d/%d", u.MemBytes, u.MemLimitBytes)
	}
	if u.CPUPercent != 40 {
		t.Errorf("cpu = %v, want 40", u.CPUPercent)
	}
}
