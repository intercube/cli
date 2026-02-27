## Intercube CLI
This command line interface allows you to connect, manage, and synchronize data from your Intercube cubes (servers).

### Onboarding

Use the onboarding command to set up your CLI config interactively:

```bash
intercube onboarding
```

The wizard can help you:
- configure login defaults (`username`, `password`, `scope`, `auth_method`, `instance_url`)
- optionally set sync defaults (`remote_user`, `file_syncing.from_server`, `file_syncing.path`)
- verify local prerequisites such as Boundary CLI and the Intercube sync helper

After onboarding, use:

```bash
intercube login
```

If required settings are missing (or the config file does not exist), the CLI
will prompt only when needed and save values automatically:
- `intercube login` prompts for required login settings
- `intercube sync files ...` prompts for missing sync defaults
- `intercube map` prompts to create mappings when none exist
