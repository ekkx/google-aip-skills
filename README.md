# google-aip

> **The problem.** Ask an LLM to design a REST or gRPC API and it'll happily
> invent its own conventions — ignoring AIP-121's resource-oriented patterns,
> making up error shapes, skipping pagination tokens. The
> [Google AIP spec](https://google.aip.dev) is public, but it lives on an HTML
> site that agents can't reliably cite from, and the training cutoff already
> missed the latest revisions.
>
> **The fix.** Ship the full, current AIP text *into the agent's context* —
> offline, structured, always current.

The same markdown tree is exposed as three entry points so whichever agent
you happen to use can consult the actual upstream spec without a network
call:

- **Claude Code plugin**
- **Codex `AGENTS.md`**
- **Cursor rule**

Other agents (Cline, Continue, plain `@file` mentions, etc.) can read the same
markdown directly. CI re-syncs daily from
[`aip-dev/google.aip.dev`](https://github.com/aip-dev/google.aip.dev) so the
bundle never lags the upstream spec.

## Install

The same `references/` tree powers every agent. Each agent just gets its own
entry point file.

### Claude Code

This repository is both a [Claude Code marketplace](https://docs.claude.com/en/docs/claude-code/plugins)
and the plugin it serves:

```text
/plugin marketplace add ekkx/google-aip-skills
/plugin install google-aip@google-aip-skills
```

Updates land automatically — the marketplace tracks this repository's default
branch, which is refreshed daily by CI from the upstream AIP spec.

### Codex CLI (and any AGENTS.md-aware agent)

Add this repository to your project as a git submodule (or vendor it however
you prefer), then point `AGENTS.md` at the bundled one:

```sh
git submodule add https://github.com/ekkx/google-aip-skills vendor/google-aip-skills
ln -s vendor/google-aip-skills/AGENTS.md AGENTS.md      # or copy it
```

Codex picks up `AGENTS.md` hierarchically — if you'd rather scope it to a
subdirectory, drop it there instead of the repo root.

### Cursor

Same submodule trick, symlinked into Cursor's rules directory:

```sh
git submodule add https://github.com/ekkx/google-aip-skills vendor/google-aip-skills
mkdir -p .cursor/rules
ln -s ../../vendor/google-aip-skills/.cursor/rules/google-aip.mdc .cursor/rules/google-aip.mdc
```

The rule uses `alwaysApply: false` and a description, so Cursor only attaches
it when the conversation actually involves API design.

### Other agents (Cline, Continue, etc.)

Anything that can read Markdown from your repository works. Point the agent at
`vendor/google-aip-skills/skills/google-aip/SKILL.md` (or `AGENTS.md`) and let
it follow the links from there.

## What you get

Once installed, the `google-aip` skill auto-loads whenever Claude detects an
API-design context (resource names, standard methods, pagination, errors, LRO,
versioning, or a specific `AIP-NNN` reference).

Skill contents:

```
skills/google-aip/
├── SKILL.md           # trigger description + scope navigation
├── SOURCE.md          # upstream commit SHA + import timestamp
└── references/
    ├── general/INDEX.md
    ├── general/resource-design/0121.md
    ├── general/resource-design/0122.md
    ├── ...
    ├── cloud/INDEX.md
    ├── auth/INDEX.md
    └── ...
```

Each AIP file is a verbatim copy from
[`aip-dev/google.aip.dev`](https://github.com/aip-dev/google.aip.dev),
including its YAML frontmatter (`id`, `state`, `created`, `placement`, …).

Only AIPs with `state: approved` are included — drafts are intentionally
excluded so Claude does not recommend unsettled guidance.

## How freshness is guaranteed

[`.github/workflows/sync.yml`](.github/workflows/sync.yml) runs daily, rebuilds
`skills/google-aip/` from the upstream `master`, and commits the result iff
anything changed. The upstream commit SHA used for each build is recorded in
`skills/google-aip/SOURCE.md`, so every snapshot is traceable.

## How AIPs are organized

The upstream repository defines **scopes** (`aip/general/`, `aip/cloud/`,
`aip/auth/`, …) and, inside each scope, **categories** declared in
`scope.yaml` (e.g. `resource-design`, `errors`, `design-patterns`).

The build mirrors that structure verbatim:

- `references/<scope>/<category>/<NNNN>.md` — the AIP itself
- `references/<scope>/INDEX.md` — generated table of contents for that scope
- `SKILL.md` — top-level scope selector

If an AIP has no `placement.category` (some scopes have none declared), it
falls into an auto-created `misc` bucket so nothing is silently dropped.

## Building locally

Requirements: Go 1.24+ and `git`.

```sh
# Clone upstream, regenerate the skill in place.
go run ./cmd/build

# Use an existing local checkout of google.aip.dev instead of cloning:
go run ./cmd/build -source /path/to/google.aip.dev

# Custom output directory:
go run ./cmd/build -out /tmp/google-aip-skill
```

The build is fully deterministic for a given upstream commit — re-running it
without upstream changes produces an identical tree.

## Repository layout

```
google-aip-skills/
├── .claude-plugin/
│   ├── marketplace.json     # this repo is a marketplace
│   └── plugin.json          # ... and also the plugin it ships
├── skills/google-aip/       # the shared payload (SKILL.md + references/)
├── AGENTS.md                # entry point for Codex / generic agents
├── .cursor/rules/
│   └── google-aip.mdc       # entry point for Cursor
├── cmd/build/main.go        # the importer
├── .github/workflows/sync.yml
├── go.mod / go.sum
└── README.md
```

## Licensing

The build tooling in this repository is MIT-licensed (see `LICENSE`). The AIP
texts under `skills/google-aip/references/` are copies from
[`aip-dev/google.aip.dev`](https://github.com/aip-dev/google.aip.dev) and
remain under that project's license.
