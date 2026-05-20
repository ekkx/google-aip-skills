# google-aip

A Claude Code plugin that bundles the full text of every approved
[Google AIP](https://google.aip.dev) (API Improvement Proposal). Claude can
reference them while designing or reviewing APIs — completely offline, no
network calls at skill-use time.

## Install

This repository is both a [Claude Code marketplace](https://docs.claude.com/en/docs/claude-code/plugins)
and the plugin it serves. From inside Claude Code:

```text
/plugin marketplace add ekkx/google-aip-skills
/plugin install google-aip@google-aip-skills
```

Subsequent updates land automatically — the marketplace tracks this repository's
default branch, which is refreshed daily by CI from the upstream AIP spec.

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
│   ├── marketplace.json   # this repo is a marketplace
│   └── plugin.json        # ... and also the plugin it ships
├── skills/google-aip/     # the skill payload (regenerated)
├── cmd/build/main.go      # the importer
├── .github/workflows/sync.yml
├── go.mod / go.sum
└── README.md
```

The `.claude/settings.json` checked into this repo only enables the
`skill-creator` plugin for *contributors* working in this checkout — it has
no effect on plugin consumers.

## Licensing

The build tooling in this repository is MIT-licensed (see `LICENSE`). The AIP
texts under `skills/google-aip/references/` are copies from
[`aip-dev/google.aip.dev`](https://github.com/aip-dev/google.aip.dev) and
remain under that project's license.
