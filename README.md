# thenie_v2

An **exact, byte-for-byte mirror** of the deployed Thenie Healthy Catering
ordering mockup, plus the documentation reconstructed from it and a runbook for
deploying it to the existing Ubuntu/Nginx server.

- **Source of truth:** <https://thenie-catering-order.netlify.app>
- **Captured:** 2026-08-22
- **Mirror:** [`site/index.html`](site/index.html) — SHA-256 `9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49`

## The one rule for `site/`

`site/index.html` is a **verbatim capture**. Nothing in it has been added,
changed, reformatted, minified, prettified or removed — not one byte. Do not
"tidy" it. If the upstream page changes, re-capture it and record the new hash
in [`docs/07-fidelity-and-verification.md`](docs/07-fidelity-and-verification.md);
never hand-edit the mirror to match.

Every observation in `docs/` was read *out of* that file. The documentation
describes the mockup; it does not modify it.

## Layout

```
site/index.html    the mirror — 4,615,031 bytes, self-contained, zero external requests
docs/              documentation reconstructed from the mirror (also an Obsidian vault)
```

## Documentation

Start at [`docs/00-index.md`](docs/00-index.md).

## Opening the docs in Obsidian

`docs/` is a plain folder of Markdown with `[[wikilinks]]`, so it *is* a vault.
In Obsidian: **Open folder as vault** → select `/home/dev/projects/thenie_v2/docs`.
No account, no sync subscription, no login required.

## Running it locally

```bash
python3 -m http.server 8080 --directory /home/dev/projects/thenie_v2/site
```

Then open <http://localhost:8080>.

## Deploying

See [`docs/13-production-deployment-runbook.md`](docs/13-production-deployment-runbook.md).
