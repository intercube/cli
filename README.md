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
- `intercube map --create-origin-dir` / `intercube map -c` creates missing origin directories before mapping

`intercube login` is kept as a deprecated alias and prints a warning to use `intercube ssh`.

### Set up a repository

Run the setup wizard from a GitHub repository to create its first Intercube server and site:

```bash
intercube setup
```

The wizard detects the application, recommends a runtime and server plan, suggests an available `mycube.dev` domain,
and shows the exact monthly price before asking for confirmation. Press Enter to accept a suggested value. You may use
your own domain instead, but custom `mycube.dev` names cannot be claimed; Intercube keeps the generated managed domain
as an alias.

If GitHub is not linked yet, the wizard opens GitHub's device authorization flow. If the Intercube GitHub App still
needs repository access, it opens the installation page and then continues the setup. No personal access token is
required.

Setup is asynchronous. The CLI saves the intent and stable server/site identifiers in the repository's
`.intercube.yaml`, polls until it is ready, and resumes an interrupted run the next time you invoke the command. A
failed setup is never retried automatically; rerun `intercube setup` and confirm the retry.

One repository can contain multiple environments, each backed by its own server, for example:

```bash
intercube setup --environment production --branch master
intercube setup --environment development --branch develop
```

On a configured repository, `intercube setup` reports the existing environment and offers to configure another one.
The committed repository config contains deployment metadata only, never credentials. User authentication and private
configuration remain user-scoped.

Platform administrators may additionally create a site on an existing ready server with `--server` and may use
`--no-charge`. Regular organization users can only create a paid new server with its initial site.

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
- stores file path mappings in config (`sync.files.items`)
- database connection details are requested interactively for each run (not persisted)
- database sync uses the selected target server by default, with an optional
  database SSH host override for clustered setups where files and MySQL live on
  separate servers

Single-server environments do not need extra sync configuration. For clustered
targets, add the database SSH defaults to `.intercube.yaml`:

```yaml
sync:
  files:
    items:
      - source: /var/www/site/
        target: /var/www/site/
  database:
    target_ssh:
      host: production-sql.example.com
      user: deploy
      port: 22
```

When `sync.database.target_ssh.host` is set, `intercube sync --database`
pre-fills the database SSH prompt with that host. The `user` and `port` values
are optional; when omitted, the selected target server's SSH user and port are
used.

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
