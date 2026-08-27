# 13 — Production Deployment Runbook — Static Site on the Existing Nginx

> **Already deployed once and just want to push a change live?**
> Skip to [Deploying an update](#deploying-an-update--the-everyday-path) —
> push here, then `git pull` + `./scripts/build-site.sh` + `cp` on the server.
> Everything from Part 1 onwards is first-time setup.

> ### If your server is already set up: nothing here needs redoing
>
> The site was re-captured on **2026-08-27** and is now a full six-page site
> rather than the order form alone. **None of that touches the server.** It is
> still one static HTML file in one web root behind the Nginx config you already
> wrote. Concretely, all of this stays exactly as it is — do **not** redo any of
> it:
>
> | Already working | Still correct? |
> |-----------------|----------------|
> | DNS A records for `thenie.id` / `www` (Part 1) | ✅ unchanged |
> | `/var/www/thenie` web root and its permissions (Part 3) | ✅ unchanged |
> | The clone at `/opt/thenie_v2` and its deploy key (Part 4) | ✅ unchanged |
> | The Nginx server blocks (Part 5) | ✅ unchanged |
> | HTTPS — certbot **or** the Cloudflare Origin Certificate (Part 7 / Appendix D) | ✅ unchanged |
> | Cloudflare proxy mode, real-IP config, `Full (strict)` (Appendix D) | ✅ unchanged |
> | UFW / firewall rules (Appendix D7) | ✅ unchanged |
> | The `X-Robots-Tag`, cache and security headers | ✅ unchanged |
>
> **The everyday deploy path is the whole job:** `git pull`,
> `./scripts/build-site.sh`, `sudo cp` the result into the web root. That is it.
>
> Three numbers changed, and the runbook below already reflects them — you only
> need them if you are checking something by hand:
>
> | | Before (2026-08-22) | Now (2026-08-27) |
> |---|---|---|
> | Mirror size | 4,615,031 B | **6,983,019 B** |
> | Mirror SHA-256 | `9d4cfefb…` | **`b66ed302…`** |
> | Deploy check | `grep -c 'class="wa-fab"'` | **`grep -c 'Floating-button clearance'`** |
>
> The last one changed because the capture now ships its own WhatsApp button, so
> our overlay no longer adds one — it only keeps the page's button from covering
> the checkout bar. See [[14-overlays]].
>
> One genuinely new fact, and it is a client-side one: the page now loads its
> font from Google. See [One thing the server does NOT serve](#one-thing-the-server-does-not-serve-the-font)
> below. Nothing to configure unless you add a `Content-Security-Policy`.
>
> **There is now also a backend engine** (`server/`, see [[15-backend-engine]]).
> It is **entirely optional and entirely additive**: the page works with it
> switched off, it is not in the deploy path for `dist/index.html`, and it
> changes nothing in Parts 1–13 or Appendix D. If you are not running it yet,
> ignore it — everything below is unaffected. When you want it, see
> [Appendix E](#appendix-e--the-backend-engine-optional).

**Audience:** someone who has never deployed a server before.
**Your setup:** the same Ubuntu machine described in the SCHOOL_CATERING runbook
— **Nginx** is the only web server, it already serves your **PHP site** (via
php-fpm) and the `meals.` / `api.` / `cdn.` subdomains for the school catering
system.
**What we add:** one more subdomain serving **one static HTML file**. Nothing
else. No Docker, no database, no application process.
**Style:** every command is written out in full, with absolute paths. Copy them
**in order**. Where you must type your own value it looks like `REPLACE_ME` — a
table tells you what to put.

> This deployment is **much simpler** than the school catering one. There is no
> app to run, no database to migrate, no secrets file, no storage bucket. If you
> have done that runbook, this one will take about fifteen minutes.

---

## How it fits together (read this once)

Your **Nginx is the front door** — it already owns ports **80** and **443**. We
do **not** change anything it already serves. We only **add one new subdomain**
that serves a folder of static files straight off the disk.

```
Internet ──443──▶  Nginx (your existing front door)
                       │
                       ├──▶  yourdomain.co.id          your existing PHP site   (untouched)
                       ├──▶  meals.sunshinefood.co.id  school catering website  (untouched)
                       ├──▶  api.sunshinefood.co.id    school catering API      (untouched)
                       ├──▶  cdn.sunshinefood.co.id    school catering storage  (untouched)
                       │
                       ├──▶  thenie.id      ◀── WE ADD THIS (the site)
                       └──▶  www.thenie.id  ◀── WE ADD THIS (301 → thenie.id)
                                     │
                                     └── /var/www/thenie/index.html   (one 6.7 MB file)
```

- **No Docker.** Nothing runs. Nginx reads a file off the disk and sends it.
- **No database.** The mockup has no backend at all — see [[08-technical-inventory]].
- **No ports.** Nothing new listens on anything.
- **No secrets.** There is nothing secret to configure.

Because nothing runs, there is nothing to crash, nothing to restart, and nothing
to monitor. The only failure modes are "Nginx config typo" and "certificate
expired".

### One thing the server does NOT serve: the font

New in the 2026-08-27 capture. The page pulls its typeface, **Baloo 2**, from
Google Fonts at render time:

```
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Baloo+2:wght@400;500;600;700;800&display=swap" rel="stylesheet">
```

That request goes **from the visitor's browser to Google**, not through your
server, so there is nothing to configure and nothing to install. But it does
change three things worth knowing before you go live:

- **The page is no longer fully self-contained.** Every other byte — all 44
  images included — is inlined in the file. The font is the single exception.
- **On a network that blocks Google, the site still works** but falls back to
  the browser's own sans-serif, so it looks noticeably different. `&display=swap`
  means text is readable immediately either way; it never blocks rendering.
- **If you ever add a `Content-Security-Policy` header, it must allow
  `fonts.googleapis.com` (style) and `fonts.gstatic.com` (font)** — otherwise
  you will silently break the typeface. This config deliberately ships no CSP;
  if you add one, test the font before calling it done.

The previous capture used system fonts only and had no external requests at all.
If self-hosting the font ever matters (privacy, an air-gapped network, EU data
rules), that is a rebuild task, not a server task — it means editing the page,
which the exact-mirror rule forbids. See [[07-fidelity-and-verification]].

---

## Placeholders — decide these now

| Placeholder | What it is | Example |
|-------------|-----------|---------|
| `thenie.id` | the subdomain to serve on | *(suggested — change everywhere below if you use another)* |
| `SERVER_IP` | your machine's public IP | `172.236.152.44` |
| `SSH_USER` | the user you log into the server as | `appuser` |
| `SSH_PORT` | the SSH port — **this machine uses 30022, not 22** | `30022` |
| `steven.william.anna@gmail.com` | email for the TLS certificate expiry warnings | — |

There are **no passwords and no secrets** in this deployment.

> **Not sure about the subdomain?** See Q-14 in [[09-open-questions]]. Anything
> works — just replace `thenie.id` consistently everywhere below.

---

## Deploying an update — the everyday path

**Already set the server up once?** This is the whole job. Everything below
Part 1 is first-time setup you do not repeat.

The server keeps its own clone of the repository at `/opt/thenie_v2`. You commit
and push from your machine; the server pulls, rebuilds, and copies the result
into the web root. Nothing else moves.

```
your machine ──git push──▶ GitHub ──git pull──▶ /opt/thenie_v2  (the clone)
                                                     │
                                                     │ ./scripts/build-site.sh
                                                     ▼
                                                 dist/index.html
                                                     │ sudo cp
                                                     ▼
                                          /var/www/thenie/index.html  ──▶ Nginx
```

### On your own machine — push first

Nothing reaches the server until it is on GitHub:

```bash
cd /home/dev/projects/thenie_v2
git status          # anything unstaged is about to be left behind
git add -A
git commit -m "describe the change"
git push origin main
```

### On the server — pull, build, publish

Log in (`ssh -p 30022 appuser@172.236.152.44`), then:

```bash
# 1. keep the version that is currently live, so you can roll back
sudo cp /var/www/thenie/index.html /var/www/thenie/index.html.bak

# 2. pull and rebuild
cd /opt/thenie_v2
git pull
./scripts/build-site.sh

# 3. publish
sudo cp /opt/thenie_v2/dist/index.html /var/www/thenie/index.html
sudo chown www-data:www-data /var/www/thenie/index.html
sudo chmod 644 /var/www/thenie/index.html

# 4. check
grep -c 'class="fab-wa"'              /var/www/thenie/index.html  # 1 = WhatsApp button is live
grep -c 'Floating-button clearance'   /var/www/thenie/index.html  # 1 = you published dist/, not the raw mirror
sha256sum /var/www/thenie/index.html                  # same hash build-site.sh printed
```

Then reload the site in a browser. **No `nginx reload`, no restart** — Nginx
reads the file off the disk on every request, and the `must-revalidate` header
means returning visitors pick up the new file immediately.

### The three things that go wrong

**You skipped `./scripts/build-site.sh`.** This is the big one. `dist/` is
git-ignored, so `git pull` brings the mirror and the overlays but *never* the
built file. Skip the build and you copy a stale `dist/index.html` — or none at
all — with no error anywhere. The site simply does not change, and it looks like
the pull failed. It did not; the build did not run.

**`git pull` refuses: "Your local changes would be overwritten".** Someone
edited the checkout on the server. Nothing in `/opt/thenie_v2` is meant to be
authored there — the fix is to throw the edits away:

```bash
cd /opt/thenie_v2
git checkout -- .
git pull
```

If you want to see what you are discarding first, `git diff` before the
`checkout`.

**`build-site.sh` refuses: "site/index.html no longer matches the recorded
capture hash".** The mirror has been modified — see the rule in `README.md`. The
build deliberately stops rather than ship a silently altered capture:

```bash
cd /opt/thenie_v2
git checkout -- site/index.html
./scripts/build-site.sh
```

### Rolling back

```bash
sudo cp /var/www/thenie/index.html.bak /var/www/thenie/index.html
sudo chown www-data:www-data /var/www/thenie/index.html
```

That restores the previous file instantly. To go back further, check out an
older commit in `/opt/thenie_v2` and rebuild — see [[14-overlays]] for what
the build is actually doing.

### If the server has no clone yet

Then this is your first deploy: do **Part 4 → Option A**, which creates
`/opt/thenie_v2` with a read-only deploy key. After that, this page is all you
need.

---

## Part 1 — Point the domain at the machine (DNS)

`thenie.id` is an **apex** (root) domain, not a subdomain, so this differs from
the school-catering setup. In your registrar's DNS panel, add **two A records**:

| Type | Name | Value |
|------|------|-------|
| A | `@` *(or blank, or `thenie.id` — registrars label the apex differently)* | `172.236.152.44` |
| A | `www` | `172.236.152.44` |

> **Do not use a CNAME for the apex.** The DNS standard does not allow a CNAME
> at the root of a domain. If your registrar offers ALIAS or ANAME, that works;
> a plain **A record** always does.
>
> **Using Cloudflare?** Either grey or orange cloud is fine — this is a plain
> static file with no presigned uploads, so the constraint that forces `cdn.` to
> stay unproxied does not apply here.

If you use **Cloudflare**, either grey cloud (DNS only) or orange cloud
(Proxied) is fine here — this is a plain static file with no presigned uploads,
so the problem that forces `cdn.` to be unproxied does not apply.

Wait a few minutes, then verify **from your own computer**:

```bash
nslookup thenie.id
nslookup www.thenie.id
```

**Both** must print `172.236.152.44`. Do not continue until they do — certbot
cannot issue a certificate otherwise.

---

## Part 2 — Log in and sanity-check the machine

```bash
ssh -p 30022 appuser@172.236.152.44
```

> **Note the port.** This machine does not use the default SSH port 22. `ssh`
> takes a lower-case `-p`, `scp` takes an upper-case `-P` — mixing them up is a
> common and confusing failure.

Confirm Nginx is running and is your front door:

```bash
systemctl is-active nginx
```

Expected output:

```
active
```

Check you have room for the file (it is 6.7 MB, so this is a formality):

```bash
df -h /var/www
```

Confirm your existing sites are healthy **before** you change anything, so you
know any later breakage was yours:

```bash
sudo nginx -t
```

Expected:

```
nginx: configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

---

## Part 3 — Create the web root

```bash
sudo mkdir -p /var/www/thenie
```

---

## Part 4 — Get the file onto the server

Two ways. **Option A** is better if the server can reach GitHub; **Option B**
needs nothing but your laptop.

### Option A — clone from GitHub (recommended)

The repository is private, so the server needs read access. Create a **deploy
key** — a key that can read this one repository and nothing else:

```bash
ssh-keygen -t ed25519 -C "thenie-deploy" -f /home/appuser/.ssh/thenie_deploy -N ""
cat /home/appuser/.ssh/thenie_deploy.pub
```

Copy the printed line. In GitHub, open
`https://github.com/stevenwilliam/thenie_v2/settings/keys` →
**Add deploy key** → paste it → title `server` → leave **Allow write access
unchecked** → **Add key**.

Tell SSH to use that key for GitHub:

```bash
sudo vi /home/appuser/.ssh/config
```

Add:

```
Host github-thenie
    HostName github.com
    User git
    IdentityFile /home/appuser/.ssh/thenie_deploy
    IdentitiesOnly yes
```

Fix the permissions (SSH refuses loose ones):

```bash
chmod 600 /home/appuser/.ssh/config /home/appuser/.ssh/thenie_deploy
chmod 644 /home/appuser/.ssh/thenie_deploy.pub
```

Test, then clone:

```bash
ssh -T git@github-thenie
sudo mkdir -p /opt/thenie_v2
sudo chown appuser:appuser /opt/thenie_v2
git clone github-thenie:stevenwilliam/thenie_v2.git /opt/thenie_v2
```

The test prints `Hi stevenwilliam/thenie_v2! You've successfully authenticated`
— "does not provide shell access" alongside it is normal and not an error.

Build the page, then publish it:

```bash
cd /opt/thenie_v2
./scripts/build-site.sh
sudo cp /opt/thenie_v2/dist/index.html /var/www/thenie/index.html
```

`build-site.sh` stitches the untouched mirror together with the overlays in
`site/overlays/` — today that means the floating WhatsApp button, see
[[14-overlays]] — and writes `dist/index.html`. It needs nothing but bash
and coreutils, and it verifies the mirror's hash before it builds. **`dist/` is
git-ignored, so it does not arrive with `git pull`; you build it on the server.**
Copying `site/index.html` directly still works, but publishes the page *without*
the WhatsApp button.

### Option B — copy straight from your laptop

Run this **on your own machine, not on the server**. This is the single most
common mistake in this runbook: if your prompt reads `appuser@…:/var/www/thenie$`
you are on the server, and `scp` will fail with

```
scp: stat local "/home/dev/projects/thenie_v2/site/index.html": No such file or directory
```

because that path exists only on your dev machine. Type `exit` first, build,
then copy the **built** file (`dist/`, not `site/`):

```bash
cd /home/dev/projects/thenie_v2
./scripts/build-site.sh
scp -P 30022 /home/dev/projects/thenie_v2/dist/index.html appuser@172.236.152.44:/tmp/index.html
```

Then back **on the server**:

```bash
sudo mv /tmp/index.html /var/www/thenie/index.html
```

### Option C — pull it from the live site, on the server

> **This deploys the mockup *without* the floating WhatsApp button.** Upstream
> is the original Netlify capture; it knows nothing about our overlays. Use it
> only to reproduce the pristine mirror — for the real site, use Option A or B.

The quickest route when you are already logged in, and it needs no key and no
laptop. The mirror is byte-identical to the live page, so fetching upstream
gives the same file — and the hash check below is what proves it:

```bash
curl -sSL https://thenie-catering-order.netlify.app/ -o /tmp/index.html
sha256sum /tmp/index.html
```

Then:

```bash
sudo mv /tmp/index.html /var/www/thenie/index.html
```

**Only use this if the hash matches** the value in the next section. If upstream
has changed since 2026-08-22 it will not, and you should use Option A or B to
deploy the captured version instead of a newer, undocumented one.

### Set ownership and permissions (all options)

```bash
sudo chown -R www-data:www-data /var/www/thenie
sudo chmod 755 /var/www/thenie
sudo chmod 644 /var/www/thenie/index.html
```

### Confirm the file arrived intact

```bash
sha256sum /var/www/thenie/index.html
grep -c 'class="fab-wa"'            /var/www/thenie/index.html  # must print 1
grep -c 'Floating-button clearance' /var/www/thenie/index.html  # must print 1
```

The `sha256sum` must match what `sha256sum dist/index.html` printed on the
machine you built on — it changes every time an overlay changes, so compare the
two, do not memorise a value. The `grep` is the quick "did the WhatsApp button
make it" check.

If you deployed the **pristine mirror** via Option C, `grep` prints `0` and the
hash must instead be exactly:

```
b66ed30212d3cb3ffe00c1385ea9a23996d8611cb3bed40a288fed99b6ed9689
```

If it does not, the file was corrupted or altered in transit — copy it again.
Do not continue with a mismatched hash. See [[07-fidelity-and-verification]].

---

## Part 5 — Create the Nginx site config

```bash
sudo vi /etc/nginx/sites-available/thenie
```

Paste this exactly:

```nginx
# Send www to the bare domain, so the site has exactly one canonical address
# and does not compete with itself in search results.
server {
    listen 80;
    listen [::]:80;
    server_name www.thenie.id;

    # The ACME challenge MUST be matched before the redirect below, or certbot
    # cannot validate www.thenie.id. A bare `return` at server level short-
    # circuits the rewrite phase before any location is chosen, so the redirect
    # lives inside `location /` rather than at server level. This is the whole
    # reason the block is shaped this way -- do not "simplify" it.
    location ^~ /.well-known/acme-challenge/ {
        root /var/www/thenie;
    }

    # Redirect straight to HTTPS, not to http://thenie.id. Sending it to plain
    # HTTP costs an extra hop, and behind a TLS-terminating proxy set to
    # "Flexible" SSL it produces an infinite redirect loop
    # (ERR_TOO_MANY_REDIRECTS) -- see Part 12.
    location / {
        return 301 https://thenie.id$request_uri;
    }
}

server {
    listen 80;
    listen [::]:80;
    server_name thenie.id;

    root /var/www/thenie;
    index index.html;

    # ---- Headers ----
    # ALL add_header directives live here at server level, and NO location
    # block below declares one. That is deliberate: nginx does not merge
    # add_header directives, it REPLACES them. The moment any location declares
    # a single add_header, every header inherited from the server block is
    # dropped for requests hitting that location -- silently, with a perfectly
    # valid config and no warning from `nginx -t`.
    #
    # This is not theoretical. An earlier draft of this runbook set
    # Cache-Control inside `location = /index.html`. It passed `nginx -t`,
    # served the file correctly, and returned the cache header with all four
    # security headers missing. It was caught by actually reading the response.
    #
    # If you ever add an add_header inside a location, repeat all five here.

    # The page is one 6.7 MB HTML file replaced wholesale on each deploy, so it
    # must never be cached hard -- otherwise visitors keep an old copy after an
    # update. This matches what Netlify serves upstream.
    add_header Cache-Control "public, max-age=0, must-revalidate" always;

    # >>> READ THIS BEFORE YOU GO LIVE <<<
    # thenie.id is the real business domain, so this line is a DECISION, not a
    # default. While it is present, Google will not index the site at all.
    #
    #   Keep it   while thenie.id is still a preview you are showing people.
    #   DELETE it the day this becomes the public site -- otherwise the business
    #             is invisible in search, and nobody will know why.
    #
    # Deleting it is not enough on its own: the page ships no meta description,
    # no Open Graph tags, no canonical URL and no JSON-LD (see
    # docs/08-technical-inventory.md), so it will index thinly and share as a
    # bare link with no title card. It does now have real <h1>s -- six of them,
    # one per client-side page -- which is a step up from the previous capture
    # but is not on its own an SEO baseline. See Q-13 in
    # docs/09-open-questions.md.
    add_header X-Robots-Tag "noindex, nofollow" always;

    # Baseline security headers. No framing, no MIME sniffing, no referrer leak.
    add_header X-Frame-Options        "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy        "strict-origin-when-cross-origin" always;

    # Everything resolves to the single page. The site DOES have client-side
    # routes now (#home, #about, #menu, #pricing, #order, #contact), but they are
    # URL *fragments*: the browser never sends them to the server, so nginx only
    # ever sees "/". No rewrite rules are needed for them. This line just keeps a
    # stray path from returning a bare 404.
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Compression. The inline CSS/JS/markup compresses well; the base64 JPEGs
    # are already compressed and barely shrink, so expect roughly 6.7 MB to
    # become roughly 4.9 MB (measured: 6,985,620 -> 5,129,765 bytes at
    # gzip_comp_level 6). That is a limit of the mirrored file, not of the
    # server -- see docs/08-technical-inventory.md.
    #
    # Do NOT list text/html in gzip_types -- nginx always gzips it, and naming
    # it again raises: [warn] duplicate MIME type "text/html".
    gzip             on;
    gzip_types       text/css application/javascript;
    gzip_min_length  1024;
    gzip_comp_level  6;

    access_log /var/log/nginx/thenie.access.log;
    error_log  /var/log/nginx/thenie.error.log;
}
```

> **This config was tested, not just written.** It was checked with `nginx -t`
> (clean), then used to actually serve the real 6.7 MB file while the response
> headers and the served bytes were compared against the capture hash. The
> `add_header` placement above is the *result* of that test — the first draft
> looked right, passed `nginx -t`, served the file fine, and silently dropped
> every security header. See [[PROGRESS]].

---

## Part 6 — Enable it, test, reload

```bash
sudo ln -s /etc/nginx/sites-available/thenie /etc/nginx/sites-enabled/thenie
sudo nginx -t
```

`nginx -t` **must** say `syntax is ok` and `test is successful`. If it does not,
you have a typo — fix it and re-test. **Do not reload on a failed test.**

```bash
sudo systemctl reload nginx
```

`reload` re-reads the config without dropping connections. Your PHP site and the
school catering subdomains keep serving throughout.

Confirm it answers over plain HTTP before adding TLS:

```bash
curl -sI http://thenie.id | head -5
```

Expect `HTTP/1.1 200 OK` and `Content-Type: text/html`.

---

## Part 7 — HTTPS

> **Proxying through Cloudflare?** `thenie.id` is. **Skip this Part entirely**
> and use [Appendix D](#appendix-d--behind-cloudflare-the-thenieid-setup) —
> certbot's HTTP-01 challenge is intercepted by the Cloudflare edge, and a
> Cloudflare Origin Certificate is both easier and longer-lived.

Install certbot if this machine does not have it yet:

```bash
sudo apt update
sudo apt -y install certbot python3-certbot-nginx
```

Issue the certificate:

```bash
sudo certbot --nginx -d thenie.id -d www.thenie.id --email steven.william.anna@gmail.com --agree-tos --no-eff-email
```

When certbot asks about redirecting HTTP→HTTPS, choose **yes (redirect)**.

Certbot edits `/etc/nginx/sites-available/thenie` in place, adding the `listen
443 ssl` block and the certificate paths. That is expected.

Renewal is automatic via a systemd timer. Confirm it:

```bash
sudo systemctl list-timers | grep certbot
sudo certbot renew --dry-run
```

The dry run must end with `Congratulations, all simulated renewals succeeded`.

---

## Part 8 — Confirm everything

```bash
curl -sI https://thenie.id | head -12
```

Check for:

- `HTTP/2 200`
- `content-type: text/html`
- `x-robots-tag: noindex, nofollow`
- `x-frame-options: SAMEORIGIN`
- `cache-control: public, max-age=0, must-revalidate`

Confirm compression is applied:

```bash
curl -sI -H "Accept-Encoding: gzip" https://thenie.id | grep -i content-encoding
```

Expect `content-encoding: gzip`.

Confirm the served bytes are the real thing — decompressed, this must equal the
hash of the file you published:

```bash
curl -s --compressed https://thenie.id | sha256sum
sha256sum /var/www/thenie/index.html      # must be identical
```

If the two differ, something between Nginx and the disk is rewriting the page.

The served hash is **not** the capture hash any more: what ships is the mirror
plus the overlays in `site/overlays/` ([[14-overlays]]), so it changes
whenever an overlay does. `b66ed30212d3cb3ffe00c1385ea9a23996d8611cb3bed40a288fed99b6ed9689`
is the hash of `site/index.html` alone — the untouched capture — and that one
never changes.

**Now confirm you broke nothing.** Open your existing PHP site in a browser, and:

```bash
curl -sI https://meals.sunshinefood.co.id | head -3
curl -s  https://api.sunshinefood.co.id/health
```

Finally, open **https://thenie.id** on a real phone. It is a
mobile-first page (see [[06-design-system]]) and the phone is where it is meant
to be judged. Walk one order end to end: pick a meal, pick dates, add it, open
the cart, press send — WhatsApp should open with the order pre-filled. **Do not
press send in WhatsApp** unless you want the business to receive a test order.

---

## Part 9 — Updating to a new version

This is the same procedure as [Deploying an update](#deploying-an-update--the-everyday-path)
near the top of the page, repeated here so Part 9 is not a dead end.

If you used **Option A** (the server has its own clone — the normal case):

```bash
# 1. keep the current version, so Part 10 can roll back to it
sudo cp /var/www/thenie/index.html /var/www/thenie/index.html.bak

# 2. pull, rebuild, publish
cd /opt/thenie_v2
git pull
./scripts/build-site.sh
sudo cp /opt/thenie_v2/dist/index.html /var/www/thenie/index.html
sudo chown www-data:www-data /var/www/thenie/index.html
sudo chmod 644 /var/www/thenie/index.html

# 3. check
sha256sum /var/www/thenie/index.html
grep -c 'class="fab-wa"'              /var/www/thenie/index.html  # 1 = WhatsApp button is live
grep -c 'Floating-button clearance'   /var/www/thenie/index.html  # 1 = you published dist/, not the raw mirror
```

**Do not skip `./scripts/build-site.sh`.** `dist/` is git-ignored, so `git pull`
brings the mirror and the overlays but never the built file — pulling without
rebuilding leaves the old page in place and looks like the deploy silently did
nothing.

If `git pull` refuses with *"Your local changes would be overwritten"*, the
checkout has been edited on the server. Throw the local edits away and pull
again — nothing on the server is meant to be authored there:

```bash
cd /opt/thenie_v2
git checkout -- .
git pull
```

If you used **Option B**, rebuild on your laptop and repeat the `scp` from Part 4.

**No Nginx reload is needed** — Nginx reads the file from disk on each request.
The `must-revalidate` header means returning visitors pick the new file up
immediately.

---

## Part 10 — Rollback

```bash
sudo cp /var/www/thenie/index.html.bak /var/www/thenie/index.html
sudo chown www-data:www-data /var/www/thenie/index.html
```

To remove the site entirely without touching anything else:

```bash
sudo rm /etc/nginx/sites-enabled/thenie
sudo nginx -t
sudo systemctl reload nginx
```

The files stay in `/var/www/thenie`; only the subdomain stops being served.

---

## Part 11 — Everyday commands

```bash
# is nginx healthy?
systemctl is-active nginx
sudo nginx -t

# watch this site's traffic
sudo tail -f /var/log/nginx/thenie.access.log

# this site's errors only
sudo tail -f /var/log/nginx/thenie.error.log

# reload after a config change (no downtime)
sudo systemctl reload nginx

# certificate expiry
sudo certbot certificates

# confirm the served page matches what is on disk
curl -s --compressed https://thenie.id | sha256sum
sha256sum /var/www/thenie/index.html

# confirm the WhatsApp button is live
curl -s --compressed https://thenie.id | grep -c 'Floating-button clearance'   # 1

# confirm the mirror in the clone is still the untouched capture
cd /opt/thenie_v2 && ./scripts/verify-mirror.sh
```

---

## Part 12 — Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `nginx -t` fails | Typo in the config | Read the line number it prints; fix; re-test. Never reload on a failed test. |
| **404 Not Found** | File missing or wrong root | `ls -la /var/www/thenie/index.html` |
| **403 Forbidden** | Permissions | `sudo chown -R www-data:www-data /var/www/thenie && sudo chmod 644 /var/www/thenie/index.html` |
| Certbot fails | DNS not resolving yet | `nslookup thenie.id` must return `172.236.152.44`; wait and retry. |
| Serves the **wrong site** | Another server block claims the name, or this one is unreachable | `sudo nginx -T \| grep -n "server_name"` |
| Page loads but photos missing | Truncated file | Compare `sha256sum` against `dist/index.html`. Re-copy. |
| Very slow first load | The 6.7 MB payload | Expected. See Q-15 in [[09-open-questions]]. |
| Old version still showing | Browser cache | Hard-refresh (Ctrl-Shift-R). Confirm `cache-control` in Part 8. |
| Pulled, but the site did not change | `./scripts/build-site.sh` was skipped, or `dist/index.html` was never copied to the web root | Re-run *Deploying an update* in full. `dist/` is git-ignored — the pull alone cannot change the served file. |
| WhatsApp button sits on top of the checkout bar | Published `site/index.html` instead of `dist/index.html` | `grep -c 'Floating-button clearance' /var/www/thenie/index.html` — if `0`, copy from `dist/`. |
| `git pull`: *local changes would be overwritten* | The server checkout was edited | `cd /opt/thenie_v2 && git checkout -- . && git pull` |
| `build-site.sh`: *no longer matches the recorded capture hash* | `site/index.html` was modified | `cd /opt/thenie_v2 && git checkout -- site/index.html` |
| Existing PHP site broke | You edited its config | `sudo nginx -T` and compare. This runbook never touches it. |
| **ERR_TOO_MANY_REDIRECTS** | A proxy or registrar is redirecting back at you | See the dedicated section below — do not start by editing Nginx. |

### ERR_TOO_MANY_REDIRECTS

A loop means two redirects point at each other. Find the pair before changing
anything. First ask what the origin itself says, bypassing DNS and any proxy:

```bash
curl -sI -H "Host: www.thenie.id" http://127.0.0.1/ | grep -Ei "^HTTP|^Location"
curl -sI -H "Host: thenie.id"     http://127.0.0.1/ | grep -Ei "^HTTP|^Location"
```

Expected: `www` gives one `301` to `https://thenie.id/`, and the apex gives
`200` (or a single `301` to HTTPS that certbot added). Then compare with what
the outside world sees:

```bash
curl -sIL --max-redirs 5 http://thenie.id 2>&1 | grep -Ei "^HTTP|^Location"
```

**If the origin is clean but the public chain loops, the loop is not on this
machine.** In order of likelihood:

1. **Cloudflare SSL mode set to "Flexible."** Cloudflare terminates TLS and
   talks to the origin over plain HTTP; the origin redirects to HTTPS;
   Cloudflare re-requests over HTTP; forever. Fix: SSL/TLS → **Full (strict)**.
   This is the most common cause and is completely invisible from the server.
2. **Registrar "web forwarding" apex → www**, while Nginx sends www → apex.
   Common on `.id` registrars and often on by default. Fix: delete the
   forwarding rule; the A records in Part 1 already do the job.
3. **Both a redirect rule and certbot's redirect on the same block** — check
   `sudo grep -nE "return|if \(" /etc/nginx/sites-available/thenie` for two
   redirects in one server block.

Only if the *origin* loops is it this config. Verify it directly:

```bash
sudo grep -nE "server_name|listen|return|if \(" /etc/nginx/sites-available/thenie
```

---

## Part 13 — Security notes

This is about as small an attack surface as a website has: **one static file, no
process, no database, no input reaching the server**. Everything the customer
types stays in their browser and goes to WhatsApp (see [[08-technical-inventory]]).

Still worth knowing:

- **The `noindex` header is now the single most consequential line in the
  config.** On a staging subdomain it was free insurance. On `thenie.id` it
  suppresses the real business in search. Decide it deliberately, and if you
  remove it, read Q-13 first — the page has no SEO baseline to index.
- **Bank details are on a public page at a memorable domain.** That was already
  true on Netlify, but a real domain gets found. See Q-17.
- **There is no access control.** Anyone with the URL sees it. If it should be
  private, add HTTP basic auth:

  ```bash
  sudo apt -y install apache2-utils
  sudo htpasswd -c /etc/nginx/.htpasswd-thenie thenie
  ```

  then inside the `server` block:

  ```nginx
  auth_basic           "Thenie mockup";
  auth_basic_user_file /etc/nginx/.htpasswd-thenie;
  ```

  followed by `sudo nginx -t && sudo systemctl reload nginx`.

- **Bank details are public** on this page (BR-8.2) — an account number and a
  personal name. That is already true of the live Netlify site, but it is worth
  a deliberate decision before you widen the audience. See Q-17 in
  [[09-open-questions]].
- **Keep the machine patched:** `sudo apt update && sudo apt upgrade`.
- **No data is stored**, so there is nothing to back up beyond the git
  repository — which is the file's real backup.

---

## Appendix D — Behind Cloudflare (the `thenie.id` setup)

**Use this instead of Parts 5, 6 and 7** when the domain is proxied through
Cloudflare — orange cloud in the DNS panel. `thenie.id` is.

### Why the default runbook loops here

With Cloudflare proxying and **no certificate on the origin**, Cloudflare's SSL
mode has to be **Flexible**, which means Cloudflare always speaks plain **HTTP**
to your server. Your server answers `301 -> https://…`. Cloudflare hands that
redirect to the browser, the browser asks for HTTPS again, Cloudflare again
fetches over HTTP, and the origin redirects again. Forever:

```
browser --HTTPS--> Cloudflare --HTTP--> origin
                                          |
                        301 https://thenie.id  <-- origin never sees HTTPS,
                                          |         so it never stops redirecting
browser <--- 301 -------- Cloudflare <----+
```

Symptom: **`ERR_TOO_MANY_REDIRECTS`**. It is invisible from the server — every
`curl` against `127.0.0.1` looks perfectly correct, because the loop only
exists across the Cloudflare hop.

The fix is to put a real certificate on the origin and set Cloudflare to
**Full (strict)** so it connects over HTTPS end to end.

**Which certificate does not matter** — a Cloudflare Origin Certificate (D1) and
a Let's Encrypt certificate (D3b) both work. What matters is that the origin
answers on 443 *and* that Cloudflare is switched off Flexible. Changing the
certificate alone fixes nothing; changing the SSL mode alone breaks the site.

### Why not certbot here

Cloudflare proxies port 80, so certbot's HTTP-01 challenge is intercepted by the
edge — and if **Always Use HTTPS** is on, the challenge is redirected and
validation fails. A **Cloudflare Origin Certificate** avoids all of it: free,
valid **15 years**, trusted by Cloudflare specifically, and no renewal timer to
forget. It is only trusted *by Cloudflare*, which is exactly right — visitors
see Cloudflare's own public certificate.

### D1 — Create the Origin Certificate

In the Cloudflare dashboard: **SSL/TLS -> Origin Server -> Create Certificate**.

- Private key type: **RSA (2048)**
- Hostnames: `thenie.id` and `*.thenie.id`
- Validity: **15 years**

Cloudflare shows two blocks, **once only**. Copy both.

On the server:

```bash
sudo mkdir -p /etc/ssl/cloudflare
sudo chmod 700 /etc/ssl/cloudflare
sudo vi /etc/ssl/cloudflare/thenie.id.pem     # paste the Origin Certificate
sudo vi /etc/ssl/cloudflare/thenie.id.key     # paste the Private Key
sudo chmod 600 /etc/ssl/cloudflare/thenie.id.key
sudo chmod 644 /etc/ssl/cloudflare/thenie.id.pem
```

### D2 — Let Nginx see real visitor IPs

Every request arrives from Cloudflare, so without this your logs record
Cloudflare's addresses instead of your visitors'.

```bash
sudo sh -c '{ \
  curl -s https://www.cloudflare.com/ips-v4 | sed "s|^|set_real_ip_from |; s|$|;|"; \
  curl -s https://www.cloudflare.com/ips-v6 | sed "s|^|set_real_ip_from |; s|$|;|"; \
} > /etc/nginx/cloudflare-ips.conf'
head -3 /etc/nginx/cloudflare-ips.conf
```

Re-run that occasionally; Cloudflare's ranges change rarely but they do change.

### D3 — Replace the site config

```bash
sudo vi /etc/nginx/sites-available/thenie
```

Replace the whole file with:

```nginx
# ---- Real visitor IPs ----
# Behind Cloudflare every request arrives from a Cloudflare address, so logs and
# any rate limiting would otherwise see Cloudflare instead of the visitor.
include /etc/nginx/cloudflare-ips.conf;
real_ip_header CF-Connecting-IP;

server {
    listen 80;
    listen [::]:80;
    server_name thenie.id www.thenie.id;

    # Kept reachable so a certificate can be validated over HTTP if you ever
    # switch away from the Cloudflare Origin certificate.
    location ^~ /.well-known/acme-challenge/ {
        root /var/www/thenie;
    }

    location / {
        return 301 https://thenie.id$request_uri;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name www.thenie.id;

    ssl_certificate     /etc/ssl/cloudflare/thenie.id.pem;
    ssl_certificate_key /etc/ssl/cloudflare/thenie.id.key;

    return 301 https://thenie.id$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name thenie.id;

    ssl_certificate     /etc/ssl/cloudflare/thenie.id.pem;
    ssl_certificate_key /etc/ssl/cloudflare/thenie.id.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    root /var/www/thenie;
    index index.html;

    add_header Cache-Control "public, max-age=0, must-revalidate" always;
    add_header X-Robots-Tag "noindex, nofollow" always;
    add_header X-Frame-Options        "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy        "strict-origin-when-cross-origin" always;

    location / {
        try_files $uri $uri/ /index.html;
    }

    gzip             on;
    gzip_types       text/css application/javascript;
    gzip_min_length  1024;
    gzip_comp_level  6;

    access_log /var/log/nginx/thenie.access.log;
    error_log  /var/log/nginx/thenie.error.log;
}
```

### D3b — Already have a Let's Encrypt certificate?

You do not need the Origin Certificate. Any certificate the origin can present
works — Cloudflare only has to be able to validate it. Swap the two cert lines
in the D3 config for:

```nginx
ssl_certificate     /etc/letsencrypt/live/thenie.id/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/thenie.id/privkey.pem;
```

Check what you actually hold first — a certificate can exist while Nginx never
references it, which is exactly the state this server was found in:

```bash
sudo certbot certificates
sudo nginx -T 2>/dev/null | grep -nE "listen 443|ssl_certificate"
```

> **Renewal breaks while Cloudflare proxies — this is the catch.**
> certbot's HTTP-01 challenge is answered by the Cloudflare edge, not your
> origin, and **Always Use HTTPS** redirects it. So `certbot renew` fails, and
> under **Full (strict)** an expired origin certificate takes the site down
> roughly 60 days later, with nothing having changed that day to explain it.
>
> Prove it now rather than discovering it then:
>
> ```bash
> sudo certbot renew --dry-run
> ```
>
> If it fails, pick one:
>
> 1. **DNS-01 with the Cloudflare plugin** — renews automatically while
>    proxied, and is the right answer if you want to stay on Let's Encrypt:
>
>    ```bash
>    sudo apt -y install python3-certbot-dns-cloudflare
>    sudo mkdir -p /etc/letsencrypt/secrets
>    sudo vi /etc/letsencrypt/secrets/cloudflare.ini   # dns_cloudflare_api_token = <token>
>    sudo chmod 600 /etc/letsencrypt/secrets/cloudflare.ini
>    sudo certbot certonly \
>      --dns-cloudflare \
>      --dns-cloudflare-credentials /etc/letsencrypt/secrets/cloudflare.ini \
>      -d thenie.id -d www.thenie.id
>    ```
>
>    Create the token in Cloudflare under **My Profile -> API Tokens**, with the
>    **Zone -> DNS -> Edit** template scoped to `thenie.id` only.
>
> 2. **Use the Origin Certificate** (D1) and forget renewal for 15 years.
> 3. **Grey-cloud the domain** for each renewal — works, but it is a manual
>    chore every 60 days and easy to forget.
>
> Option 2 is the least work; option 1 is the most standard.

### D4 — Enable, test, reload

```bash
sudo ln -sf /etc/nginx/sites-available/thenie /etc/nginx/sites-enabled/thenie
sudo nginx -t
sudo systemctl reload nginx
```

Confirm the origin now answers on **443**, which it previously did not:

```bash
curl -skI -H "Host: thenie.id" https://127.0.0.1/ | head -3
```

### D5 — Switch Cloudflare to Full (strict)

**SSL/TLS -> Overview -> Full (strict)**. Do this only *after* D4 succeeds — the
origin must already be answering HTTPS or the site goes down.

Then **SSL/TLS -> Edge Certificates -> Always Use HTTPS: On**, so Cloudflare
handles the HTTP-to-HTTPS upgrade at the edge instead of your origin.

### D6 — Verify

```bash
curl -sIL --max-redirs 5 http://thenie.id     2>&1 | grep -Ei "^HTTP|^location"
curl -sIL --max-redirs 5 http://www.thenie.id 2>&1 | grep -Ei "^HTTP|^location"
```

Both must end in a single `200`, with `www` passing through exactly one `301` to
`https://thenie.id/`. No repeated `Location` lines — a repeat is still a loop.

```bash
curl -s https://thenie.id | sha256sum
sha256sum /var/www/thenie/index.html
```

Both must print the same hash — the one `build-site.sh` reported when you
published. (Not the capture hash; see Part 8.)

### D7 — Optional hardening

Once Cloudflare is the only route in, refuse direct-to-IP traffic so nobody can
bypass the edge:

```bash
sudo ufw allow from 173.245.48.0/20 to any port 443 proto tcp
# ... repeat for each range in /etc/nginx/cloudflare-ips.conf, then:
# sudo ufw delete allow 443/tcp
```

Do this **last**, and keep your SSH port (30022) open, or you will lock yourself
out. Cloudflare's **Authenticated Origin Pulls** achieves the same thing more
cleanly if you prefer.

### Verified

The config in D3 was tested locally against nginx 1.28.3 with a throwaway
certificate: `nginx -t` clean, HTTP returns one `301` to `https://thenie.id/`,
HTTPS apex returns `200` over HTTP/2 with all five headers, `www` over HTTPS
returns a single `301` to the apex, and the bytes served over TLS hash to the
capture value.

---

## Appendix E — The backend engine (optional)

**Read this only when you want to make the weekly menu editable without
re-capturing the page.** Everything in Parts 1–13 works without it, and nothing
in this appendix changes any of it.

What it adds: a small Go service and a PostgreSQL database, so publishing next
week's menu is one API call instead of an HTML edit. Full design in
[[15-backend-engine]].

### What stays exactly as it is

| Already working | Affected? |
|-----------------|-----------|
| DNS, the web root, the clone at `/opt/thenie_v2` | ❌ no change |
| The Nginx server blocks from Part 5 | ➕ **one `location` block added**, nothing changed |
| HTTPS, certbot or the Cloudflare Origin Certificate | ❌ no change |
| The everyday deploy path (`git pull`, build, `cp`) | ❌ no change |
| The page itself if the service is down | ❌ still works — it falls back to the captured content |

### E1 — Database and role

The machine already runs PostgreSQL for the school catering system. This adds
one more database and one more role, exactly as `healthy_catering` and `ruuma`
already do — **it does not touch either of them**.

```bash
sudo -u postgres psql -c "CREATE ROLE thenie LOGIN PASSWORD 'REPLACE_ME'"
sudo -u postgres createdb -O thenie thenie
sudo -u postgres createdb -O thenie thenie_test
```

Generate the password with:

```bash
head -c 24 /dev/urandom | base64 | tr -d '/+=' | head -c 28
```

### E2 — Build and configure

```bash
cd /opt/thenie_v2/server
go build -o bin/thenied ./cmd/thenied

cd /opt/thenie_v2
sudo cp .env.example .env
sudo vi .env
sudo chmod 600 .env
sudo chown appuser:appuser .env
```

Three values must be real:

| Key | Value |
|-----|-------|
| `APP_ENV` | `production` |
| `DATABASE_URL` | the role and password from E1 |
| `ADMIN_TOKEN` | at least 24 characters — `head -c 32 /dev/urandom \| base64 \| tr -d '/+=' \| head -c 40` |

`APP_ENV=production` makes the service **refuse to start** without an
`ADMIN_TOKEN`. That is deliberate: a public deployment with unauthenticated
write endpoints is not a warning, it is an outage waiting to happen.

### E3 — Migrate and seed

```bash
cd /opt/thenie_v2
./server/bin/thenied migrate up
./server/bin/thenied seed
./server/bin/thenied validate
```

`seed` reads the content **out of `site/index.html`**, so the database starts as
a faithful copy of what the site actually shipped. `validate` re-checks it
against every domain rule and prints what is wrong, if anything.

### E4 — Run it under systemd

```bash
sudo vi /etc/systemd/system/thenied.service
```

```ini
[Unit]
Description=Thenie site-configuration service
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=appuser
Group=appuser
WorkingDirectory=/opt/thenie_v2
ExecStart=/opt/thenie_v2/server/bin/thenied serve
Restart=on-failure
RestartSec=5

# It reads one .env file and listens on one loopback port. Nothing else.
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/thenie_v2
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
LockPersonality=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now thenied
sudo systemctl status thenied --no-pager
curl -s http://127.0.0.1:8082/healthz
```

The service binds `:8082`. **Do not open that port in the firewall** — Nginx
reaches it over loopback, and nothing outside the machine should talk to it
directly.

### E5 — One Nginx location block

Add this **inside the existing `thenie.id` server block**, alongside the
`location /` that is already there. Do not create a new server block, and do not
touch anything else in the file.

```nginx
    # The config API for the hydration overlay. Same origin as the page, which
    # means the browser never preflights and no CORS configuration is needed.
    location /api/ {
        proxy_pass         http://127.0.0.1:8082;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;
    }
```

> **The `add_header` trap from Part 5 applies here too.** This block declares no
> `add_header`, so it inherits all five from the server block. If you ever add
> one to it, you must repeat all five — nginx replaces, it does not merge.

```bash
sudo nginx -t
sudo systemctl reload nginx
curl -s https://thenie.id/api/v1/site-config/revision
```

Serving the API from the same origin is the reason no `CORS_ORIGINS` value is
needed. If you ever move it to its own host, set `CORS_ORIGINS` to the page's
origin and rebuild with `THENIE_API_BASE`.

### E6 — Rebuild the page once

The hydration overlay ships in `site/overlays/`, so it arrives with `git pull` —
but `dist/` is git-ignored, so it only reaches the served page when you rebuild:

```bash
cd /opt/thenie_v2
git pull
./scripts/build-site.sh
sudo cp dist/index.html /var/www/thenie/index.html
sudo chown www-data:www-data /var/www/thenie/index.html
```

Check it landed:

```bash
grep -c 'Content hydration' /var/www/thenie/index.html   # 1
```

Then load the site and open the browser console. One line confirms it:

```
[thenie] hydrated revision 370 (4 menu block(s), 2 contact link(s))
```

### E7 — Publishing a menu

```bash
curl -X PUT https://thenie.id/api/v1/admin/menu/cycles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d @next-week.json
```

The payload shape is in [[15-backend-engine]]. The change is live on the next
page load — no rebuild, no `cp`, no re-capture.

### E8 — Everyday commands

```bash
sudo systemctl status thenied --no-pager
sudo journalctl -u thenied -f
sudo journalctl -u thenied --since "1 hour ago" -p err

cd /opt/thenie_v2 && ./server/bin/thenied validate     # is the stored content sane?
curl -s https://thenie.id/api/v1/site-config/revision  # has anything changed?
```

### E9 — Turning it off

Three levels, least to most drastic:

```bash
# 1. Stop hydrating, keep the service running (no deploy needed)
curl -X PUT https://thenie.id/api/v1/admin/params/site.hydration_enabled \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' \
  -d '{"value":"false"}'

# 2. Stop the service. The page falls back to captured content on its own.
sudo systemctl stop thenied

# 3. Remove the overlay entirely and rebuild.
cd /opt/thenie_v2 && rm site/overlays/hydrate.html && ./scripts/build-site.sh
sudo cp dist/index.html /var/www/thenie/index.html
```

At every level the site keeps working. That is the property the whole design is
built around.

---

## Appendix A — Apache instead of Nginx

If this machine runs Apache as its front door, do Parts 1–4 and 7–13 as written
and swap Parts 5–6 for this.

```bash
sudo vi /etc/apache2/sites-available/thenie.conf
```

```apache
<VirtualHost *:80>
    ServerName thenie.id
    DocumentRoot /var/www/thenie

    <Directory /var/www/thenie>
        Options -Indexes +FollowSymLinks
        AllowOverride None
        Require all granted
    </Directory>

    Header always set X-Robots-Tag        "noindex, nofollow"
    Header always set X-Frame-Options     "SAMEORIGIN"
    Header always set X-Content-Type-Options "nosniff"
    Header always set Referrer-Policy     "strict-origin-when-cross-origin"
    <FilesMatch "index\.html$">
        Header always set Cache-Control "public, max-age=0, must-revalidate"
    </FilesMatch>

    AddOutputFilterByType DEFLATE text/html text/css application/javascript

    ErrorLog  ${APACHE_LOG_DIR}/thenie.error.log
    CustomLog ${APACHE_LOG_DIR}/thenie.access.log combined
</VirtualHost>
```

```bash
sudo a2enmod headers deflate
sudo a2ensite thenie
sudo apache2ctl configtest
sudo systemctl reload apache2
sudo certbot --apache -d thenie.id
```

---

## Appendix B — No domain yet? Serve it on a port

To look at it before DNS is ready, serve it on port **8095** and reach it at
`http://172.236.152.44:8095`.

```nginx
server {
    listen 8095;
    root /var/www/thenie;
    index index.html;
    location / { try_files $uri $uri/ /index.html; }
    add_header X-Robots-Tag "noindex, nofollow" always;
}
```

```bash
sudo nginx -t && sudo systemctl reload nginx
sudo ufw allow 8095/tcp     # only if ufw is active
```

This is **plain HTTP with no certificate** — fine for a look, not for real use.
Remove the block and the firewall rule once the subdomain works:

```bash
sudo ufw delete allow 8095/tcp
```

---

## Appendix C — Purely local preview

No server involved at all:

```bash
python3 -m http.server 8080 --directory /home/dev/projects/thenie_v2/site
```

Open <http://localhost:8080>. Stop it with Ctrl-C.

Related: [[07-fidelity-and-verification]] · [[08-technical-inventory]] · [[09-open-questions]]
