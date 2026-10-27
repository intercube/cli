package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type mysqlSyncConfig struct {
	TargetServerAddress string
	TargetServerName    string
	SourceDatabase      string
	SourceUser          string
	SourceHost          string
	SourcePort          int
	SourcePasswordEnv   string
	TargetDatabase      string
	TargetUser          string
	TargetHost          string
	TargetPort          int
	TargetPasswordEnv   string
	DatabaseSSH         syncSSHTarget
	DumpFlags           []string
}

type syncSSHTarget struct {
	Host     string
	Username string
	Port     int
}

func runDatabaseSync(cmd *cobra.Command, target ResolvedSyncTarget, settings *SyncSettings, dryRun bool, autoApprove bool) error {
	if isNonInteractiveMode() {
		return fmt.Errorf("database sync requires interactive prompts in current implementation; run with an interactive terminal")
	}

	if err := ensureCommandAvailable("mysqldump"); err != nil {
		return err
	}
	if err := ensureCommandAvailable("mysql"); err != nil {
		return err
	}
	if err := ensureCommandAvailable("ssh"); err != nil {
		return err
	}

	databaseConfig, err := promptMySQLSyncConfig(target, settings)
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
	connection := ResolvedSyncTarget{Host: databaseConfig.DatabaseSSH.Host, Username: databaseConfig.DatabaseSSH.Username, Port: databaseConfig.DatabaseSSH.Port}
	socket, err := gatewayControlPath(connection)
	if err != nil {
		return err
	}
	if !dryRun {
		if err := ensureGatewayMaster(cmd, connection, socket); err != nil {
			return err
		}
	}
	fmt.Printf("Running: mysqldump %s | mysql through the authorized SSH tunnel (credentials hidden)\n", strings.Join(dumpArgs, " "))
	if dryRun {
		return nil
	}
	port, closeTunnel, err := openDatabaseTunnel(cmd, connection, socket, databaseConfig.TargetHost, databaseConfig.TargetPort)
	if err != nil {
		return err
	}
	defer closeTunnel()
	importCommand := exec.CommandContext(cmd.Context(), "mysql", buildLocalMySQLImportArgs(databaseConfig, port)...)
	importCommand.Env = append(os.Environ(), "MYSQL_PWD="+targetPassword)
	importCommand.Stdout = os.Stdout
	importCommand.Stderr = os.Stderr
	stdinPipe, err := importCommand.StdinPipe()
	if err != nil {
		return err
	}
	if err := importCommand.Start(); err != nil {
		return err
	}

	dumpCommand := exec.CommandContext(cmd.Context(), "mysqldump", dumpArgs...)
	dumpCommand.Stdout = stdinPipe
	dumpCommand.Stderr = os.Stderr
	dumpCommand.Env = append(os.Environ(), "MYSQL_PWD="+sourcePassword)

	dumpErr := dumpCommand.Run()
	_ = stdinPipe.Close()
	importErr := importCommand.Wait()

	if dumpErr != nil {
		return dumpErr
	}

	if importErr != nil {
		return importErr
	}

	return nil
}

func ensureCommandAvailable(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("required command %q not found", name)
	}

	return nil
}

func promptMySQLSyncConfig(target ResolvedSyncTarget, settings *SyncSettings) (mysqlSyncConfig, error) {
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

	databaseSSH, err := promptDatabaseSSHTarget(target, settings)
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
		TargetServerName:  target.ServerName, TargetServerAddress: target.ServerAddress,
		DumpFlags: strings.Fields(strings.TrimSpace(flagsRaw)),
	}, nil
}

func promptDatabaseSSHTarget(target ResolvedSyncTarget, settings *SyncSettings) (syncSSHTarget, error) {
	configured, hasOverride := configuredDatabaseSSHTarget(target, settings)
	expected := defaultDatabaseSSHTarget(target)
	if hasOverride && configured != expected {
		return syncSSHTarget{}, fmt.Errorf("database SSH overrides must match the authorized gateway target; select the appropriate site instead")
	}
	return expected, nil
}

func configuredDatabaseSSHTarget(target ResolvedSyncTarget, settings *SyncSettings) (syncSSHTarget, bool) {
	defaultTarget := defaultDatabaseSSHTarget(target)
	if settings == nil {
		return defaultTarget, false
	}

	configured := settings.Database.TargetSSH
	host := strings.TrimSpace(configured.Host)
	if host == "" {
		return defaultTarget, false
	}

	username := strings.TrimSpace(configured.User)
	if username == "" {
		username = defaultTarget.Username
	}

	port := configured.Port
	if port <= 0 {
		port = defaultTarget.Port
	}

	return syncSSHTarget{
		Host:     host,
		Username: username,
		Port:     port,
	}, true
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

// Target credentials stay in the local mysql environment, outside SSH exec requests and shell recordings.
func buildLocalMySQLImportArgs(config mysqlSyncConfig, port int) []string {
	return []string{"--protocol=TCP", "--host=127.0.0.1", "--port=" + strconv.Itoa(port), "--user=" + config.TargetUser, "--", config.TargetDatabase}
}

func openDatabaseTunnel(cmd *cobra.Command, target ResolvedSyncTarget, socket, host string, port int) (int, func(), error) {
	if host == "" || strings.ContainsAny(host, " \t\r\n[]") || port < 1 || port > 65535 {
		return 0, nil, fmt.Errorf("invalid target database host or port")
	}
	destination := host
	if strings.Contains(host, ":") {
		destination = "[" + host + "]"
	}
	for attempt := 0; attempt < 3; attempt++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return 0, nil, err
		}
		localPort := listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()
		forward := fmt.Sprintf("127.0.0.1:%d:%s:%d", localPort, destination, port)
		args := []string{"-S", socket, "-O", "forward", "-L", forward, "-p", strconv.Itoa(target.Port), "-l", target.Username, "--", target.Host}
		if err := exec.CommandContext(cmd.Context(), "ssh", args...).Run(); err != nil {
			continue
		}
		cleanup := func() {
			args[3] = "cancel"
			_ = exec.Command("ssh", args...).Run()
		}
		return localPort, cleanup, nil
	}
	return 0, nil, fmt.Errorf("unable to open the MySQL tunnel; reconnect and try again")
}

func shellQuote(value string) string {
	escaped := strings.ReplaceAll(value, "'", "'\\''")
	return "'" + escaped + "'"
}

func isSameDatabaseTarget(config mysqlSyncConfig) bool {
	serverName := config.TargetServerName
	if serverName == "" {
		serverName = config.DatabaseSSH.Host
	}
	if !isSameDatabaseHost(config.SourceHost, config.TargetHost, serverName) && !isSameDatabaseHost(config.SourceHost, config.TargetHost, config.TargetServerAddress) {
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
