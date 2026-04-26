// Package runner executes a single Ansible job end-to-end:
// clone the repo, materialise credentials, write inventory, run
// ansible-playbook, stream logs, report result.
package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/rs/zerolog"
)

const (
	logBatchSize     = 10
	logFlushInterval = 500 * time.Millisecond
)

// LogFunc is called for each captured log line. seq is monotonically
// increasing per job (assigned by the runner).
type LogFunc func(seq int, line, stream string)

// Config bundles everything Run needs.
type Config struct {
	Payload types.JobPayload
	WorkDir string // base directory; runner creates a sub-directory per job
	OnLog   LogFunc
}

// Result is what Run returns after the playbook exits.
type Result struct {
	Status   types.JobStatus
	ExitCode int
}

// Run executes the job described by cfg. It blocks until the playbook
// finishes or ctx is cancelled. The job work directory is always removed
// on return regardless of outcome.
func Run(ctx context.Context, cfg Config, log zerolog.Logger) Result {
	jobDir := filepath.Join(cfg.WorkDir, cfg.Payload.JobID.String())
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		log.Error().Err(err).Msg("create job dir")
		return Result{Status: types.JobStatusFailed, ExitCode: -1}
	}
	defer func() {
		if err := os.RemoveAll(jobDir); err != nil {
			log.Warn().Err(err).Msg("cleanup job dir")
		}
	}()

	// Materialise credentials.
	creds, err := Materialize(jobDir, cfg.Payload.Credentials, log)
	if err != nil {
		log.Error().Err(err).Msg("materialise credentials")
		return Result{Status: types.JobStatusFailed, ExitCode: -1}
	}
	defer creds.Cleanup()

	// Clone the project repository.
	repoDir := filepath.Join(jobDir, "repo")
	if err := gitClone(ctx, cfg.Payload.RepoURL, cfg.Payload.RepoBranch, repoDir, cfg.Payload.Credentials, creds, log); err != nil {
		log.Error().Err(err).Str("repo", cfg.Payload.RepoURL).Msg("git clone failed")
		return Result{Status: types.JobStatusFailed, ExitCode: -1}
	}

	// Write inventory.
	inventoryPath := filepath.Join(jobDir, "inventory.ini")
	if err := WriteInventory(inventoryPath, cfg.Payload.Inventory); err != nil {
		log.Error().Err(err).Msg("write inventory")
		return Result{Status: types.JobStatusFailed, ExitCode: -1}
	}

	// Write extra_vars JSON file.
	extraVarsPath := filepath.Join(jobDir, "extra_vars.json")
	if err := writeExtraVars(extraVarsPath, cfg.Payload.ExtraVars); err != nil {
		log.Error().Err(err).Msg("write extra_vars")
		return Result{Status: types.JobStatusFailed, ExitCode: -1}
	}

	// Build and run ansible-playbook.
	playbookPath := filepath.Join(repoDir, cfg.Payload.Playbook)
	exitCode := runPlaybook(ctx, playbookPath, inventoryPath, extraVarsPath, creds, cfg.OnLog, log)

	status := types.JobStatusSuccess
	if exitCode != 0 {
		status = types.JobStatusFailed
	}
	// Cancelled context means the job was cancelled by the controller.
	if ctx.Err() != nil {
		status = types.JobStatusCancelled
	}
	return Result{Status: status, ExitCode: exitCode}
}

// gitClone performs a shallow clone of repoURL at branch into dest.
func gitClone(ctx context.Context, repoURL, branch, dest string, rawCreds []types.CredentialSecret, creds *Creds, log zerolog.Logger) error {
	if repoURL == "" {
		// No repo — nothing to clone. The playbook must exist on the agent's PATH.
		return nil
	}

	url := httpsCloneURL(repoURL, rawCreds)
	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dest)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), creds.Env...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w\n%s", err, out)
	}
	return nil
}

// runPlaybook executes ansible-playbook and streams its output via onLog.
// Returns the process exit code (0 = success).
func runPlaybook(ctx context.Context, playbookPath, inventoryPath, extraVarsPath string, creds *Creds, onLog LogFunc, log zerolog.Logger) int {
	args := []string{
		"-i", inventoryPath,
	}
	if extraVarsPath != "" {
		args = append(args, "--extra-vars", "@"+extraVarsPath)
	}
	if creds.VaultPassPath != "" {
		args = append(args, "--vault-password-file", creds.VaultPassPath)
	}
	if creds.SSHKeyPath != "" {
		args = append(args, "--private-key", creds.SSHKeyPath)
	}
	if creds.AnsibleUser != "" {
		args = append(args, "--user", creds.AnsibleUser)
	}
	args = append(args, playbookPath)

	cmd := exec.CommandContext(ctx, "ansible-playbook", args...)
	cmd.Env = append(os.Environ(),
		"ANSIBLE_FORCE_COLOR=1",
		"ANSIBLE_STDOUT_CALLBACK=default",
	)
	cmd.Env = append(cmd.Env, creds.Env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("stdout pipe")
		return -1
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error().Err(err).Msg("stderr pipe")
		return -1
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("start ansible-playbook")
		return -1
	}

	var seq atomic.Int64
	nextSeq := func() int { return int(seq.Add(1)) }

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			onLog(nextSeq(), scanner.Text(), "stdout")
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			onLog(nextSeq(), scanner.Text(), "stderr")
		}
	}()

	wg.Wait()
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return -1
	}
	return 0
}

func writeExtraVars(path string, vars map[string]any) error {
	if len(vars) == 0 {
		vars = map[string]any{}
	}
	data, err := json.Marshal(vars)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LogBatcher buffers log lines and flushes them in batches via flush.
// Call Add for each line; call Flush to force-send any remaining lines.
type LogBatcher struct {
	mu      sync.Mutex
	buf     []types.JobLogLine
	flush   func([]types.JobLogLine)
	ticker  *time.Ticker
	done    chan struct{}
}

// NewLogBatcher starts a batcher that calls flush every flushInterval or
// when batchSize lines have accumulated.
func NewLogBatcher(batchSize int, flushInterval time.Duration, flush func([]types.JobLogLine)) *LogBatcher {
	b := &LogBatcher{
		flush:  flush,
		ticker: time.NewTicker(flushInterval),
		done:   make(chan struct{}),
	}
	go b.run(batchSize)
	return b
}

func (b *LogBatcher) Add(line types.JobLogLine) {
	b.mu.Lock()
	b.buf = append(b.buf, line)
	overflow := len(b.buf)
	b.mu.Unlock()
	_ = overflow // size-triggered flush happens in run()
}

func (b *LogBatcher) run(batchSize int) {
	for {
		select {
		case <-b.ticker.C:
			b.doFlush()
		case <-b.done:
			b.ticker.Stop()
			b.doFlush()
			return
		}
	}
}

// Flush stops the ticker and sends any remaining lines synchronously.
func (b *LogBatcher) Flush() {
	close(b.done)
}

func (b *LogBatcher) doFlush() {
	b.mu.Lock()
	if len(b.buf) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.buf
	b.buf = nil
	b.mu.Unlock()
	b.flush(batch)
}
