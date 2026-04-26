package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donnie-ellis/aop/pkg/types"
)

func readInventory(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	return string(data)
}

func TestWriteInventory_Empty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.ini")
	if err := WriteInventory(path, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := readInventory(t, path)
	if !strings.Contains(content, "[all]") {
		t.Error("should contain [all] section even when empty")
	}
}

func TestWriteInventory_SingleHostNoGroups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.ini")
	hosts := []types.InventoryHost{
		{Hostname: "web01", Groups: nil, Vars: nil},
	}
	if err := WriteInventory(path, hosts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := readInventory(t, path)

	if !strings.Contains(content, "[all]") {
		t.Error("missing [all] section")
	}
	if !strings.Contains(content, "web01") {
		t.Error("missing hostname web01")
	}
	if !strings.Contains(content, "ansible_host=web01") {
		t.Error("missing ansible_host var")
	}
}

func TestWriteInventory_HostVars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.ini")
	hosts := []types.InventoryHost{
		{
			Hostname: "db01",
			Vars:     map[string]any{"ansible_port": "2222", "ansible_python_interpreter": "/usr/bin/python3"},
		},
	}
	if err := WriteInventory(path, hosts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := readInventory(t, path)

	if !strings.Contains(content, "ansible_port=2222") {
		t.Error("missing ansible_port var")
	}
	if !strings.Contains(content, "ansible_python_interpreter=/usr/bin/python3") {
		t.Error("missing ansible_python_interpreter var")
	}
}

func TestWriteInventory_Groups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.ini")
	hosts := []types.InventoryHost{
		{Hostname: "web01", Groups: []string{"web", "prod"}},
		{Hostname: "web02", Groups: []string{"web"}},
		{Hostname: "db01", Groups: []string{"db", "prod"}},
	}
	if err := WriteInventory(path, hosts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := readInventory(t, path)

	if !strings.Contains(content, "[web]") {
		t.Error("missing [web] group")
	}
	if !strings.Contains(content, "[db]") {
		t.Error("missing [db] group")
	}
	if !strings.Contains(content, "[prod]") {
		t.Error("missing [prod] group")
	}

	// web group should have both web01 and web02
	webIdx := strings.Index(content, "[web]")
	if webIdx == -1 {
		t.Fatal("[web] section not found")
	}
	webSection := content[webIdx:]
	if !strings.Contains(webSection, "web01") {
		t.Error("web01 should be in [web] group")
	}
	if !strings.Contains(webSection, "web02") {
		t.Error("web02 should be in [web] group")
	}
}

func TestWriteInventory_AllSectionContainsAllHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.ini")
	hosts := []types.InventoryHost{
		{Hostname: "alpha"},
		{Hostname: "beta"},
		{Hostname: "gamma"},
	}
	if err := WriteInventory(path, hosts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := readInventory(t, path)

	allIdx := strings.Index(content, "[all]")
	if allIdx == -1 {
		t.Fatal("[all] not found")
	}
	// All hosts must appear in the all section (before first group or end of file).
	nextSection := strings.Index(content[allIdx+1:], "[")
	var allSection string
	if nextSection == -1 {
		allSection = content[allIdx:]
	} else {
		allSection = content[allIdx : allIdx+1+nextSection]
	}

	for _, h := range hosts {
		if !strings.Contains(allSection, h.Hostname) {
			t.Errorf("host %q not found in [all] section", h.Hostname)
		}
	}
}

func TestWriteInventory_FileCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.ini")
	if err := WriteInventory(path, []types.InventoryHost{{Hostname: "h1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}
