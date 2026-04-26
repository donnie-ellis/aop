package inventory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "inventory-*.ini")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestParseINI_Empty(t *testing.T) {
	path := writeFile(t, "")
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(hosts))
	}
}

func TestParseINI_CommentsAndBlanks(t *testing.T) {
	path := writeFile(t, `
# This is a comment
; Another comment

`)
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 0 {
		t.Errorf("expected 0 hosts from comment-only file, got %d", len(hosts))
	}
}

func TestParseINI_SingleHostNoGroup(t *testing.T) {
	path := writeFile(t, `
[all]
web01
`)
	projectID := uuid.New()
	hosts, err := parseINIInventory(projectID, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].Hostname != "web01" {
		t.Errorf("hostname: got %q", hosts[0].Hostname)
	}
	if hosts[0].ProjectID != projectID {
		t.Errorf("project_id not set correctly")
	}
	if len(hosts[0].Groups) != 0 {
		t.Errorf("expected no groups for host in [all], got %v", hosts[0].Groups)
	}
}

func TestParseINI_InlineVars(t *testing.T) {
	path := writeFile(t, `
[all]
web01 ansible_port=2222 ansible_user=deploy
`)
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	h := hosts[0]
	if h.Vars["ansible_port"] != "2222" {
		t.Errorf("ansible_port: got %v", h.Vars["ansible_port"])
	}
	if h.Vars["ansible_user"] != "deploy" {
		t.Errorf("ansible_user: got %v", h.Vars["ansible_user"])
	}
}

func TestParseINI_GroupMembership(t *testing.T) {
	path := writeFile(t, `
[all]
web01
db01

[web]
web01

[db]
db01
`)
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hostMap := map[string][]string{}
	for _, h := range hosts {
		hostMap[h.Hostname] = h.Groups
	}

	if groups, ok := hostMap["web01"]; !ok {
		t.Error("web01 not found")
	} else if !contains(groups, "web") {
		t.Errorf("web01 should be in [web] group; got %v", groups)
	}

	if groups, ok := hostMap["db01"]; !ok {
		t.Error("db01 not found")
	} else if !contains(groups, "db") {
		t.Errorf("db01 should be in [db] group; got %v", groups)
	}
}

func TestParseINI_MultipleGroups(t *testing.T) {
	path := writeFile(t, `
[all]
web01

[web]
web01

[prod]
web01
`)
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	groups := hosts[0].Groups
	if !contains(groups, "web") {
		t.Errorf("missing [web] group: %v", groups)
	}
	if !contains(groups, "prod") {
		t.Errorf("missing [prod] group: %v", groups)
	}
}

func TestParseINI_NoDuplicateGroups(t *testing.T) {
	// If a host appears multiple times in the same group section it should not
	// accumulate duplicate group entries.
	path := writeFile(t, `
[web]
web01
web01
`)
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	count := 0
	for _, g := range hosts[0].Groups {
		if g == "web" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected web group exactly once, got %d times", count)
	}
}

func TestParseINI_VarsAndChildrenSectionsSkipped(t *testing.T) {
	path := writeFile(t, `
[web]
web01

[web:vars]
ansible_user=ubuntu

[web:children]
subgroup
`)
	hosts, err := parseINIInventory(uuid.New(), path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// :vars and :children sections should not produce hosts.
	for _, h := range hosts {
		if h.Hostname == "ansible_user=ubuntu" || h.Hostname == "subgroup" {
			t.Errorf("unexpected host %q from vars/children section", h.Hostname)
		}
	}
}

func TestParseINI_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.ini")
	_, err := parseINIInventory(uuid.New(), path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseINI_ProjectIDPropagated(t *testing.T) {
	path := writeFile(t, "[all]\nweb01\n")
	projectID := uuid.New()
	hosts, err := parseINIInventory(projectID, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, h := range hosts {
		if h.ProjectID != projectID {
			t.Errorf("host %q has wrong project_id", h.Hostname)
		}
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
