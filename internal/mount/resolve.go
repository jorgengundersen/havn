package mount

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jorgengundersen/havn/internal/config"
)

// Resolve produces all mount specs and environment variables needed for
// container creation based on the merged config, project path, and home
// directory.
func Resolve(cfg config.Config, projectPath, homeDir string, opts ResolveOpts) (ResolveResult, error) {
	var mounts []Spec

	// 1. Project directory — always present, rw.
	mounts = append(mounts, Spec{
		Source:   projectPath,
		Target:   projectPath,
		ReadOnly: false,
		Type:     "bind",
	})

	// 2. Config file mounts.
	configMounts, err := resolveConfigMounts(cfg.Mounts.Config, homeDir, opts)
	if err != nil {
		return ResolveResult{}, err
	}
	mounts = append(mounts, configMounts...)

	// 3. Runtime environment and SSH agent forwarding.
	env := map[string]string{
		"NIX_CONFIG": "flake-registry = " + nixRegistryPath,
	}
	sshMounts := resolveSSHMounts(cfg.Mounts.SSH, homeDir, opts)
	mounts = append(mounts, sshMounts.Mounts...)
	for k, v := range sshMounts.Env {
		env[k] = v
	}

	// 4. Named volumes.
	mounts = append(mounts,
		Spec{Source: cfg.Volumes.Nix, Target: "/nix", ReadOnly: false, Type: "volume"},
		Spec{Source: cfg.Volumes.Data, Target: "/home/devuser/.local/share", ReadOnly: false, Type: "volume"},
		Spec{Source: cfg.Volumes.Cache, Target: "/home/devuser/.cache", ReadOnly: false, Type: "volume"},
		Spec{Source: cfg.Volumes.State, Target: "/home/devuser/.local/state", ReadOnly: false, Type: "volume"},
	)

	return ResolveResult{
		Mounts:       mounts,
		ConfigMounts: configMounts,
		Env:          env,
	}, nil
}

const (
	containerHome   = "/home/devuser"
	nixRegistryPath = "/home/devuser/.local/state/nix/registry.json"
)

// parseConfigEntry splits a "path:mode" entry and validates the mode.
func parseConfigEntry(entry string) (path, mode string, err error) {
	parts := strings.SplitN(entry, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", "", &InvalidMountEntryError{Entry: entry, Reason: "missing mode (expected path:ro or path:rw)"}
	}
	mode = parts[1]
	if mode != "ro" && mode != "rw" {
		return "", "", &InvalidMountEntryError{Entry: entry, Reason: fmt.Sprintf("unknown mode %q (expected ro or rw)", mode)}
	}
	return parts[0], mode, nil
}

// expandPath resolves ~ and environment variables in a mount path, then
// makes it absolute relative to homeDir.
func expandPath(raw, homeDir string) string {
	p := raw
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(homeDir, p[2:])
	} else if p == "~" {
		p = homeDir
	} else if !filepath.IsAbs(p) {
		// Relative paths are relative to home.
		p = filepath.Join(homeDir, p)
	}
	p = os.Expand(p, os.Getenv)
	return p
}

// resolveConfigMounts resolves the mounts.config entries to Specs.
func resolveConfigMounts(entries []string, homeDir string, opts ResolveOpts) ([]Spec, error) {
	var mounts []Spec
	homeAbs := filepath.Clean(homeDir)
	for _, entry := range entries {
		relPath, mode, err := parseConfigEntry(entry)
		if err != nil {
			return nil, err
		}

		hostPattern := expandPath(relPath, homeDir)
		readOnly := mode == "ro"

		matches, err := opts.Glob(hostPattern)
		if err != nil {
			return nil, fmt.Errorf("glob config mount %q: %w", entry, err)
		}

		for _, hostPath := range matches {
			if !opts.Exists(hostPath) {
				continue
			}

			hostAbs := filepath.Clean(hostPath)
			relToHome, err := filepath.Rel(homeAbs, hostAbs)
			if err != nil || relToHome == ".." || strings.HasPrefix(relToHome, ".."+string(filepath.Separator)) {
				return nil, &InvalidMountEntryError{Entry: entry, Reason: "path resolves outside the home directory"}
			}
			// Compute relative path from homeDir to derive container target.
			rel, err := filepath.Rel(homeDir, hostAbs)
			if err != nil {
				rel = filepath.Base(hostPath)
			}
			target := filepath.Join(containerHome, rel)

			mounts = append(mounts, Spec{
				Source:   hostPath,
				Target:   target,
				ReadOnly: readOnly,
				Type:     "bind",
			})
		}
	}
	return mounts, nil
}

// resolveSSHMounts produces mounts and env vars for SSH agent/key forwarding.
func resolveSSHMounts(ssh config.SSHConfig, homeDir string, opts ResolveOpts) ResolveResult {
	var mounts []Spec
	env := make(map[string]string)

	if ssh.ForwardAgent && opts.SSHAuthSock != "" && opts.Exists(opts.SSHAuthSock) {
		mounts = append(mounts, Spec{
			Source:   opts.SSHAuthSock,
			Target:   "/ssh-agent",
			ReadOnly: true,
			Type:     "bind",
		})
		env["SSH_AUTH_SOCK"] = "/ssh-agent"
	}

	if ssh.AuthorizedKeys {
		authKeysPath := filepath.Join(homeDir, ".ssh", "authorized_keys")
		if opts.Exists(authKeysPath) {
			mounts = append(mounts, Spec{
				Source:   authKeysPath,
				Target:   filepath.Join(containerHome, ".ssh", "authorized_keys"),
				ReadOnly: true,
				Type:     "bind",
			})
		}
	}

	return ResolveResult{Mounts: mounts, Env: env}
}
