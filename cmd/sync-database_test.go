package cmd

import (
	"reflect"
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

func TestBuildDatabaseSSHArgsUsesClusterOverride(t *testing.T) {
	target := syncSSHTarget{
		Host:     "sql.example.com",
		Username: "dbadmin",
		Port:     2200,
	}

	got := buildDatabaseSSHArgs(target, "mysql --host=127.0.0.1 shop")
	want := []string{"-p", "2200", "dbadmin@sql.example.com", "mysql --host=127.0.0.1 shop"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ssh args: got %#v want %#v", got, want)
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
