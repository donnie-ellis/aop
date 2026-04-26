package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/donnie-ellis/aop/pkg/types"
	"github.com/rs/zerolog"
)

// Creds holds the materialised state of all credentials for one job.
// Env contains extra environment variables to set on the ansible-playbook
// and git commands. Cleanup removes all written files; it is always called
// from a deferred block regardless of job outcome.
type Creds struct {
	SSHKeyPath    string   // absolute path to PEM key file (ssh_key / machine_user)
	VaultPassPath string   // absolute path to vault password file
	AnsibleUser   string   // remote_user override (machine_user only)
	Env           []string // KEY=VALUE pairs to append to command env
	Cleanup       func()
}

// Materialize writes credential secrets to the job work directory.
// Files are created with mode 0600. Paths are never logged.
func Materialize(jobDir string, creds []types.CredentialSecret, log zerolog.Logger) (*Creds, error) {
	result := &Creds{}
	var written []string

	result.Cleanup = func() {
		for _, p := range written {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				log.Warn().Str("kind", "credential").Msg("cleanup: failed to remove temp file")
			}
		}
	}

	for _, c := range creds {
		switch c.Kind {
		case types.CredentialKindSSHKey:
			key := c.Fields["private_key"]
			if key == "" {
				return nil, fmt.Errorf("ssh_key credential is missing private_key field")
			}
			path := filepath.Join(jobDir, "id_rsa")
			if err := writeSecret(path, []byte(key)); err != nil {
				return nil, fmt.Errorf("write ssh key: %w", err)
			}
			written = append(written, path)
			result.SSHKeyPath = path
			result.Env = append(result.Env,
				"GIT_SSH_COMMAND=ssh -i "+path+" -o StrictHostKeyChecking=no -o BatchMode=yes",
			)

		case types.CredentialKindMachineUser:
			key := c.Fields["private_key"]
			if key == "" {
				return nil, fmt.Errorf("machine_user credential is missing private_key field")
			}
			path := filepath.Join(jobDir, "machine_id_rsa")
			if err := writeSecret(path, []byte(key)); err != nil {
				return nil, fmt.Errorf("write machine_user key: %w", err)
			}
			written = append(written, path)
			result.SSHKeyPath = path
			result.AnsibleUser = c.Fields["username"]

		case types.CredentialKindVaultPassword:
			pass := c.Fields["password"]
			if pass == "" {
				return nil, fmt.Errorf("vault_password credential is missing password field")
			}
			path := filepath.Join(jobDir, "vault_pass.txt")
			if err := writeSecret(path, []byte(pass)); err != nil {
				return nil, fmt.Errorf("write vault password: %w", err)
			}
			written = append(written, path)
			result.VaultPassPath = path

		case types.CredentialKindHTTPSToken:
			// HTTPS tokens are embedded in the clone URL; nothing to write to disk.
		}
	}

	return result, nil
}

// httpsCloneURL returns repoURL with the token embedded as a userinfo prefix
// if an https_token credential is present.
func httpsCloneURL(repoURL string, creds []types.CredentialSecret) string {
	for _, c := range creds {
		if c.Kind == types.CredentialKindHTTPSToken {
			tok := c.Fields["token"]
			if tok == "" {
				continue
			}
			// Insert oauth2:<token>@ after the scheme.
			// https://github.com/... → https://oauth2:TOKEN@github.com/...
			const prefix = "https://"
			if len(repoURL) > len(prefix) && repoURL[:len(prefix)] == prefix {
				return prefix + "oauth2:" + tok + "@" + repoURL[len(prefix):]
			}
		}
	}
	return repoURL
}

func writeSecret(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}
