package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTask(t *testing.T, root, dir, yaml, readme string) {
	t.Helper()
	p := filepath.Join(root, dir)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "task.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if readme != "" {
		if err := os.WriteFile(filepath.Join(p, "README.md"), []byte(readme), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadDirDefaultsAndOrder(t *testing.T) {
	root := t.TempDir()
	writeTask(t, root, "beta", `
id: beta
title: Beta
order: 2
difficulty: 2
category: Containers
points: 250
hints: ["first", "second"]
setup: ["echo secret-setup"]
checks:
  - name: works
    cmd: echo secret-cmd
`, "# Beta\n\nDo the beta thing.\n")
	writeTask(t, root, "alpha", `
title: Alpha
order: 1
difficulty: 9
`, "# Alpha\n\n- list item\n\nFirst paragraph line that becomes the description.\n")
	writeTask(t, root, "Bad_ID", "title: bad\n", "")
	writeTask(t, root, "broken", "title: [unterminated\n", "")
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "alpha" || got[1].ID != "beta" {
		t.Fatalf("unexpected tasks: %+v", got)
	}

	a, b := got[0], got[1]
	if a.Difficulty != 3 || a.Points != 300 || a.Category != DefaultCategory {
		t.Fatalf("alpha defaults: %+v", a)
	}
	if a.Description != "First paragraph line that becomes the description." {
		t.Fatalf("alpha description from README: %q", a.Description)
	}
	if a.Hints == nil || len(a.Hints) != 0 {
		t.Fatalf("alpha hints must be empty, non-nil: %#v", a.Hints)
	}
	if b.Points != 250 || b.Category != "containers" || len(b.Hints) != 2 || !strings.Contains(b.Readme, "beta thing") {
		t.Fatalf("beta fields: %+v", b)
	}
	if b.Checks[0].Cmd != "echo secret-cmd" || len(b.Setup) != 1 {
		t.Fatal("cmd/setup must still be loaded for the runtime")
	}
	raw, _ := json.Marshal(b)
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("task JSON leaks cmd/setup: %s", raw)
	}
}

func TestReloadKeepsTasksOnFailure(t *testing.T) {
	root := t.TempDir()
	writeTask(t, root, "one", "title: One\norder: 1\n", "")
	if err := Init(root); err != nil {
		t.Fatal(err)
	}
	if len(GetTasks()) != 1 {
		t.Fatal("expected one task")
	}
	if _, err := GetTask("one"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetTask("missing"); err == nil {
		t.Fatal("missing task must error")
	}

	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := ReloadTasks(); err == nil {
		t.Fatal("reload of a missing dir must fail")
	}
	if len(GetTasks()) != 1 {
		t.Fatal("failed reload must keep previously loaded tasks")
	}

	// GetTasks returns a copy.
	ts := GetTasks()
	ts[0].Title = "mutated"
	if GetTasks()[0].Title == "mutated" {
		t.Fatal("GetTasks must return a copy")
	}
}

func TestRepositoryTasksLoad(t *testing.T) {
	dir := filepath.Join("..", "..", "tasks")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("tasks directory not present")
	}
	got, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("expected repository tasks")
	}
	for _, tk := range got {
		if !IDPattern.MatchString(tk.ID) || tk.Title == "" || tk.Points <= 0 || len(tk.Checks) == 0 {
			t.Errorf("task %q is incomplete: %+v", tk.ID, tk)
		}
		if tk.Difficulty < 1 || tk.Difficulty > 3 {
			t.Errorf("task %q difficulty %d", tk.ID, tk.Difficulty)
		}
	}
}
