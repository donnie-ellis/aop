// Package inventory implements the InventorySource interface for projects
// that store their Ansible inventory in a Git repository as an INI file.
package inventory

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// inventoryStore is the narrow DB interface GitFileSource needs.
type inventoryStore interface {
	GetAllProjects(ctx context.Context) ([]types.Project, error)
	GetCredentialWithSecret(ctx context.Context, id uuid.UUID) (*types.Credential, error)
	UpsertInventoryHosts(ctx context.Context, projectID uuid.UUID, hosts []types.InventoryHost) error
	UpdateProjectSyncStatus(ctx context.Context, id uuid.UUID, status types.InventorySyncStatus, syncErr string) error
}

// GitSync periodically clones/pulls all projects and syncs their inventory
// into the database. It satisfies the InventorySource interface conceptually —
// in v1 it operates as a background goroutine rather than being called per-job.
type GitSync struct {
	store        inventoryStore
	workspaceDir string
	log          zerolog.Logger
}

func NewGitSync(store inventoryStore, workspaceDir string, log zerolog.Logger) *GitSync {
	return &GitSync{store: store, workspaceDir: workspaceDir, log: log}
}

// Run blocks, syncing all projects on every interval until ctx is cancelled.
func (g *GitSync) Run(ctx context.Context, interval time.Duration) {
	if err := os.MkdirAll(g.workspaceDir, 0755); err != nil {
		g.log.Error().Err(err).Str("dir", g.workspaceDir).Msg("create workspace dir")
	}

	g.syncAll(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.syncAll(ctx)
		}
	}
}

func (g *GitSync) syncAll(ctx context.Context) {
	projects, err := g.store.GetAllProjects(ctx)
	if err != nil {
		g.log.Error().Err(err).Msg("load projects")
		return
	}
	for _, p := range projects {
		if p.RepoURL == "" {
			continue
		}
		if err := g.syncProject(ctx, p); err != nil {
			g.log.Error().Err(err).Str("project_id", p.ID.String()).Str("name", p.Name).Msg("sync project")
		}
	}
}

func (g *GitSync) syncProject(ctx context.Context, p types.Project) error {
	repoDir := filepath.Join(g.workspaceDir, p.ID.String())

	cloneURL := p.RepoURL
	var gitSSH string
	if p.CredentialID != nil {
		cred, err := g.store.GetCredentialWithSecret(ctx, *p.CredentialID)
		if err != nil {
			return fmt.Errorf("get credential: %w", err)
		}
		switch cred.Kind {
		case types.CredentialKindHTTPSToken:
			tok := "" // would need SecretsProvider to decrypt; simplified for v1
			_ = tok   // HTTPS token injection requires the decrypted fields
		case types.CredentialKindSSHKey:
			_ = gitSSH // SSH key handling would write a temp key file
		}
	}

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		if err := cloneRepo(ctx, cloneURL, p.Branch, repoDir, gitSSH); err != nil {
			syncErr := err.Error()
			_ = g.store.UpdateProjectSyncStatus(ctx, p.ID, types.InventorySyncStatusFailed, syncErr)
			return fmt.Errorf("clone: %w", err)
		}
	} else {
		if err := fetchReset(ctx, repoDir, p.Branch, gitSSH); err != nil {
			syncErr := err.Error()
			_ = g.store.UpdateProjectSyncStatus(ctx, p.ID, types.InventorySyncStatusFailed, syncErr)
			return fmt.Errorf("fetch/reset: %w", err)
		}
	}

	inventoryPath := filepath.Join(repoDir, p.InventoryPath)
	hosts, err := parseINIInventory(p.ID, inventoryPath)
	if err != nil {
		syncErr := err.Error()
		_ = g.store.UpdateProjectSyncStatus(ctx, p.ID, types.InventorySyncStatusFailed, syncErr)
		return fmt.Errorf("parse inventory: %w", err)
	}

	if err := g.store.UpsertInventoryHosts(ctx, p.ID, hosts); err != nil {
		syncErr := err.Error()
		_ = g.store.UpdateProjectSyncStatus(ctx, p.ID, types.InventorySyncStatusFailed, syncErr)
		return fmt.Errorf("upsert hosts: %w", err)
	}

	if err := g.store.UpdateProjectSyncStatus(ctx, p.ID, types.InventorySyncStatusOK, ""); err != nil {
		g.log.Warn().Err(err).Str("project_id", p.ID.String()).Msg("update sync status")
	}

	g.log.Info().
		Str("project_id", p.ID.String()).
		Str("name", p.Name).
		Int("hosts", len(hosts)).
		Msg("inventory synced")
	return nil
}

func cloneRepo(ctx context.Context, url, branch, dest, gitSSH string) error {
	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dest)
	return runGit(ctx, gitSSH, args...)
}

func fetchReset(ctx context.Context, repoDir, branch, gitSSH string) error {
	if err := runGitIn(ctx, repoDir, gitSSH, "fetch", "--depth", "1", "origin", branch); err != nil {
		return err
	}
	return runGitIn(ctx, repoDir, gitSSH, "reset", "--hard", "origin/"+branch)
}

func runGit(ctx context.Context, gitSSH string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = gitEnv(gitSSH)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", args[0], err, bytes.TrimSpace(out))
	}
	return nil
}

func runGitIn(ctx context.Context, dir, gitSSH string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(gitSSH)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\n%s", args[0], err, bytes.TrimSpace(out))
	}
	return nil
}

func gitEnv(sshCmd string) []string {
	env := os.Environ()
	if sshCmd != "" {
		env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	}
	return env
}

// parseINIInventory reads an Ansible INI-format inventory file and returns
// one InventoryHost per host entry. Groups are collected from section headers.
// This is a simplified parser; it does not handle child groups, group vars
// sections, or :vars / :children qualifiers (v2 enhancement).
func parseINIInventory(projectID uuid.UUID, path string) ([]types.InventoryHost, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	hostMap := map[string]*types.InventoryHost{} // hostname → host
	var currentGroup string
	var skipSection bool // true inside :vars / :children sections

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section header: [groupname] or [groupname:vars] / [groupname:children]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			if strings.Contains(section, ":") {
				// :vars and :children sections — skip lines until next section.
				currentGroup = ""
				skipSection = true
			} else {
				currentGroup = section
				skipSection = false
			}
			continue
		}

		if skipSection {
			continue
		}

		// Host entry: hostname [var=val ...]
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		hostname := parts[0]

		h, exists := hostMap[hostname]
		if !exists {
			h = &types.InventoryHost{
				ProjectID: projectID,
				Hostname:  hostname,
				Vars:      map[string]any{},
			}
			hostMap[hostname] = h
		}

		// Parse inline vars.
		for _, kv := range parts[1:] {
			if idx := strings.Index(kv, "="); idx > 0 {
				h.Vars[kv[:idx]] = kv[idx+1:]
			}
		}

		// Assign group (avoid duplicates and skip [all]).
		if currentGroup != "" && currentGroup != "all" {
			found := false
			for _, g := range h.Groups {
				if g == currentGroup {
					found = true
					break
				}
			}
			if !found {
				h.Groups = append(h.Groups, currentGroup)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan inventory: %w", err)
	}

	hosts := make([]types.InventoryHost, 0, len(hostMap))
	for _, h := range hostMap {
		hosts = append(hosts, *h)
	}
	return hosts, nil
}
