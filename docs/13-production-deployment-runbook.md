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
                       └──▶  thenie.sunshinefood.co.id  ◀── WE ADD THIS
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
| `thenie.sunshinefood.co.id` | the subdomain to serve on | *(suggested — change everywhere below if you use another)* |
| `SERVER_IP` | your machine's public IP | `172.236.152.44` |
| `SSH_USER` | the user you log into the server as | `appuser` |
| `SSH_PORT` | the SSH port — **this machine uses 30022, not 22** | `30022` |
| `you@sunshinefood.co.id` | email for the TLS certificate expiry warnings | — |

There are **no passwords and no secrets** in this deployment.

> **Not sure about the subdomain?** See Q-14 in [[09-open-questions]]. Anything
> works — just replace `thenie.sunshinefood.co.id` consistently everywhere below.

---

## Part 1 — Point the subdomain at the machine (DNS)

In your registrar's DNS panel, add **one A record**:

| Type | Name | Value |
|------|------|-------|
| A | `thenie` | `SERVER_IP` |

If you use **Cloudflare**, either grey cloud (DNS only) or orange cloud
(Proxied) is fine here — this is a plain static file with no presigned uploads,
so the problem that forces `cdn.` to be unproxied does not apply.

Wait a few minutes, then verify **from your own computer**:

```bash
nslookup thenie.sunshinefood.co.id
```

It must print `SERVER_IP`. **Do not continue until it does** — certbot cannot
issue a certificate otherwise.

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
sudo chown SSH_USER:SSH_USER /opt/thenie_v2
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
server {
    listen 80;
    listen [::]:80;
    server_name thenie.sunshinefood.co.id;

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

    # This is a mockup. Keep it out of Google so it cannot be confused with, or
    # outrank, the real site. Remove this line when it goes properly live.
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
curl -sI http://thenie.sunshinefood.co.id | head -5
```

Expect `HTTP/1.1 200 OK` and `Content-Type: text/html`.

---

## Part 7 — HTTPS

Install certbot if this machine does not have it yet:

```bash
sudo apt update
sudo apt -y install certbot python3-certbot-nginx
```

Issue the certificate:

```bash
sudo certbot --nginx -d thenie.sunshinefood.co.id --email you@sunshinefood.co.id --agree-tos --no-eff-email
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
curl -sI https://thenie.sunshinefood.co.id | head -12
```

Check for:

- `HTTP/2 200`
- `content-type: text/html`
- `x-robots-tag: noindex, nofollow`
- `x-frame-options: SAMEORIGIN`
- `cache-control: public, max-age=0, must-revalidate`

Confirm compression is applied:

```bash
curl -sI -H "Accept-Encoding: gzip" https://thenie.sunshinefood.co.id | grep -i content-encoding
```

Expect `content-encoding: gzip`.

Confirm the served bytes are the real thing — decompressed, this must be the
capture hash again:

```bash
curl -s --compressed https://thenie.sunshinefood.co.id | sha256sum
```

```
9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49
```

**Now confirm you broke nothing.** Open your existing PHP site in a browser, and:

```bash
curl -sI https://meals.sunshinefood.co.id | head -3
curl -s  https://api.sunshinefood.co.id/health
```

Finally, open **https://thenie.sunshinefood.co.id** on a real phone. It is a
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
curl -s --compressed https://thenie.sunshinefood.co.id | sha256sum
```

---

## Part 12 — Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `nginx -t` fails | Typo in the config | Read the line number it prints; fix; re-test. Never reload on a failed test. |
| **404 Not Found** | File missing or wrong root | `ls -la /var/www/thenie/index.html` |
| **403 Forbidden** | Permissions | `sudo chown -R www-data:www-data /var/www/thenie && sudo chmod 644 /var/www/thenie/index.html` |
| Certbot fails | DNS not resolving yet | `nslookup thenie.sunshinefood.co.id` must return `SERVER_IP`; wait and retry. |
| Serves the **wrong site** | Another server block claims the name, or this one is unreachable | `sudo nginx -T \| grep -n "server_name"` |
| Page loads but photos missing | Truncated file | Compare `sha256sum` against Part 4. Re-copy. |
| Very slow first load | The 4.6 MB payload | Expected. See Q-15 in [[09-open-questions]]. |
| Old version still showing | Browser cache | Hard-refresh (Ctrl-Shift-R). Confirm `cache-control` in Part 8. |
| Existing PHP site broke | You edited its config | `sudo nginx -T` and compare. This runbook never touches it. |

---

## Part 13 — Security notes

This is about as small an attack surface as a website has: **one static file, no
process, no database, no input reaching the server**. Everything the customer
types stays in their browser and goes to WhatsApp (see [[08-technical-inventory]]).

Still worth knowing:

- **The `noindex` header is deliberate.** A mockup competing with the real
  business in search results is a genuine risk. Remove it only on purpose.
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

## Appendix A — Apache instead of Nginx

If this machine runs Apache as its front door, do Parts 1–4 and 7–13 as written
and swap Parts 5–6 for this.

```bash
sudo vi /etc/apache2/sites-available/thenie.conf
```

```apache
<VirtualHost *:80>
    ServerName thenie.sunshinefood.co.id
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
sudo certbot --apache -d thenie.sunshinefood.co.id
```

---

## Appendix B — No domain yet? Serve it on a port

To look at it before DNS is ready, serve it on port **8095** and reach it at
`http://SERVER_IP:8095`.

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
