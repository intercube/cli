package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type mysqlSyncConfig struct {
	SourceDatabase    string
	SourceUser        string
	SourceHost        string
	SourcePort        int
	SourcePasswordEnv string
	TargetDatabase    string
	TargetUser        string
	TargetHost        string
	TargetPort        int
	TargetPasswordEnv string
	DatabaseSSH       syncSSHTarget
	DumpFlags         []string
}

type syncSSHTarget struct {
	Host     string
	Username string
	Port     int
}

func runDatabaseSync(cmd *cobra.Command, target ResolvedSyncTarget, _ *SyncSettings, dryRun bool, autoApprove bool) error {
	if isNonInteractiveMode() {
		return fmt.Errorf("database sync requires interactive prompts in current implementation; run with an interactive terminal")
	}

	if err := ensureCommandAvailable("mysqldump"); err != nil {
		return err
	}
	if err := ensureCommandAvailable("ssh"); err != nil {
		return err
	}

	databaseConfig, err := promptMySQLSyncConfig(target)
	if err != nil {
		return err
	}

	if isSameDatabaseTarget(databaseConfig) {
		return fmt.Errorf("source and target resolve to the same database destination")
	}

	fmt.Println("Database sync plan:")
	fmt.Printf("  Source: %s@%s:%d/%s\n", databaseConfig.SourceUser, databaseConfig.SourceHost, databaseConfig.SourcePort, databaseConfig.SourceDatabase)
	fmt.Printf("  Target: %s@%s:%d/%s (via %s@%s:%d)\n", databaseConfig.TargetUser, databaseConfig.TargetHost, databaseConfig.TargetPort, databaseConfig.TargetDatabase, databaseConfig.DatabaseSSH.Username, databaseConfig.DatabaseSSH.Host, databaseConfig.DatabaseSSH.Port)

	if !autoApprove {
		confirmed, confirmErr := promptYesNo("Continue with MySQL import into target?")
		if confirmErr != nil {
			return confirmErr
		}

		if !confirmed {
			fmt.Println("Database sync cancelled.")
			return nil
		}
	}

	sourcePassword := os.Getenv(databaseConfig.SourcePasswordEnv)
	if strings.TrimSpace(sourcePassword) == "" {
		return fmt.Errorf("source password env var %q is empty", databaseConfig.SourcePasswordEnv)
	}

	targetPassword := os.Getenv(databaseConfig.TargetPasswordEnv)
	if strings.TrimSpace(targetPassword) == "" {
		return fmt.Errorf("target password env var %q is empty", databaseConfig.TargetPasswordEnv)
	}

	dumpArgs := buildMySQLDumpArgs(databaseConfig)
	remoteCommand := buildRemoteMySQLImportCommand(databaseConfig, targetPassword)
	sshArgs := buildDatabaseSSHArgs(databaseConfig.DatabaseSSH, remoteCommand)

	fmt.Printf("Running: MYSQL_PWD=<hidden> mysqldump %s | ssh %s\n", strings.Join(dumpArgs, " "), strings.Join(sshArgs, " "))

	if dryRun {
		return nil
	}

	sshCommand := exec.CommandContext(cmd.Context(), "ssh", sshArgs...)
	sshCommand.Stdout = os.Stdout
	sshCommand.Stderr = os.Stderr

	stdinPipe, err := sshCommand.StdinPipe()
	if err != nil {
		return err
	}

	if err := sshCommand.Start(); err != nil {
		return err
	}

	dumpCommand := exec.CommandContext(cmd.Context(), "mysqldump", dumpArgs...)
	dumpCommand.Stdout = stdinPipe
	dumpCommand.Stderr = os.Stderr
	dumpCommand.Env = append(os.Environ(), "MYSQL_PWD="+sourcePassword)

	dumpErr := dumpCommand.Run()
	_ = stdinPipe.Close()
	sshErr := sshCommand.Wait()

	if dumpErr != nil {
		return dumpErr
	}

	if sshErr != nil {
		return sshErr
	}

	return nil
}

func ensureCommandAvailable(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("required command %q not found", name)
	}

	return nil
}

func promptMySQLSyncConfig(target ResolvedSyncTarget) (mysqlSyncConfig, error) {
	sourceDatabase, err := promptText("Source MySQL database", "", requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	sourceUser, err := promptText("Source MySQL user", "root", requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	sourceHost, err := promptText("Source MySQL host", "127.0.0.1", requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	sourcePort, err := promptPort("Source MySQL port", "3306")
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	sourcePasswordEnv, err := promptText("Source DB password env var", "SYNC_SOURCE_DB_PASSWORD", requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	targetDatabase, err := promptText("Target MySQL database", sourceDatabase, requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	targetUser, err := promptText("Target MySQL user", sourceUser, requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	targetHost, err := promptText("Target MySQL host (on target server)", "127.0.0.1", requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	targetPort, err := promptPort("Target MySQL port", "3306")
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	targetPasswordEnv, err := promptText("Target DB password env var", "SYNC_TARGET_DB_PASSWORD", requiredValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	databaseSSH, err := promptDatabaseSSHTarget(target)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	flagsRaw, err := promptText("Extra mysqldump flags (space-separated, optional)", "--single-transaction --quick", optionalValue, 0)
	if err != nil {
		return mysqlSyncConfig{}, err
	}

	return mysqlSyncConfig{
		SourceDatabase:    strings.TrimSpace(sourceDatabase),
		SourceUser:        strings.TrimSpace(sourceUser),
		SourceHost:        strings.TrimSpace(sourceHost),
		SourcePort:        sourcePort,
		SourcePasswordEnv: strings.TrimSpace(sourcePasswordEnv),
		TargetDatabase:    strings.TrimSpace(targetDatabase),
		TargetUser:        strings.TrimSpace(targetUser),
		TargetHost:        strings.TrimSpace(targetHost),
		TargetPort:        targetPort,
		TargetPasswordEnv: strings.TrimSpace(targetPasswordEnv),
		DatabaseSSH:       databaseSSH,
		DumpFlags:         strings.Fields(strings.TrimSpace(flagsRaw)),
	}, nil
}

func promptDatabaseSSHTarget(target ResolvedSyncTarget) (syncSSHTarget, error) {
	defaultTarget := defaultDatabaseSSHTarget(target)

	host, err := promptText("Database SSH host override (optional)", "", optionalValue, 0)
	if err != nil {
		return syncSSHTarget{}, err
	}
	if strings.TrimSpace(host) == "" {
		return defaultTarget, nil
	}

	username, err := promptText("Database SSH user", defaultTarget.Username, requiredValue, 0)
	if err != nil {
		return syncSSHTarget{}, err
	}

	port, err := promptPort("Database SSH port", strconv.Itoa(defaultTarget.Port))
	if err != nil {
		return syncSSHTarget{}, err
	}

	return syncSSHTarget{
		Host:     strings.TrimSpace(host),
		Username: strings.TrimSpace(username),
		Port:     port,
	}, nil
}

func defaultDatabaseSSHTarget(target ResolvedSyncTarget) syncSSHTarget {
	return syncSSHTarget{
		Host:     strings.TrimSpace(target.Host),
		Username: strings.TrimSpace(target.Username),
		Port:     target.Port,
	}
}

func promptPort(label, defaultValue string) (int, error) {
	value, err := promptText(label, defaultValue, requiredValue, 0)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid port %q", value)
	}

	return parsed, nil
}

func buildMySQLDumpArgs(config mysqlSyncConfig) []string {
	args := []string{
		fmt.Sprintf("--host=%s", config.SourceHost),
		fmt.Sprintf("--port=%d", config.SourcePort),
		fmt.Sprintf("--user=%s", config.SourceUser),
	}

	args = append(args, config.DumpFlags...)
	args = append(args, config.SourceDatabase)
	return args
}

func buildRemoteMySQLImportCommand(config mysqlSyncConfig, targetPassword string) string {
	return fmt.Sprintf(
		"MYSQL_PWD=%s mysql --host=%s --port=%d --user=%s %s",
		shellQuote(targetPassword),
		shellQuote(config.TargetHost),
		config.TargetPort,
		shellQuote(config.TargetUser),
		shellQuote(config.TargetDatabase),
	)
}

func buildDatabaseSSHArgs(target syncSSHTarget, remoteCommand string) []string {
	return []string{"-p", strconv.Itoa(target.Port), fmt.Sprintf("%s@%s", target.Username, target.Host), remoteCommand}
}

func shellQuote(value string) string {
	escaped := strings.ReplaceAll(value, "'", "'\\''")
	return "'" + escaped + "'"
}

func isSameDatabaseTarget(config mysqlSyncConfig) bool {
	if !isSameDatabaseHost(config.SourceHost, config.TargetHost, config.DatabaseSSH.Host) {
		return false
	}

	if config.TargetPort != config.SourcePort {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(config.SourceDatabase), strings.TrimSpace(config.TargetDatabase))
}

func isSameDatabaseHost(sourceHost string, targetHost string, targetSSHHost string) bool {
	source := strings.TrimSpace(sourceHost)
	target := strings.TrimSpace(targetHost)
	if source == "" || target == "" {
		return false
	}

	if !isLoopbackHost(target) {
		return strings.EqualFold(source, target)
	}

	return strings.EqualFold(source, strings.TrimSpace(targetSSHHost))
}

func isLoopbackHost(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	return normalized == "localhost" || normalized == "127.0.0.1" || normalized == "::1"
}
