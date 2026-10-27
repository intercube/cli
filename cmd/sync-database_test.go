package cmd

import (
	"strings"
	"testing"

	"github.com/intercube/cli/util"
)

func TestDefaultDatabaseSSHTargetUsesSelectedSyncTarget(t *testing.T) {
	target := ResolvedSyncTarget{
		Host:     "file.example.com",
		Username: "siteuser",
		Port:     2222,
	}

	got := defaultDatabaseSSHTarget(target)
	want := syncSSHTarget{Host: "file.example.com", Username: "siteuser", Port: 2222}

	if got != want {
		t.Fatalf("unexpected default database SSH target: got %+v want %+v", got, want)
	}
}

func TestConfiguredDatabaseSSHTargetOverridesSelectedSyncTarget(t *testing.T) {
	target := ResolvedSyncTarget{
		Host:     "file.example.com",
		Username: "siteuser",
		Port:     2222,
	}
	settings := SyncSettings{
		Database: util.SyncDatabase{
			TargetSSH: util.SyncDatabaseTargetSSH{
				Host: "sql.example.com",
				User: "dbadmin",
				Port: 2200,
			},
		},
	}

	got, ok := configuredDatabaseSSHTarget(target, &settings)
	want := syncSSHTarget{Host: "sql.example.com", Username: "dbadmin", Port: 2200}

	if !ok {
		t.Fatalf("expected configured database SSH target")
	}
	if got != want {
		t.Fatalf("unexpected configured database SSH target: got %+v want %+v", got, want)
	}
}

func TestConfiguredDatabaseSSHTargetFallsBackToSelectedUserAndPort(t *testing.T) {
	target := ResolvedSyncTarget{
		Host:     "file.example.com",
		Username: "siteuser",
		Port:     2222,
	}
	settings := SyncSettings{
		Database: util.SyncDatabase{
			TargetSSH: util.SyncDatabaseTargetSSH{
				Host: "sql.example.com",
			},
		},
	}

	got, ok := configuredDatabaseSSHTarget(target, &settings)
	want := syncSSHTarget{Host: "sql.example.com", Username: "siteuser", Port: 2222}

	if !ok {
		t.Fatalf("expected configured database SSH target")
	}
	if got != want {
		t.Fatalf("unexpected configured database SSH target: got %+v want %+v", got, want)
	}
}

func TestConfiguredDatabaseSSHTargetWithoutHostUsesSelectedSyncTarget(t *testing.T) {
	target := ResolvedSyncTarget{
		Host:     "file.example.com",
		Username: "siteuser",
		Port:     2222,
	}
	settings := SyncSettings{
		Database: util.SyncDatabase{
			TargetSSH: util.SyncDatabaseTargetSSH{
				User: "dbadmin",
				Port: 2200,
			},
		},
	}

	got, ok := configuredDatabaseSSHTarget(target, &settings)
	want := syncSSHTarget{Host: "file.example.com", Username: "siteuser", Port: 2222}

	if ok {
		t.Fatalf("expected missing host to disable configured database SSH target")
	}
	if got != want {
		t.Fatalf("unexpected default database SSH target: got %+v want %+v", got, want)
	}
}

func TestMySQLImportUsesLocalTunnelWithoutPasswordInArguments(t *testing.T) {
	got := buildLocalMySQLImportArgs(mysqlSyncConfig{TargetDatabase: "shop", TargetUser: "site_db"}, 33071)
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "--protocol=TCP --host=127.0.0.1 --port=33071") || !strings.Contains(joined, "--user=site_db -- shop") || strings.Contains(joined, "MYSQL_PWD") {
		t.Fatalf("unexpected tunnel import arguments: %#v", got)
	}
}

func TestIsSameDatabaseTargetUsesDatabaseSSHHost(t *testing.T) {
	config := mysqlSyncConfig{
		SourceDatabase: "shop",
		SourceHost:     "file.example.com",
		SourcePort:     3306,
		TargetDatabase: "shop",
		TargetHost:     "127.0.0.1",
		TargetPort:     3306,
		DatabaseSSH:    syncSSHTarget{Host: "file.example.com", Username: "siteuser", Port: 22},
	}

	if !isSameDatabaseTarget(config) {
		t.Fatalf("expected loopback target to use database SSH host for same-target detection")
	}

	config.DatabaseSSH.Host = "target-file.example.com"
	if isSameDatabaseTarget(config) {
		t.Fatalf("expected different database SSH host to avoid same-target match for loopback target")
	}
}

func TestIsSameDatabaseTargetUsesExplicitTargetMySQLHost(t *testing.T) {
	config := mysqlSyncConfig{
		SourceDatabase: "shop",
		SourceHost:     "sql.example.com",
		SourcePort:     3306,
		TargetDatabase: "shop",
		TargetHost:     "sql.example.com",
		TargetPort:     3306,
		DatabaseSSH:    syncSSHTarget{Host: "file.example.com", Username: "siteuser", Port: 22},
	}

	if !isSameDatabaseTarget(config) {
		t.Fatalf("expected explicit target MySQL host to drive same-target detection")
	}
}

func TestSameDatabaseGuardUsesManagedServerInsteadOfGateway(t *testing.T) {
	config := mysqlSyncConfig{SourceHost: "managed.example.com", SourcePort: 3306, SourceDatabase: "shop", TargetHost: "127.0.0.1", TargetPort: 3306, TargetDatabase: "shop", TargetServerName: "managed.example.com", DatabaseSSH: syncSSHTarget{Host: "access.example.com"}}
	if !isSameDatabaseTarget(config) {
		t.Fatal("the gateway must not hide an identical source and destination database")
	}
}
