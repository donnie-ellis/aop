package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/rs/zerolog"
)

var discardLog = zerolog.Nop()

func TestMaterialize_SSHKey(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindSSHKey, Fields: map[string]string{"private_key": "FAKEPEM"}},
	}

	c, err := Materialize(dir, creds, discardLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.SSHKeyPath == "" {
		t.Fatal("SSHKeyPath should be set")
	}

	data, err := os.ReadFile(c.SSHKeyPath)
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	if string(data) != "FAKEPEM" {
		t.Errorf("key content: got %q, want FAKEPEM", string(data))
	}

	info, err := os.Stat(c.SSHKeyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("key file perms: got %o, want 0600", info.Mode().Perm())
	}

	hasGitSSH := false
	for _, e := range c.Env {
		if strings.HasPrefix(e, "GIT_SSH_COMMAND") {
			hasGitSSH = true
		}
	}
	if !hasGitSSH {
		t.Error("GIT_SSH_COMMAND not set in Env")
	}

	c.Cleanup()
	if _, err := os.Stat(c.SSHKeyPath); !os.IsNotExist(err) {
		t.Error("key file should be removed after Cleanup")
	}
}

func TestMaterialize_SSHKey_MissingField(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindSSHKey, Fields: map[string]string{}},
	}
	_, err := Materialize(dir, creds, discardLog)
	if err == nil {
		t.Fatal("expected error for missing private_key")
	}
}

func TestMaterialize_VaultPassword(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindVaultPassword, Fields: map[string]string{"password": "s3cr3t"}},
	}

	c, err := Materialize(dir, creds, discardLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.VaultPassPath == "" {
		t.Fatal("VaultPassPath should be set")
	}

	data, _ := os.ReadFile(c.VaultPassPath)
	if string(data) != "s3cr3t" {
		t.Errorf("vault pass content: got %q", string(data))
	}

	info, _ := os.Stat(c.VaultPassPath)
	if info.Mode().Perm() != 0600 {
		t.Errorf("vault pass perms: got %o, want 0600", info.Mode().Perm())
	}

	c.Cleanup()
	if _, err := os.Stat(c.VaultPassPath); !os.IsNotExist(err) {
		t.Error("vault pass file should be removed after Cleanup")
	}
}

func TestMaterialize_VaultPassword_MissingField(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindVaultPassword, Fields: map[string]string{}},
	}
	_, err := Materialize(dir, creds, discardLog)
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestMaterialize_MachineUser(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindMachineUser, Fields: map[string]string{
			"private_key": "MACHINEKEY",
			"username":    "deploy",
		}},
	}

	c, err := Materialize(dir, creds, discardLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.SSHKeyPath == "" {
		t.Fatal("SSHKeyPath should be set for machine_user")
	}
	if c.AnsibleUser != "deploy" {
		t.Errorf("AnsibleUser: got %q, want deploy", c.AnsibleUser)
	}

	info, _ := os.Stat(c.SSHKeyPath)
	if info.Mode().Perm() != 0600 {
		t.Errorf("machine key perms: got %o, want 0600", info.Mode().Perm())
	}
}

func TestMaterialize_MachineUser_MissingField(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindMachineUser, Fields: map[string]string{"username": "deploy"}},
	}
	_, err := Materialize(dir, creds, discardLog)
	if err == nil {
		t.Fatal("expected error for missing private_key")
	}
}

func TestMaterialize_HTTPSToken_NothingWritten(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindHTTPSToken, Fields: map[string]string{"token": "ghp_abc123"}},
	}

	c, err := Materialize(dir, creds, discardLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No files should be written for HTTPS tokens.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files written, got %d", len(entries))
	}
	if c.SSHKeyPath != "" || c.VaultPassPath != "" {
		t.Error("SSHKeyPath and VaultPassPath should be empty for https_token")
	}
}

func TestMaterialize_Empty(t *testing.T) {
	dir := t.TempDir()
	c, err := Materialize(dir, nil, discardLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SSHKeyPath != "" || c.VaultPassPath != "" || c.AnsibleUser != "" {
		t.Error("empty creds should produce zeroed Creds")
	}
	c.Cleanup() // should not panic
}

func TestMaterialize_MultipleCredentials(t *testing.T) {
	dir := t.TempDir()
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindSSHKey, Fields: map[string]string{"private_key": "KEY"}},
		{Kind: types.CredentialKindVaultPassword, Fields: map[string]string{"password": "PASS"}},
	}

	c, err := Materialize(dir, creds, discardLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.SSHKeyPath == "" {
		t.Error("SSHKeyPath should be set")
	}
	if c.VaultPassPath == "" {
		t.Error("VaultPassPath should be set")
	}

	c.Cleanup()

	if _, err := os.Stat(c.SSHKeyPath); !os.IsNotExist(err) {
		t.Error("ssh key should be cleaned up")
	}
	if _, err := os.Stat(c.VaultPassPath); !os.IsNotExist(err) {
		t.Error("vault pass should be cleaned up")
	}
}

func TestHTTPSCloneURL_WithToken(t *testing.T) {
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindHTTPSToken, Fields: map[string]string{"token": "mytoken"}},
	}
	got := httpsCloneURL("https://github.com/org/repo.git", creds)
	want := "https://oauth2:mytoken@github.com/org/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestHTTPSCloneURL_NoToken(t *testing.T) {
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindSSHKey, Fields: map[string]string{"private_key": "KEY"}},
	}
	url := "https://github.com/org/repo.git"
	got := httpsCloneURL(url, creds)
	if got != url {
		t.Errorf("got %q, want %q (passthrough)", got, url)
	}
}

func TestHTTPSCloneURL_EmptyToken(t *testing.T) {
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindHTTPSToken, Fields: map[string]string{"token": ""}},
	}
	url := "https://github.com/org/repo.git"
	got := httpsCloneURL(url, creds)
	if got != url {
		t.Errorf("empty token should not modify URL; got %q", got)
	}
}

func TestHTTPSCloneURL_NonHTTPS(t *testing.T) {
	creds := []types.CredentialSecret{
		{Kind: types.CredentialKindHTTPSToken, Fields: map[string]string{"token": "tok"}},
	}
	url := "git@github.com:org/repo.git"
	got := httpsCloneURL(url, creds)
	if got != url {
		t.Errorf("non-https URL should be unchanged; got %q", got)
	}
}

func TestWriteSecret_CreatesFileWithCorrectPerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := writeSecret(path, []byte("data")); err != nil {
		t.Fatalf("writeSecret: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("perms: got %o, want 0600", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if string(data) != "data" {
		t.Errorf("content: got %q", string(data))
	}
}
