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

Onboarding verifies OpenSSH, checks for optional rsync support and signs in through
Clerk. Existing Boundary login/password settings are removed when the user
configuration is rewritten.

```bash
intercube ssh shop.example.com           # fixed site user
intercube ssh --root web.example.com     # explicit platform administrators only
intercube ssh shop.example.com --connect-only
intercube ssh shop.example.com -L 127.0.0.1:3307:127.0.0.1:3306 --tunnel
```

The target list covers all of your organization memberships. Customers always
connect as the selected site's Linux user; `--ssh_username` only filters
already-authorized accounts. Warpgate provides browser approval and server-side
SSH credentials. Personal SSH keys and the Boundary binary are unnecessary.
`--sites` remains accepted because site metadata is now always available.
Reusable SSH and sync connections require OpenSSH with `ControlMaster` and
`ControlPersist` multiplexing support.

`intercube login` remains a deprecated alias for `intercube ssh`. Missing Clerk
authentication is reported with instructions to run `intercube auth login`.
Non-interactive SSH uses an existing approved control connection; establish one
with `--connect-only` first. Missing file mappings still prompt in interactive
sync, while `intercube map --interactive` manages local mappings.

Temporary developers may keep using assigned site keys with direct SSH. Those
connections bypass gateway recording and revocation.

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

To connect a repository to a site that already exists but has no pipeline project, use the searchable site selector:

```bash
intercube setup --existing-site
intercube setup -s -b develop
intercube setup -s -b develop -d wordpress
```

The selected site is enabled for CI/CD, connected to the detected workflow, and synchronized to GoCD. GoCD starts the
first deployment and follows later commits on the selected branch. Sites that already have a pipeline project are not
changed; manage those projects in Dashboard instead. Existing sites are deliberately selector-only, so there is no
direct numeric `--site` flag.

Setup supports `-e` for `--environment`, `-b` for `--branch`, `-d` for `--directory`, `-p` for `--plan`, `-o` for
`--organization`, `-s` for `--existing-site`, and `-y` for `--yes`.

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
- database sync uses the selected site's authorized gateway connection; the
  prompted MySQL host and port are reached from that site's server

File mappings can be saved in `.intercube.yaml`:

```yaml
sync:
  files:
    items:
      - source: /var/www/site/
        target: /var/www/site/
```

Remove old `sync.database.target_ssh` overrides before using the gateway flow.
For clustered databases, enter the database address reachable from the selected
site's server in the MySQL host prompt. To use another server as the SSH endpoint,
select an authorized site on that server.

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

### Gateway transport for synchronization

File and database synchronization use the selected site's Warpgate target and a
shared OpenSSH control connection. Database imports use a local `mysql` client
through an SSH tunnel; install both `mysql` and `mysqldump`. Passwords are read
from the chosen environment variables and are excluded from command arguments
and printed plans. Legacy arbitrary SSH host/user overrides are rejected; select
the appropriate authorized site instead. Native SSH tunnels also support normal
database clients and prepared statements.
