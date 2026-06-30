## Intercube CLI
This command line interface allows you to connect, manage, and synchronize data from your Intercube cubes (servers).

### Installing & updating

Install (or update) with the Go toolchain:

```bash
go install github.com/intercube/cli/cmd/intercube@latest
```

Once installed, you can update in place without remembering that command:

```bash
intercube self-update            # update to the latest version
intercube self-update v1.0.19    # pin or roll back to a specific version
intercube self-update --force    # reinstall even if already up to date
```

`self-update` shells out to `go install`, so it requires the Go toolchain on your
PATH. It installs to your `$GOBIN` (or `$GOPATH/bin`); if a different copy of
`intercube` appears earlier on your PATH, the command warns you so you can fix it.

Check your current version any time:

```bash
intercube version
intercube --version
```

### Onboarding

Use the onboarding command to set up your CLI config interactively:

```bash
intercube onboarding
```

The wizard can help you:
- configure login defaults (`username`, `password`, `scope`, `auth_method`, `instance_url`)
- optionally configure file path mappings for `intercube sync`
- verify local prerequisites such as Boundary CLI and `rsync`

After onboarding, use:

```bash
intercube ssh
```

If required settings are missing (or the config file does not exist), the CLI
will prompt only when needed and save values automatically:
- `intercube ssh` prompts for required login settings
- `intercube sync` prompts for missing file path mappings
- `intercube map --interactive` prompts to create mappings when none exist

`intercube login` is kept as a deprecated alias and prints a warning to use `intercube ssh`.

### Sync

Use sync from a source environment host:

```bash
intercube sync
intercube sync staging.example.com
intercube sync --files
intercube sync --database
intercube sync --dry-run
```

Behavior:
- always fetches current site inventory at runtime
- interactive target selection when no argument is passed
- argument auto-resolves against site ID/domain/server/user when possible
- stores only file path mappings in config (`sync.files.items`)
- database details are requested interactively for each run (not persisted)
- database sync uses the selected target server by default, with an optional
  database SSH host override for clustered setups where files and MySQL live on
  separate servers

### Context-aware defaults

The CLI resolves defaults using context detection and a fixed precedence.

Context detection order:
1. `--context` / `INTERCUBE_CONTEXT` (`pipeline`, `server`, `repository`, `global`)
2. `CI=true` -> pipeline mode
3. server config at `/etc/intercube.yaml` -> server mode
4. nearest repo containing `.intercube.yaml` -> repository mode
5. fallback -> global mode

Resolution precedence:
1. command flags
2. environment variables
3. active context config
4. user defaults

Supported default keys:

```yaml
context:
  org_id: org_xxx
  site_id: "58"
  server_id: "42"

behavior:
  non_interactive: false
```

Config scopes:
- user: `~/.intercube.yaml`
- repository: `<repo>/.intercube.yaml`
- server: `~/.intercube.yaml` (same user-level config, used when `--context server` is selected)

Environment overrides:
- `INTERCUBE_ORG_ID` (preferred) and `INTERCUBE_ORGANIZATION_ID` (legacy)
- `INTERCUBE_SITE_ID`
- `INTERCUBE_SERVER_ID`
- `INTERCUBE_NON_INTERACTIVE`

In non-interactive mode, commands fail instead of prompting when required values are missing.
