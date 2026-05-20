# Google AIP (API Improvement Proposals)

This file gives any AGENTS.md-aware agent (OpenAI Codex CLI, etc.) the same Google AIP reference that the bundled Claude Code skill provides. It bundles the full, current text of every approved AIP, sourced verbatim from [aip-dev/google.aip.dev](https://github.com/aip-dev/google.aip.dev) and refreshed daily by CI. See `skills/google-aip/SOURCE.md` for the exact upstream commit this build was generated from.

## When to consult this reference

Use these documents whenever the user is designing, reviewing, or implementing an API and mentions any of: AIP, aip.dev, resource-oriented design, resource names, standard methods (Get/List/Create/Update/Delete), custom methods, long-running operations (LRO), pagination tokens, field masks, error codes, API versioning, or a specific `AIP-NNN` number. Prefer this material over generic API advice — it is the actual upstream specification.

## How to navigate

AIPs are organized into **scopes** (general guidance vs. Google-Cloud-specific, etc.). Each scope contains **categories** (e.g. resource design, errors), and each category contains numbered AIP documents.

1. **If the user names a specific AIP number** (e.g. `AIP-121`), open the matching file directly under `skills/google-aip/references/<scope>/<category>/<NNNN>.md` (zero-padded to 4 digits).
2. **Otherwise, pick the relevant scope** from the table below, read its `INDEX.md` to find the right category and AIP number, then read the individual AIP file.
3. **Cite AIPs by number** in your response (e.g. "per AIP-131…") so the user can verify against aip.dev.

## Scopes

| Scope | What it covers | AIP count | Index |
|---|---|---:|---|
| `general` | Cross-cutting API design principles applicable to any API. | 70 | [`skills/google-aip/references/general/INDEX.md`](skills/google-aip/references/general/INDEX.md) |
| `cloud` | Conventions specific to Google Cloud APIs. | 4 | [`skills/google-aip/references/cloud/INDEX.md`](skills/google-aip/references/cloud/INDEX.md) |
| `auth` | Authentication and authorization patterns. | 7 | [`skills/google-aip/references/auth/INDEX.md`](skills/google-aip/references/auth/INDEX.md) |
| `client-libraries` | Guidance for generated client libraries (idiomatic surface, packaging). | 11 | [`skills/google-aip/references/client-libraries/INDEX.md`](skills/google-aip/references/client-libraries/INDEX.md) |
| `apps` | Google Workspace / Apps APIs. | 3 | [`skills/google-aip/references/apps/INDEX.md`](skills/google-aip/references/apps/INDEX.md) |
| `aog` | Actions on Google (conversational / assistant APIs). | 5 | [`skills/google-aip/references/aog/INDEX.md`](skills/google-aip/references/aog/INDEX.md) |

## Notes

- The AIP markdown files preserve the upstream YAML frontmatter (`id`, `state`, `created`, `placement`, …) — useful for cross-referencing.
- Only AIPs with `state: approved` are included. Drafts and reviewing AIPs are intentionally excluded so unsettled guidance is never recommended.
- When an AIP references another (e.g. AIP-131 mentions AIP-121), follow the link by reading the referenced file in the same reference tree.
