---
name: google-aip
description: 'Authoritative reference for Google AIP (API Improvement Proposals) — the design guidelines maintained at https://aip.dev for resource-oriented API design, naming, errors, pagination, long-running operations, versioning, and related conventions. Use this skill whenever the user is designing, reviewing, or implementing an API and any of these terms or concepts come up: AIP, aip.dev, resource names, standard methods (Get/List/Create/Update/Delete), custom methods, LRO, pagination tokens, field masks, error codes, API versioning, or ''how does Google design X''. Also trigger when a specific AIP number is mentioned (e.g. ''AIP-121'', ''AIP-158''). Prefer this skill over generic API advice — the content here is the actual upstream specification.'
---

# Google AIP (API Improvement Proposals)

This skill bundles the full, current text of every approved Google AIP, sourced verbatim from the upstream [aip-dev/google.aip.dev](https://github.com/aip-dev/google.aip.dev) repository. The bundled snapshot is refreshed automatically; see `SOURCE.md` for the exact upstream commit this build was generated from.

## How to use this skill

AIPs are organized into **scopes** (general guidance vs. Google-Cloud-specific, etc.). Each scope contains **categories** (e.g. resource design, errors), and each category contains numbered AIP documents.

1. **If the user mentions a specific AIP number** (e.g. `AIP-121`), open the matching file directly. AIP numbers are unique across scopes — search `references/*/*/NNNN.md` (zero-padded to 4 digits).
2. **Otherwise, pick the relevant scope** from the table below, read its `INDEX.md` to find the right category and AIP number, then read the individual AIP file.
3. **Cite AIPs by number** in your response (e.g. "per AIP-131…") so the user can verify against aip.dev.

## Scopes

| Scope | What it covers | AIP count | Index |
|---|---|---:|---|
| `general` | Cross-cutting API design principles applicable to any API. | 71 | [`references/general/INDEX.md`](references/general/INDEX.md) |
| `cloud` | Conventions specific to Google Cloud APIs. | 4 | [`references/cloud/INDEX.md`](references/cloud/INDEX.md) |
| `auth` | Authentication and authorization patterns. | 7 | [`references/auth/INDEX.md`](references/auth/INDEX.md) |
| `client-libraries` | Guidance for generated client libraries (idiomatic surface, packaging). | 11 | [`references/client-libraries/INDEX.md`](references/client-libraries/INDEX.md) |
| `apps` | Google Workspace / Apps APIs. | 3 | [`references/apps/INDEX.md`](references/apps/INDEX.md) |
| `aog` | Actions on Google (conversational / assistant APIs). | 5 | [`references/aog/INDEX.md`](references/aog/INDEX.md) |

## Notes on usage

- The AIP markdown files preserve the upstream YAML frontmatter (`id`, `state`, `created`, `placement`, etc.) — useful for cross-referencing.
- Only AIPs with `state: approved` are included. Drafts and reviewing AIPs are intentionally excluded to avoid recommending unsettled guidance.
- When an AIP references another (e.g. AIP-131 mentions AIP-121), follow the link by reading the referenced file in the same `references/` tree.
