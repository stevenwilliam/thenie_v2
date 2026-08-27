# 17 — Admin UI and RBAC

Until now every write was guarded by one shared bearer token. That is fine for a
single operator and useless the moment two people need different access, or
anyone needs to know who changed a price.

This adds real identity: accounts, roles, permissions, sessions, an audit trail,
and a web UI to use them from.

**http://192.168.88.101:8082/admin/** (dev) · `https://thenie.id/admin/` behind
the Nginx proxy from [[13-production-deployment-runbook]] Appendix E.

---

## 1. The model

**Deny by default.** A route is only reachable through a `requirePerm` gate, so
adding one without a permission makes it unreachable rather than public.

```
admin_users ──< admin_user_roles >── admin_roles ──< admin_role_permissions >── admin_permissions
      │
      ├──< admin_sessions      (token hash, expiry, last seen)
      └──< admin_audit_log     (who, what, when, from where)
```

A user's permissions are the **union** across their roles. Holding two grants
the sum, never the intersection — asserted by a test, because the opposite is a
plausible-looking bug.

### Permissions

| Code | Grants |
|------|--------|
| `menu.read` / `menu.write` / `menu.publish` | see, edit, publish weekly menus |
| `price.read` / `price.write` | plan rates, tier ladders, Kantor bands |
| `rules.read` / `rules.write` | the eight pricing thresholds |
| `content.read` / `content.write` | `sys_parameters` and site content |
| `user.manage` | accounts and roles |
| `audit.read` | the activity log |

The Go constants and the seeded rows are pinned together by
`TestPermissionConstantsMatchTheSeed`. That test earns its place: a permission in
Go but not the database is a route **nobody can ever reach**, and one in the
database but not in Go is a checkbox in the UI that **grants nothing**. Both fail
silently, and both look like "permissions are broken".

### Seeded roles

| Role | Holds |
|------|-------|
| **owner** | everything, including `user.manage` |
| **manager** | everything except `user.manage` |
| **editor** | menu + content write, price/rules read only |
| **viewer** | every `*.read` |

Roles are editable except **owner**, which is refused. It is the floor of the
system — the thing you use to fix a role edit that went wrong — so it is not
something to edit while standing on it.

---

## 2. Getting in

There is no sign-up page and no default password, so the first account is made
from the CLI:

```bash
./server/bin/thenied user create --email you@thenie.id --name "Your Name" --roles owner
```

The password is **prompted for, never passed as an argument** — an argument
lands in shell history and in the process list where any other user on the box
can read it.

```bash
thenied user list
thenied user password --email you@thenie.id      # also revokes that user's sessions
thenied user roles    --email x@y.id --roles manager
thenied user activate | deactivate --email x@y.id
```

Everything after that is doable in the UI.

---

## 3. What the UI does

| Screen | Needs | What it is for |
|--------|-------|----------------|
| **Ringkasan** | — | Revision, plan count, live menu, pricing mode, and the output of `/validate` |
| **Menu Mingguan** | `menu.read` | The editor this whole backend exists for: a week, four plan tabs, days with dishes and gram weights, the Thursday meat flag, publish |
| **Harga** | `price.read` | Plan rates, tier ladders, Kantor bands |
| **Aturan Harga** | `rules.read` | The eight thresholds, with the invariant errors shown inline |
| **Parameter** | `content.read` | The 26 `sys_parameters`, grouped, typed inputs |
| **Pengguna** | `user.manage` | Accounts, roles, permission checkboxes |
| **Log Aktivitas** | `audit.read` | Who changed what, and when |

A screen the user cannot reach is greyed out in the nav. A screen they can read
but not write renders with disabled inputs and a banner saying which permission
is missing.

> **Hiding is a courtesy, not a control.** Every screen the UI hides is also
> refused by the server. Editing `state.perms` in the browser console gains
> exactly nothing but a form that returns 403 — verified below.

### How it is built

Vanilla JS, no build step, no dependencies, embedded in the binary with
`go:embed`. The same choice the rest of this repository makes: it is a handful
of screens over a REST API, and a framework plus a bundler would be more
machinery than the thing it builds. Embedding rather than serving from disk
means a deployment cannot get its web root out of step with its binary.

There is no CDN, no font host and no analytics. The admin tool for a business's
prices should not be reporting to third parties, and a tool that works offline
is a tool that works when a CDN does not.

Every element is built with a `h()` helper that goes through `textContent`;
setting `innerHTML` throws by construction. That is the whole XSS story for this
app.

The modal is a real dialog — `role="dialog"`, `aria-modal`, Escape to close,
focus moved to the first control. The captured site's own modal has none of
that (A11Y-10 in [[06-design-system]]); there was no reason to repeat it.

Colours come from the site's palette, but text and fills use `--maroon-deep`
`#B84B39` rather than `--maroon` `#E1614A`: A11Y-1 and A11Y-2 record that the
lighter coral fails WCAG AA at 3.49:1, and repeating a known defect in a tool
built from scratch would be careless.

---

## 4. Security

| Measure | Why |
|---------|-----|
| **argon2id**, 64 MB / t=3 / p=2 | Carried over from healthy_catering. Password hashing is the last place to be original. |
| **Sessions, not JWTs** | One service, one database. A session revocable the instant someone leaves is worth more than a stateless token valid until it expires. |
| **Only the token's SHA-256 is stored** | A leaked database backup contains no usable session. Plain SHA-256, not argon2: it is a 256-bit random value, so there is no dictionary to slow down. |
| **HttpOnly + Secure + SameSite=Strict** | JavaScript cannot read the cookie. Strict costs nothing here — nothing links into the admin UI from elsewhere. |
| **`X-Admin-Request` required on writes** | A browser cannot set a custom header cross-origin without a CORS preflight this API refuses. With SameSite=Strict that closes CSRF without a token round-trip. |
| **Identical error for every login failure** | Unknown email, wrong password and deactivated account all answer "Email atau kata sandi salah." Distinguishing them turns the login form into an account-enumeration oracle. |
| **A dummy hash on unknown accounts** | Without it, response timing tells an attacker which addresses are registered. |
| **Lockout: 5 failures → 15 minutes** | Makes online guessing pointless without ever permanently locking a real person out. |
| **Deactivation kills live sessions** | Otherwise the person stays logged in until their session happens to expire. |
| **Password change revokes sessions** | A password change means "I no longer trust what was open before". |
| **Last-admin guard** | You cannot delete, deactivate or demote the last account holding `user.manage`. Without it, one click leaves a system nobody can administer and the only way back is hand-written SQL against production. |
| **`X-Frame-Options: DENY`, `Cache-Control: no-store`** | No clickjacking; no stale bundle talking to a newer API. |

### The service token stays

`ADMIN_TOKEN` did not go away. It is now a **machine credential**: full
permissions, exempt from the CSRF header (it is used by scripts, not browsers),
and attributed in the audit log as `service-token` rather than a person.
Removing it would break the deploy scripts and cron jobs documented in
[[15-backend-engine]].

---

## 5. Verified by running

Three accounts were created — `owner`, `editor`, `viewer` — and every route
called as each of them:

```
ACTION                             owner    editor   viewer
GET  /params (content.read)        200      200      200
GET  /pricing-rules (rules.read)   200      200      200
GET  /audit (audit.read)           200      403      200
GET  /users (user.manage)          200      403      403
PUT  /params (content.write)       200      200      403
PUT  /plans rates (price.write)    200      403      403
PUT  /pricing-rules (rules.write)  200      403      403
```

And the security properties:

| Check | Result |
|-------|--------|
| Unauthenticated request | `401 UNAUTHENTICATED` |
| Write without `X-Admin-Request` | `403` — CSRF header refused |
| Wrong password | `Email atau kata sandi salah.` |
| Nonexistent account | **the same message** — no enumeration |
| 5 wrong passwords | 6th attempt refused **even with the correct password** |
| Service token | `200`, resolves to `service-token` with all 11 permissions |
| Delete / deactivate / demote the last admin | refused, with an actionable message |
| Weaken the `owner` role | refused |
| Second owner exists | the demote is then allowed |

The UI was rendered in headless Chrome as each role: the owner sees eleven
permissions and every screen; the editor sees the Harga screen read-only with a
banner and `Pengguna` / `Log Aktivitas` greyed out.

### Two bugs the tests caught

- **`CountUsersWithPermission` broke on an empty exclusion.** The query used
  `? = '' OR u.id <> ?::uuid`, and Postgres evaluates the `''::uuid` cast even
  when the OR would short-circuit past it — so it raised
  *invalid input syntax for type uuid* instead of counting everyone. Now a
  NULLable parameter with `IS DISTINCT FROM`. Production never hit this path,
  but it was one refactor away from disabling the last-admin guard.
- **The CLI's second password prompt hit EOF when piped.** `bufio.NewReader` was
  built per prompt, so the first one buffered past the newline and threw the
  rest away. One shared reader.

---

## 6. Running it

Nothing in the deployment changes: same binary, same port, same Nginx block.
Migration `0009_rbac` adds the tables; `thenied user create` adds the first
account.

```bash
./server/bin/thenied migrate up
./server/bin/thenied user create --email you@thenie.id --name "You" --roles owner
./server/bin/thenied serve
# then open /admin/
```

Expired sessions are swept at start-up and hourly.

### If you lock yourself out

The service token still works, and it holds every permission:

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H 'Content-Type: application/json' -X PUT \
     -d '{"roles":["owner"]}' \
     http://127.0.0.1:8082/api/v1/admin/users/<id>/roles
```

Or from the machine itself: `thenied user roles --email you@thenie.id --roles owner`.

Related: [[15-backend-engine]] · [[16-server-side-pricing]] · [[13-production-deployment-runbook]]
