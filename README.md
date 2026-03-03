## Intercube CLI
This command line interface allows you to connect, manage, and synchronize data from your Intercube cubes (servers).

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
