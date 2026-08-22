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

**Additions go in `site/overlays/`, never in the mirror.** `scripts/build-site.sh`
stitches mirror + overlays into `dist/index.html`, and that built file is what
gets deployed. Today there is one overlay: the floating WhatsApp button
([`docs/14-whatsapp-fab.md`](docs/14-whatsapp-fab.md)).

## Layout

```
site/index.html    the mirror — 4,615,031 bytes, self-contained, zero external requests
site/overlays/     our additions, injected at build time (WhatsApp button)
dist/index.html    build output — what the server serves (git-ignored)
docs/              documentation reconstructed from the mirror (also an Obsidian vault)
docs/screenshots/  rendered evidence — home, menu, order, catering kantor
scripts/           build-site.sh — mirror + overlays → dist/
                   verify-mirror.sh — re-prove the capture against disk and upstream
tests/             pricing-engine tests, run against code extracted from the mirror
```

## Tests

```bash
node --test tests/
```

31 tests, no dependencies. They extract the real `analyze()` out of
`site/index.html` at run time and assert the documented pricing rules against
it — see [`tests/README.md`](tests/README.md).

## Documentation

Start at [`docs/00-index.md`](docs/00-index.md).

## Opening the docs in Obsidian

`docs/` is a plain folder of Markdown with `[[wikilinks]]`, so it *is* a vault.
In Obsidian: **Open folder as vault** → select `/home/dev/projects/thenie_v2/docs`.
No account, no sync subscription, no login required.

## Building

```bash
./scripts/build-site.sh
```

Writes `dist/index.html` — the mirror with every overlay in `site/overlays/`
injected before `</body>`. It re-verifies the mirror's SHA-256 first and refuses
to build from a modified capture.

## Running it locally

```bash
./scripts/build-site.sh
python3 -m http.server 8080 --directory /home/dev/projects/thenie_v2/dist
```

Then open <http://localhost:8080>. Serve `site/` instead to see the pristine
mockup without our additions.

## Deploying

See [`docs/13-production-deployment-runbook.md`](docs/13-production-deployment-runbook.md).
