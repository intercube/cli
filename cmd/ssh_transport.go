package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/intercube/cli/util/inventory"
	"github.com/spf13/cobra"
)

func gatewayConnection(access inventory.SSHAccess, target inventory.SSHTarget) ResolvedSyncTarget {
	id := ""
	if target.SiteID != nil {
		id = strconv.Itoa(*target.SiteID)
	}
	return ResolvedSyncTarget{SiteID: id, DisplayName: target.Name, ServerName: target.ServerName, ServerAddress: target.Address, Host: access.Host, Port: access.Port, Username: access.Username + ":" + target.Name}
}

func gatewayControlPath(target ResolvedSyncTarget) (string, error) {
	if target.Host == "" || strings.HasPrefix(target.Host, "-") || strings.ContainsAny(target.Host, " \t\r\n/") || target.Port < 1 || target.Port > 65535 || !regexp.MustCompile(`^nexus-user_[A-Za-z0-9_]+:nexus-(site|root)-[0-9]+$`).MatchString(target.Username) {
		return "", fmt.Errorf("inventory returned invalid gateway connection details")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	directory := filepath.Join(cache, "intercube", "ssh")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	if err := os.Chmod(directory, 0700); err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d/%s", target.Host, target.Port, target.Username)))
	path := filepath.Join(directory, fmt.Sprintf("%x", hash[:12]))
	if len(path) > 100 {
		return "", fmt.Errorf("SSH control socket path is too long: shorten the user cache directory")
	}
	return path, nil
}

func gatewaySSHArgs(target ResolvedSyncTarget, socket string) []string {
	return []string{"-p", strconv.Itoa(target.Port), "-l", target.Username,
		"-o", "ControlMaster=auto", "-o", "ControlPersist=300", "-o", "ControlPath=" + socket,
		"-o", "PubkeyAuthentication=no", "-o", "PreferredAuthentications=keyboard-interactive,password",
		"-o", "ExitOnForwardFailure=yes"}
}

func ensureGatewayMaster(cmd *cobra.Command, target ResolvedSyncTarget, socket string) error {
	check := exec.CommandContext(cmd.Context(), "ssh", "-S", socket, "-O", "check", "-p", strconv.Itoa(target.Port), "-l", target.Username, "--", target.Host)
	if check.Run() == nil {
		return nil
	}
	if err := ensureInteractiveMode("Warpgate browser approval; first run `intercube ssh <site> --connect-only`"); err != nil {
		return err
	}
	arguments := append(gatewaySSHArgs(target, socket), "-N", "-f", "--", target.Host)
	command := exec.CommandContext(cmd.Context(), "ssh", arguments...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
