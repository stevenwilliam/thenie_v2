# 13 — Production Deployment Runbook — Static Site on the Existing Nginx

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
                                     └── /var/www/thenie/index.html   (one 4.6 MB file)
```

- **No Docker.** Nothing runs. Nginx reads a file off the disk and sends it.
- **No database.** The mockup has no backend at all — see [[08-technical-inventory]].
- **No ports.** Nothing new listens on anything.
- **No secrets.** There is nothing secret to configure.

Because nothing runs, there is nothing to crash, nothing to restart, and nothing
to monitor. The only failure modes are "Nginx config typo" and "certificate
expired".

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

Check you have room for the file (it is 4.6 MB, so this is a formality):

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

Publish the file:

```bash
sudo cp /opt/thenie_v2/site/index.html /var/www/thenie/index.html
```

### Option B — copy straight from your laptop

Run this **on your own machine, not on the server**. This is the single most
common mistake in this runbook: if your prompt reads `appuser@…:/var/www/thenie$`
you are on the server, and `scp` will fail with

```
scp: stat local "/home/dev/projects/thenie_v2/site/index.html": No such file or directory
```

because that path exists only on your dev machine. Type `exit` first, then:

```bash
scp -P 30022 /home/dev/projects/thenie_v2/site/index.html appuser@172.236.152.44:/tmp/index.html
```

Then back **on the server**:

```bash
sudo mv /tmp/index.html /var/www/thenie/index.html
```

### Option C — pull it from the live site, on the server

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
```

It **must** print:

```
9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49
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

    # The page is one 4.6 MB HTML file replaced wholesale on each deploy, so it
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
    # no Open Graph tags and no <h1> (see docs/08-technical-inventory.md), so it
    # will index thinly and share as a bare link. See Q-13 in
    # docs/09-open-questions.md.
    add_header X-Robots-Tag "noindex, nofollow" always;

    # Baseline security headers. No framing, no MIME sniffing, no referrer leak.
    add_header X-Frame-Options        "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy        "strict-origin-when-cross-origin" always;

    # Everything resolves to the single page. There are no other files and no
    # client-side routes, but this keeps a stray path from returning a bare 404.
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Compression. The inline CSS/JS/markup compresses well; the base64 JPEGs
    # are already compressed and barely shrink, so expect roughly 4.6 MB to
    # become roughly 4.4 MB. That is a limit of the mirrored file, not of the
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
> (clean), then used to actually serve the real 4.6 MB file while the response
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

Confirm the served bytes are the real thing — decompressed, this must be the
capture hash again:

```bash
curl -s --compressed https://thenie.id | sha256sum
```

```
9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49
```

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

If you used **Option A**:

```bash
cd /opt/thenie_v2
git pull
sudo cp /opt/thenie_v2/site/index.html /var/www/thenie/index.html
sudo chown www-data:www-data /var/www/thenie/index.html
sudo chmod 644 /var/www/thenie/index.html
sha256sum /var/www/thenie/index.html
```

If you used **Option B**, repeat the `scp` from Part 4.

**No Nginx reload is needed** — Nginx reads the file from disk on each request.
The `must-revalidate` header means returning visitors pick the new file up
immediately.

Always keep a copy of the previous version first:

```bash
sudo cp /var/www/thenie/index.html /var/www/thenie/index.html.bak
```

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

# confirm the served file is still the exact capture
curl -s --compressed https://thenie.id | sha256sum
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
| Page loads but photos missing | Truncated file | Compare `sha256sum` against Part 4. Re-copy. |
| Very slow first load | The 4.6 MB payload | Expected. See Q-15 in [[09-open-questions]]. |
| Old version still showing | Browser cache | Hard-refresh (Ctrl-Shift-R). Confirm `cache-control` in Part 8. |
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
```

Must print `9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49`.

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
