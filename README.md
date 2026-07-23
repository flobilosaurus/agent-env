# agentenv

`agentenv` launches any AI coding agent with a project-specific identity by setting an isolated profile `HOME`.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/flobilosaurus/agent-env/main/install.sh | sh
```

Options:

```sh
AGENTENV_INSTALL_DIR=/usr/local/bin sh -c "$(curl -fsSL https://raw.githubusercontent.com/flobilosaurus/agent-env/main/install.sh)"
AGENTENV_VERSION=v0.1.0 sh -c "$(curl -fsSL https://raw.githubusercontent.com/flobilosaurus/agent-env/main/install.sh)"
```

## Build

```sh
mise install
mise run build
```

## Commands

```sh
agentenv run [--select] <agent> [args...]
agentenv wrap <agent>
agentenv doctor [agent]
```

Examples:

```sh
agentenv run pi --version
agentenv run --select pi
agentenv run claude
agentenv run codex --help
```

`run` is agent-agnostic. It resolves the real executable from `PATH` while skipping the agentenv wrapper bin directory to avoid recursion. On first use in an unmapped project, it opens a terminal profile selector/creator and stores the local project-to-profile mapping. Use `agentenv run --select <agent>` to force the selector even when a mapping already exists; arguments after `<agent>` are passed through unchanged.

`wrap <agent>` writes `$AGENTENV_HOME/bin/<agent>` and updates your shell startup file (`.zshrc`, `.bashrc`, `.profile`, Nushell `env.nu`, or fish `conf.d/agentenv.fish`) with an agentenv-managed block that puts that wrapper directory before real agent binaries on `PATH`. Restart your shell or source the updated file before running the agent command directly.

`doctor [agent]` checks config readability, project mapping, profile home paths, wrapper/PATH state, real-agent resolution, and when an agent is provided runs `/resolved/real-agent --version` as a light probe.

## Runtime files

- Config: `${AGENTENV_CONFIG_HOME:-user config dir}/agentenv/config.toml`
- Agentenv data root: `${AGENTENV_HOME:-$HOME/.local/share/agentenv}`
- Wrapper bin dir: `$AGENTENV_HOME/bin`
- Isolated profile HOME: `$AGENTENV_HOME/profiles/<profile>/home`

`AGENTENV_HOME` is agentenv's data root. It is not the same as the `HOME` value passed to agents.

## Security model v1

agentenv only sets `HOME` for the child process. It does not set XDG variables, manage secrets, or write repo-local mapping files. Profiles have names only.

## Development

```sh
mise run fmt
mise run test
mise run test:e2e
mise run build
```

E2E tests build `agentenv`, use fake agents, and isolate config/data directories with `AGENTENV_CONFIG_HOME` and `AGENTENV_HOME`.
