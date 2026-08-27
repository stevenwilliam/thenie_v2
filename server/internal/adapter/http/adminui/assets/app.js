/* Thenie admin UI.
 *
 * Vanilla JS, no build step, no dependencies — the same choice the rest of this
 * repository makes. It is a handful of screens over a REST API; a framework
 * plus a bundler would be more machinery than the thing it builds.
 *
 * THE RULE THAT MATTERS: this file decides what to DRAW. It never decides what
 * is ALLOWED. Every screen it hides is also refused by the server, so a user
 * who edits `state.perms` in the console gains exactly nothing but a form that
 * returns 403. Hiding is a courtesy, not a control.
 */
'use strict';

const API = '/api/v1/admin';

const state = {
  user: null,
  perms: new Set(),
  route: 'dashboard',
  data: {},
};

/* ---------- tiny DOM helpers ---------- */

// h('div.card', {onclick}, child, child) — everything is built from this rather
// than from innerHTML, so no value from the API or a form can ever be parsed as
// markup. That is the whole XSS story for this app.
function h(spec, props, ...kids) {
  const [tag, ...classes] = String(spec).split('.');
  const el = document.createElement(tag || 'div');
  if (classes.length) el.className = classes.join(' ');
  for (const [k, v] of Object.entries(props || {})) {
    if (v === null || v === undefined || v === false) continue;
    if (k.startsWith('on') && typeof v === 'function') el.addEventListener(k.slice(2), v);
    else if (k === 'text') el.textContent = v;
    else if (k === 'html') throw new Error('refusing to set innerHTML');
    else if (k === 'value') el.value = v;
    else if (k === 'checked') el.checked = !!v;
    else if (k === 'disabled') el.disabled = !!v;
    else el.setAttribute(k, v);
  }
  for (const kid of kids.flat(9)) {
    if (kid === null || kid === undefined || kid === false) continue;
    el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  }
  return el;
}
const $ = (sel, root = document) => root.querySelector(sel);

function toast(msg, kind = 'ok') {
  const box = $('#toast');
  const el = h('div.banner.' + kind, { text: msg });
  box.append(el);
  setTimeout(() => el.remove(), kind === 'err' ? 8000 : 3500);
}

function can(p) { return state.perms.has(p); }

/* ---------- API ---------- */

async function api(path, opts = {}) {
  const init = {
    method: opts.method || 'GET',
    credentials: 'same-origin',
    headers: { 'Accept': 'application/json' },
  };
  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body);
    init.headers['Content-Type'] = 'application/json';
  }
  // Required by the server on every state-changing request. A browser cannot
  // set a custom header cross-origin without a CORS preflight this API refuses,
  // so this plus SameSite=Strict is what closes CSRF.
  if (init.method !== 'GET' && init.method !== 'HEAD') init.headers['X-Admin-Request'] = '1';

  const res = await fetch(API + path, init);
  if (res.status === 204) return null;
  let body = null;
  try { body = await res.json(); } catch (_) { /* empty body */ }
  if (!res.ok) {
    const err = new Error((body && body.error && body.error.message) || ('HTTP ' + res.status));
    err.status = res.status;
    err.code = body && body.error && body.error.code;
    err.details = body && body.error && body.error.details;
    throw err;
  }
  return body;
}

/* ---------- routing ---------- */

const ROUTES = [
  { id: 'dashboard', label: 'Ringkasan',   icon: '◉', perm: null },
  { id: 'menu',      label: 'Menu Mingguan', icon: '🍱', perm: 'menu.read' },
  { id: 'prices',    label: 'Harga',       icon: '💰', perm: 'price.read' },
  { id: 'rules',     label: 'Aturan Harga', icon: '⚖️', perm: 'rules.read' },
  { id: 'params',    label: 'Parameter',   icon: '⚙️', perm: 'content.read' },
  { id: 'users',     label: 'Pengguna',    icon: '👥', perm: 'user.manage' },
  { id: 'audit',     label: 'Log Aktivitas', icon: '🧾', perm: 'audit.read' },
];

function go(route) {
  state.route = route;
  location.hash = route;
  render();
}

/* ---------- boot ---------- */

async function boot() {
  try {
    const me = await api('/auth/me');
    state.user = me.user;
    state.perms = new Set((me.user && me.user.permissions) || []);
  } catch (err) {
    state.user = null;
  }
  const hash = location.hash.replace('#', '');
  if (hash && ROUTES.some(r => r.id === hash)) state.route = hash;
  render();
}

window.addEventListener('hashchange', () => {
  const hash = location.hash.replace('#', '');
  if (hash && hash !== state.route && ROUTES.some(r => r.id === hash)) { state.route = hash; render(); }
});

/* ---------- render ---------- */

function render() {
  const root = $('#root');
  root.replaceChildren(state.user ? shell() : loginScreen());
}

function loginScreen() {
  const email = h('input', { type: 'email', id: 'lg-email', autocomplete: 'username', required: 'required' });
  const pass = h('input', { type: 'password', id: 'lg-pass', autocomplete: 'current-password', required: 'required' });
  const msg = h('div');
  const btn = h('button.btn', { type: 'submit', text: 'Masuk' });

  const form = h('form', {
    onsubmit: async (e) => {
      e.preventDefault();
      msg.replaceChildren();
      btn.disabled = true;
      try {
        const r = await api('/auth/login', { method: 'POST', body: { email: email.value, password: pass.value } });
        state.user = r.user;
        state.perms = new Set(r.user.permissions || []);
        render();
      } catch (err) {
        msg.replaceChildren(h('div.banner.err', { text: err.message }));
        pass.value = '';
        pass.focus();
      } finally {
        btn.disabled = false;
      }
    },
  },
    h('div', {}, h('label', { for: 'lg-email', text: 'Email' }), email),
    h('div', { style: 'margin-top:12px' }, h('label', { for: 'lg-pass', text: 'Kata sandi' }), pass),
    h('div', { style: 'margin-top:16px' }, btn),
  );

  return h('div.login', {},
    h('div.card', {},
      h('div.brand', {}, h('span.dot'), h('span', {}, 'Thenie Admin', h('small', { text: 'Panel Konfigurasi' }))),
      msg, form,
      h('p.muted', { style: 'margin:14px 0 0', text: 'Belum punya akun? Minta administrator membuatkannya.' }),
    ));
}

function shell() {
  const nav = h('div.nav', {}, ROUTES.map(r => {
    const allowed = !r.perm || can(r.perm);
    return h('button', {
      'aria-current': state.route === r.id ? 'page' : null,
      disabled: !allowed,
      title: allowed ? null : 'Anda tidak punya akses ke bagian ini',
      onclick: () => allowed && go(r.id),
    }, h('span', { 'aria-hidden': 'true', text: r.icon }), r.label);
  }));

  const side = h('aside.side', {},
    h('div.brand', {}, h('span.dot'), h('span', {}, 'Thenie Admin', h('small', { text: 'Panel Konfigurasi' }))),
    nav,
    h('div.foot', {},
      h('b', { text: state.user.name || state.user.email }),
      h('div', { text: state.user.is_service ? 'service token' : (state.user.email || '') }),
      h('div', { style: 'margin-top:6px' }, h('span.pill.grey', { text: state.perms.size + ' izin' })),
      h('button.btn.ghost.sm', {
        style: 'margin-top:10px',
        text: 'Keluar',
        onclick: async () => { await api('/auth/logout', { method: 'POST' }); state.user = null; state.perms = new Set(); render(); },
      })));

  const main = h('main.main', {}, h('div', { id: 'view' }, h('p.muted', { text: 'Memuat…' })));
  const app = h('div.app', {}, side, main);
  drawView();
  return app;

  function drawView() {
    const view = () => $('#view');
    const draw = VIEWS[state.route] || VIEWS.dashboard;
    Promise.resolve(draw()).then(node => {
      const v = view();
      if (v) v.replaceChildren(node);
    }).catch(err => {
      const v = view();
      if (v) v.replaceChildren(h('div.banner.err', { text: err.message }));
    });
  }
}

function refresh() { render(); }

/* ---------- views ---------- */

const VIEWS = {};

VIEWS.dashboard = async () => {
  const wrap = h('div', {},
    h('div.head', {}, h('div', {}, h('h1', { text: 'Ringkasan' }),
      h('p', { text: 'Status konfigurasi yang sedang dipakai situs.' }))));

  let cfg = null, problems = null;
  try { cfg = await fetch('/api/v1/site-config').then(r => r.json()); } catch (_) { /* offline */ }
  if (can('content.read')) {
    try { problems = await api('/validate'); } catch (err) { problems = { ok: false, problems: [err.message] }; }
  }

  if (problems) {
    wrap.append(problems.ok
      ? h('div.banner.ok', { text: '✓ Semua konfigurasi valid (revisi ' + problems.revision + ').' })
      : h('div.banner.err', {}, h('b', { text: problems.problems.length + ' masalah ditemukan:' }),
          h('ul', {}, problems.problems.map(p => h('li', { text: p })))));
  }

  if (cfg) {
    const cur = cfg.menu && cfg.menu.current;
    const next = cfg.menu && cfg.menu.next;
    wrap.append(h('div.grid.four', {},
      stat('Revisi konten', String(cfg.revision)),
      stat('Paket langganan', String((cfg.plans || []).length)),
      stat('Siklus menu terbit', String((cfg.menu && cfg.menu.cycles || []).length)),
      stat('Mode harga', cfg.params ? (cfg.params['order.pricing_mode'] || '—') : '—'),
    ));
    wrap.append(h('div.card', {},
      h('h2', { text: 'Menu yang sedang tayang' }),
      h('p', {}, h('b', { text: 'Minggu ini: ' }), cur ? cur.label : 'belum ada'),
      h('p', {}, h('b', { text: 'Minggu depan: ' }), next ? next.label : 'belum ada'),
      can('menu.read') ? h('button.btn.sm', { text: 'Kelola menu →', onclick: () => go('menu') }) : null,
    ));
  } else {
    wrap.append(h('div.banner.warn', { text: 'Tidak bisa membaca /api/v1/site-config.' }));
  }
  return wrap;

  function stat(label, value) {
    return h('div.card', { style: 'margin:0' },
      h('div.muted', { text: label }),
      h('div', { style: 'font-size:1.6rem;font-weight:700;margin-top:2px', text: value }));
  }
};

/* ----- menu editor: the screen this whole thing exists for ----- */

VIEWS.menu = async () => {
  const cfg = await fetch('/api/v1/site-config').then(r => r.json());
  const plans = cfg.plans || [];
  const cycles = (cfg.menu && cfg.menu.cycles || []).slice().reverse();
  const writable = can('menu.write');

  const wrap = h('div', {});
  wrap.append(h('div.head', {},
    h('div', {}, h('h1', { text: 'Menu Mingguan' }),
      h('p', { text: 'Satu siklus adalah satu minggu. Menyimpan menulis seluruh minggu sekaligus — kirim minggu itu seperti seharusnya tampil, bukan hanya bagian yang berubah.' })),
    writable ? h('button.btn', { text: '+ Minggu baru', onclick: () => openEditor(null) }) : null));

  if (!writable) wrap.append(h('div.banner.warn', { text: 'Anda hanya bisa melihat. Perlu izin menu.write untuk mengubah.' }));

  const table = h('table', {},
    h('thead', {}, h('tr', {},
      h('th', { text: 'Minggu' }), h('th', { text: 'Periode' }), h('th', { text: 'Label' }),
      h('th', { text: 'Menu' }), h('th', { text: '' }))),
    h('tbody', {}, cycles.length ? cycles.map(cycleRow) : h('tr', {}, h('td', { colspan: '5', text: 'Belum ada siklus menu.' }))));
  wrap.append(h('div.card', {}, table));
  return wrap;

  function cycleRow(c) {
    const counts = Object.entries(c.days || {}).map(([slug, d]) => slug + ' ' + d.length).join(' · ');
    return h('tr', {},
      h('td', {}, h('b', { text: c.iso_year + '-W' + String(c.iso_week).padStart(2, '0') })),
      h('td', { text: c.starts_on + ' → ' + c.ends_on }),
      h('td', { text: c.label }),
      h('td', {}, h('span.muted', { text: counts || 'kosong' })),
      h('td', {}, h('div.row', {},
        h('button.btn.ghost.sm', { text: writable ? 'Ubah' : 'Lihat', onclick: () => openEditor(c) }),
        writable && can('menu.publish') ? h('button.btn.ghost.sm', {
          text: 'Tarik', title: 'Sembunyikan dari situs',
          onclick: () => act('/menu/cycles/' + c.iso_year + '/' + c.iso_week + '/unpublish', 'POST', 'Siklus ditarik'),
        }) : null,
        writable ? h('button.btn.danger.sm', {
          text: 'Hapus',
          onclick: () => {
            if (!confirm('Hapus siklus ' + c.label + '? Menu minggu ini akan hilang dari situs.')) return;
            act('/menu/cycles/' + c.iso_year + '/' + c.iso_week, 'DELETE', 'Siklus dihapus');
          },
        }) : null)));
  }

  async function act(path, method, ok) {
    try { await api(path, { method }); toast(ok); refresh(); }
    catch (err) { toast(err.message, 'err'); }
  }

  function openEditor(cycle) {
    const isNew = !cycle;
    const model = {
      iso_year: cycle ? cycle.iso_year : new Date().getFullYear(),
      iso_week: cycle ? cycle.iso_week : isoWeekOf(new Date()) + 1,
      starts_on: cycle ? cycle.starts_on : nextMonday(),
      ends_on: cycle ? cycle.ends_on : addDays(nextMonday(), 4),
      label: cycle ? cycle.label : '',
      publish: true,
      days: JSON.parse(JSON.stringify((cycle && cycle.days) || {})),
    };
    let activePlan = plans.length ? plans[0].slug : null;

    const dayHost = h('div.daygrid');
    const tabs = h('div.plantabs', {}, plans.map(p => h('button', {
      type: 'button', 'aria-pressed': p.slug === activePlan ? 'true' : 'false',
      text: p.name, onclick: () => { activePlan = p.slug; drawTabs(); drawDays(); },
    })));

    function drawTabs() {
      Array.from(tabs.children).forEach((b, i) =>
        b.setAttribute('aria-pressed', plans[i].slug === activePlan ? 'true' : 'false'));
    }

    function drawDays() {
      const plan = plans.find(p => p.slug === activePlan);
      const list = model.days[activePlan] || (model.days[activePlan] = []);
      dayHost.replaceChildren(
        h('p.muted', {}, 'Tanggal harus berada di dalam periode siklus.',
          plan && !plan.delivers_sunday ? ' ' + plan.name + ' tidak mengantar hari Minggu, jadi tanggal Minggu akan ditolak.' : ''),
        ...list.map((d, i) => dayCard(d, i, list)),
        writable ? h('button.btn.ghost.sm', {
          text: '+ Tambah hari',
          onclick: () => { list.push({ date: model.starts_on, kcal: 0, is_meat_day: false, items: [{ name: '', grams: 0 }] }); drawDays(); },
        }) : null);
    }

    function dayCard(d, idx, list) {
      const items = h('div.items', {}, (d.items || []).map((it, j) => h('div.item', {},
        h('input', { type: 'text', value: it.name, placeholder: 'Nama menu', disabled: !writable,
          oninput: e => { it.name = e.target.value; } }),
        h('input', { type: 'number', value: it.grams || '', placeholder: 'gram', min: '0', disabled: !writable,
          oninput: e => { it.grams = parseInt(e.target.value, 10) || 0; } }),
        writable ? h('button.btn.ghost.sm', { text: '×', title: 'Hapus item',
          onclick: () => { d.items.splice(j, 1); drawDays(); } }) : null)));

      return h('div.day', {},
        h('header', {},
          h('div.row', {},
            h('input', { type: 'date', value: d.date, disabled: !writable,
              oninput: e => { d.date = e.target.value; } }),
            h('label.chk', {}, h('input', { type: 'checkbox', checked: d.is_meat_day, disabled: !writable,
              onchange: e => { d.is_meat_day = e.target.checked; } }), 'Hari daging ⭐'),
            h('input', { type: 'number', value: d.kcal || '', placeholder: 'kkal', min: '0',
              style: 'width:90px', disabled: !writable,
              oninput: e => { d.kcal = parseInt(e.target.value, 10) || 0; } })),
          writable ? h('button.btn.ghost.sm', { text: 'Hapus hari',
            onclick: () => { list.splice(idx, 1); drawDays(); } }) : null),
        items,
        writable ? h('button.btn.ghost.sm', { style: 'margin-top:6px', text: '+ item',
          onclick: () => { (d.items || (d.items = [])).push({ name: '', grams: 0 }); drawDays(); } }) : null);
    }

    const err = h('div');
    const body = h('div', {},
      err,
      h('div.grid.two', {},
        field('Tahun', h('input', { type: 'number', value: model.iso_year, disabled: !writable || !isNew,
          oninput: e => { model.iso_year = parseInt(e.target.value, 10); } })),
        field('Minggu ke-', h('input', { type: 'number', min: '1', max: '53', value: model.iso_week,
          disabled: !writable || !isNew, oninput: e => { model.iso_week = parseInt(e.target.value, 10); } })),
        field('Mulai', h('input', { type: 'date', value: model.starts_on, disabled: !writable,
          oninput: e => { model.starts_on = e.target.value; } })),
        field('Sampai', h('input', { type: 'date', value: model.ends_on, disabled: !writable,
          oninput: e => { model.ends_on = e.target.value; } }))),
      field('Label (tampil di situs)', h('input', { type: 'text', value: model.label, disabled: !writable,
        placeholder: 'Minggu ke-36 · 31 Agustus – 4 September 2026',
        oninput: e => { model.label = e.target.value; } })),
      h('label.chk', { style: 'margin-top:10px' },
        h('input', { type: 'checkbox', checked: model.publish, disabled: !writable || !can('menu.publish'),
          onchange: e => { model.publish = e.target.checked; } }), 'Terbitkan ke situs'),
      h('hr', { style: 'border:0;border-top:1px solid var(--line);margin:16px 0' }),
      tabs, dayHost);

    drawDays();

    const save = h('button.btn', { text: 'Simpan minggu', disabled: !writable,
      onclick: async () => {
        err.replaceChildren();
        save.disabled = true;
        try {
          await api('/menu/cycles', { method: 'PUT', body: model });
          toast('Menu tersimpan');
          closeModal();
          refresh();
        } catch (e2) {
          err.replaceChildren(h('div.banner.err', { text: e2.message }));
          save.disabled = false;
        }
      } });

    openModal(isNew ? 'Minggu baru' : ('Ubah ' + model.label), body, [save]);
  }
};

function field(label, input) {
  return h('div', {}, h('label', { text: label }), input);
}

/* ----- prices ----- */

VIEWS.prices = async () => {
  const cfg = await fetch('/api/v1/site-config').then(r => r.json());
  const writable = can('price.write');
  const wrap = h('div', {}, h('div.head', {}, h('div', {},
    h('h1', { text: 'Harga' }),
    h('p', { text: 'Harga dalam Rupiah penuh. Aturan tier menolak tabel yang lebih murah untuk komitmen lebih panjang — itu bentuk yang diandalkan kalkulator situs.' }))));

  if (!writable) wrap.append(h('div.banner.warn', { text: 'Anda hanya bisa melihat. Perlu izin price.write untuk mengubah.' }));

  const KEYS = [
    ['daily', 'Harian'], ['weeklyPerDay', 'Mingguan/hari'], ['monthlyPerDay', 'Bulanan/hari'],
    ['flexiWeeklyPerDay', 'Flexi Mingguan/hari'], ['flexiMonthlyPerDay', 'Flexi Bulanan/hari'],
  ];

  for (const plan of cfg.plans || []) {
    const draft = Object.assign({}, plan.rates);
    const msg = h('div');
    const inputs = KEYS.map(([k, label]) => field(label,
      h('input', { type: 'number', min: '1', value: draft[k], disabled: !writable,
        oninput: e => { draft[k] = parseInt(e.target.value, 10) || 0; } })));
    const btn = h('button.btn.sm', { text: 'Simpan', disabled: !writable, onclick: async () => {
      msg.replaceChildren(); btn.disabled = true;
      try { await api('/plans/' + plan.slug + '/rates', { method: 'PUT', body: draft }); toast('Harga ' + plan.name + ' tersimpan'); }
      catch (e) { msg.replaceChildren(h('div.banner.err', { text: e.message })); }
      finally { btn.disabled = !writable; }
    } });
    wrap.append(h('div.card', {},
      h('h2', {}, plan.name, ' ', h('span.pill.grey', { text: plan.slug })), msg,
      h('div.grid.four', {}, inputs), h('div', { style: 'margin-top:10px' }, btn)));
  }

  for (const prod of cfg.tier_products || []) {
    for (const pkg of prod.packages || []) {
      const bands = JSON.parse(JSON.stringify(pkg.tiers || []));
      const msg = h('div');
      const rows = h('tbody', {}, bands.map((b, i) => h('tr', {},
        h('td', {}, h('input', { type: 'number', value: b.min, disabled: !writable, oninput: e => { b.min = +e.target.value; } })),
        h('td', {}, h('input', { type: 'number', value: b.max, disabled: !writable, oninput: e => { b.max = +e.target.value; } })),
        h('td', {}, h('input', { type: 'number', value: b.price, disabled: !writable, oninput: e => { b.price = +e.target.value; } })))));
      const btn = h('button.btn.sm', { text: 'Simpan', disabled: !writable, onclick: async () => {
        msg.replaceChildren(); btn.disabled = true;
        try {
          await api('/tier-products/' + prod.slug + '/packages/' + encodeURIComponent(pkg.name) + '/prices',
            { method: 'PUT', body: { min_qty: prod.min_qty, tiers: bands } });
          toast(prod.name + ' / ' + pkg.name + ' tersimpan');
        } catch (e) { msg.replaceChildren(h('div.banner.err', { text: e.message })); }
        finally { btn.disabled = !writable; }
      } });
      wrap.append(h('div.card', {},
        h('h2', {}, prod.name, ' — ', pkg.name),
        h('p.muted', { text: 'Minimal ' + prod.min_qty + ' ' + prod.unit + '. Tingkatan harus menyambung tanpa celah.' }),
        msg,
        h('table', {}, h('thead', {}, h('tr', {}, h('th', { text: 'Dari' }), h('th', { text: 'Sampai' }), h('th', { text: 'Harga' }))), rows),
        h('div', { style: 'margin-top:10px' }, btn)));
    }
  }

  const kantor = cfg.kantor || {};
  for (const [grade, periods] of Object.entries(kantor.rates || {})) {
    for (const [period, bands0] of Object.entries(periods)) {
      const bands = JSON.parse(JSON.stringify(bands0));
      const msg = h('div');
      const rows = h('tbody', {}, bands.map(b => h('tr', {},
        h('td', {}, h('input', { type: 'number', value: b.min, disabled: !writable, oninput: e => { b.min = +e.target.value; } })),
        h('td', {}, h('input', { type: 'number', value: b.max, disabled: !writable, oninput: e => { b.max = +e.target.value; } })),
        h('td', {}, h('input', { type: 'number', value: b.price, disabled: !writable, oninput: e => { b.price = +e.target.value; } })))));
      const btn = h('button.btn.sm', { text: 'Simpan', disabled: !writable, onclick: async () => {
        msg.replaceChildren(); btn.disabled = true;
        try { await api('/kantor/' + grade + '/' + period + '/rates', { method: 'PUT', body: { bands } });
          toast('Kantor ' + grade + '/' + period + ' tersimpan'); }
        catch (e) { msg.replaceChildren(h('div.banner.err', { text: e.message })); }
        finally { btn.disabled = !writable; }
      } });
      wrap.append(h('div.card', {},
        h('h2', {}, 'Catering Kantor — ', grade, ' / ', period),
        h('p.muted', { text: (kantor.periods || {})[period] + ' hari kerja per paket. Harga per pax per hari.' }),
        msg,
        h('table', {}, h('thead', {}, h('tr', {}, h('th', { text: 'Pax dari' }), h('th', { text: 'Sampai' }), h('th', { text: 'Rp/pax/hari' }))), rows),
        h('div', { style: 'margin-top:10px' }, btn)));
    }
  }
  return wrap;
};

/* ----- pricing rules ----- */

VIEWS.rules = async () => {
  const r = await api('/pricing-rules');
  const draft = Object.assign({}, r.pricing_rules);
  const writable = can('rules.write');

  const FIELDS = [
    ['weekly_min_days', 'Minimal hari untuk Mingguan', 'Tanggal berturutan sebanyak ini ke atas dapat harga Mingguan.'],
    ['monthly_min_days', 'Minimal hari untuk Bulanan', 'Tanggal berturutan sebanyak ini ke atas dapat harga Bulanan.'],
    ['consecutive_flexi_weekly_max_days', 'Batas Flexi Mingguan (berturutan)', 'Di atas ini tapi belum Bulanan → harga penuh per hari.'],
    ['flexi_monthly_max_span_days', 'Rentang maks Flexi Bulanan', 'Tanggal tersebar masih dapat diskon bulanan bila muat dalam rentang ini.'],
    ['weekday_routine_max_span_days', 'Rentang maks rutin Sen–Jum/Sab (Bulanan)', 'Rutin hari kerja dihitung Bulanan penuh bila muat di sini.'],
    ['weekly_routine_max_span_days', 'Rentang maks rutin antar minggu', 'Order yang nyambung 2 minggu dihitung Mingguan bila muat di sini.'],
    ['weekly_routine_min_days_in_week', 'Minimal hari terisi dalam 1 minggu', 'Salah satu minggu kalender harus memuat sebanyak ini.'],
    ['pax_table_max_pax', 'Batas tabel harga per pax', 'Di atas ini harga lanjut linear dari baris terakhir.'],
  ];

  const msg = h('div');
  const btn = h('button.btn', { text: 'Simpan aturan', disabled: !writable, onclick: async () => {
    msg.replaceChildren(); btn.disabled = true;
    try { await api('/pricing-rules', { method: 'PUT', body: draft }); toast('Aturan harga tersimpan'); }
    catch (e) { msg.replaceChildren(h('div.banner.err', { text: e.message })); }
    finally { btn.disabled = !writable; }
  } });

  return h('div', {},
    h('div.head', {}, h('div', {}, h('h1', { text: 'Aturan Harga' }),
      h('p', { text: 'Ini bukan mengubah harga — ini mengubah harga mana yang berlaku. Kombinasi yang membuat satu tingkat tidak pernah tercapai akan ditolak.' }))),
    !writable ? h('div.banner.warn', { text: 'Anda hanya bisa melihat. Perlu izin rules.write untuk mengubah.' }) : null,
    msg,
    h('div.card', {}, h('div.grid.two', {}, FIELDS.map(([k, label, hint]) =>
      h('div', {}, h('label', { text: label }),
        h('input', { type: 'number', min: '1', value: draft[k], disabled: !writable,
          oninput: e => { draft[k] = parseInt(e.target.value, 10) || 0; } }),
        h('div.muted', { style: 'margin-top:3px', text: hint }))))),
    h('div', {}, btn));
};

/* ----- parameters ----- */

VIEWS.params = async () => {
  const r = await api('/params');
  const writable = can('content.write');
  const groups = {};
  for (const p of r.params || []) (groups[p.group] || (groups[p.group] = [])).push(p);

  const wrap = h('div', {}, h('div.head', {}, h('div', {},
    h('h1', { text: 'Parameter' }),
    h('p', { text: 'Nilai yang bisa berubah tanpa mengubah kode: kontak, rekening, ongkir, batas waktu, mode harga.' }))));
  if (!writable) wrap.append(h('div.banner.warn', { text: 'Anda hanya bisa melihat. Perlu izin content.write untuk mengubah.' }));

  for (const [group, items] of Object.entries(groups)) {
    wrap.append(h('div.card', {}, h('h2', { text: group }),
      h('table', {}, h('tbody', {}, items.map(p => {
        const input = p.value_type === 'bool'
          ? h('select', { disabled: !writable }, h('option', { value: 'true', text: 'true', selected: p.value === 'true' ? 'selected' : null }),
              h('option', { value: 'false', text: 'false', selected: p.value === 'false' ? 'selected' : null }))
          : h('input', { type: p.value_type === 'int' ? 'number' : 'text', value: p.value, disabled: !writable });
        const btn = h('button.btn.sm', { text: 'Simpan', disabled: !writable, onclick: async () => {
          btn.disabled = true;
          try { await api('/params/' + encodeURIComponent(p.key), { method: 'PUT', body: { value: String(input.value) } }); toast(p.key + ' tersimpan'); }
          catch (e) { toast(e.message, 'err'); }
          finally { btn.disabled = !writable; }
        } });
        return h('tr', {},
          h('td', {}, h('b', { text: p.label }), h('div.muted', { text: p.key }),
            p.description ? h('div.muted', { text: p.description }) : null),
          h('td', { style: 'width:280px' }, input),
          h('td', { style: 'width:90px' }, btn));
      })))));
  }
  return wrap;
};

/* ----- users & roles ----- */

VIEWS.users = async () => {
  const [u, r] = await Promise.all([api('/users'), api('/roles')]);
  const roles = r.roles || [];
  const permissions = r.permissions || [];

  const wrap = h('div', {}, h('div.head', {},
    h('div', {}, h('h1', { text: 'Pengguna & Peran' }),
      h('p', { text: 'Peran adalah kumpulan izin. Seorang pengguna mendapat gabungan izin dari semua perannya.' })),
    h('button.btn', { text: '+ Pengguna', onclick: () => userForm(null, roles) })));

  wrap.append(h('div.card', {}, h('table', {},
    h('thead', {}, h('tr', {},
      h('th', { text: 'Nama' }), h('th', { text: 'Email' }), h('th', { text: 'Peran' }),
      h('th', { text: 'Status' }), h('th', { text: 'Login terakhir' }), h('th', { text: '' }))),
    h('tbody', {}, (u.users || []).map(userRow)))));

  function statusPill(user) {
    if (user.locked_until) return h('span.pill.off', { text: 'terkunci' });
    return user.is_active ? h('span.pill', { text: 'aktif' }) : h('span.pill.off', { text: 'nonaktif' });
  }

  function userRow(user) {
    const del = h('button.btn.danger.sm', { text: 'Hapus', onclick: async () => {
      if (!confirm('Hapus akun ' + user.email + '?')) return;
      try { await api('/users/' + user.id, { method: 'DELETE' }); toast('Akun dihapus'); refresh(); }
      catch (e) { toast(e.message, 'err'); }
    } });
    const actions = h('div.row', {},
      h('button.btn.ghost.sm', { text: 'Ubah', onclick: () => userForm(user, roles) }),
      h('button.btn.ghost.sm', { text: 'Sandi', onclick: () => passwordForm(user) }),
      del);
    return h('tr', {},
      h('td', {}, h('b', { text: user.name })),
      h('td', { text: user.email }),
      h('td', {}, (user.roles || []).map(c => h('span.pill.grey', { style: 'margin-right:4px', text: c }))),
      h('td', {}, statusPill(user)),
      h('td.muted', { text: user.last_login_at ? new Date(user.last_login_at).toLocaleString('id-ID') : 'belum pernah' }),
      h('td', {}, actions));
  }

  wrap.append(h('div.card', {}, h('h2', { text: 'Peran' }),
    h('p.muted', { text: 'Peran Owner tidak bisa diubah — peran itu jaring pengaman sistem.' }),
    roles.map(role => {
      const held = new Set(role.permissions || []);
      const locked = role.code === 'owner';
      const boxes = permissions.map(p => h('label.chk', {},
        h('input', { type: 'checkbox', checked: held.has(p.code), disabled: locked,
          onchange: e => { e.target.checked ? held.add(p.code) : held.delete(p.code); } }),
        h('span', {}, p.label, ' ', h('span.muted', { text: p.code }))));
      const btn = h('button.btn.sm', { text: 'Simpan peran', disabled: locked, onclick: async () => {
        btn.disabled = true;
        try { await api('/roles/' + role.code + '/permissions', { method: 'PUT', body: { permissions: [...held] } });
          toast('Peran ' + role.name + ' tersimpan'); }
        catch (e) { toast(e.message, 'err'); }
        finally { btn.disabled = locked; }
      } });
      return h('div', { style: 'border-top:1px solid var(--line);padding-top:12px;margin-top:12px' },
        h('h3', {}, role.name, ' ', h('span.pill.grey', { text: role.code }), locked ? ' 🔒' : ''),
        h('p.muted', { text: role.description }),
        h('div.grid.two', {}, boxes),
        h('div', { style: 'margin-top:8px' }, btn));
    })));
  return wrap;

  function userForm(user, roles) {
    const isNew = !user;
    const model = { email: user ? user.email : '', name: user ? user.name : '',
      password: '', is_active: user ? user.is_active : true,
      roles: new Set(user ? user.roles : []) };
    const err = h('div');
    const body = h('div', {}, err,
      field('Nama', h('input', { type: 'text', value: model.name, oninput: e => { model.name = e.target.value; } })),
      field('Email', h('input', { type: 'email', value: model.email, disabled: !isNew,
        oninput: e => { model.email = e.target.value; } })),
      isNew ? field('Kata sandi (min. 12 karakter)',
        h('input', { type: 'password', autocomplete: 'new-password', oninput: e => { model.password = e.target.value; } })) : null,
      !isNew ? h('label.chk', {}, h('input', { type: 'checkbox', checked: model.is_active,
        onchange: e => { model.is_active = e.target.checked; } }), 'Akun aktif') : null,
      h('div', { style: 'margin-top:12px' }, h('label', { text: 'Peran' }),
        roles.map(r2 => h('label.chk', {}, h('input', { type: 'checkbox', checked: model.roles.has(r2.code),
          onchange: e => { e.target.checked ? model.roles.add(r2.code) : model.roles.delete(r2.code); } }),
          h('span', {}, r2.name, ' — ', h('span.muted', { text: r2.description }))))));

    const save = h('button.btn', { text: isNew ? 'Buat akun' : 'Simpan', onclick: async () => {
      err.replaceChildren(); save.disabled = true;
      try {
        if (isNew) {
          await api('/users', { method: 'POST', body: { email: model.email, name: model.name,
            password: model.password, roles: [...model.roles] } });
        } else {
          await api('/users/' + user.id, { method: 'PUT', body: { name: model.name, is_active: model.is_active } });
          await api('/users/' + user.id + '/roles', { method: 'PUT', body: { roles: [...model.roles] } });
        }
        toast('Tersimpan'); closeModal(); refresh();
      } catch (e) { err.replaceChildren(h('div.banner.err', { text: e.message })); save.disabled = false; }
    } });
    openModal(isNew ? 'Pengguna baru' : ('Ubah ' + user.email), body, [save]);
  }

  function passwordForm(user) {
    let pw = '';
    const err = h('div');
    const save = h('button.btn', { text: 'Ganti sandi', onclick: async () => {
      err.replaceChildren(); save.disabled = true;
      try { await api('/users/' + user.id + '/password', { method: 'PUT', body: { password: pw } });
        toast('Sandi diganti; semua sesi pengguna itu dicabut'); closeModal(); }
      catch (e) { err.replaceChildren(h('div.banner.err', { text: e.message })); save.disabled = false; }
    } });
    openModal('Ganti sandi ' + user.email,
      h('div', {}, err,
        h('p.muted', { text: 'Mengganti sandi otomatis mencabut semua sesi aktif akun ini.' }),
        field('Kata sandi baru (min. 12 karakter)',
          h('input', { type: 'password', autocomplete: 'new-password', oninput: e => { pw = e.target.value; } }))),
      [save]);
  }
};

/* ----- audit ----- */

VIEWS.audit = async () => {
  const r = await api('/audit?limit=200');
  return h('div', {},
    h('div.head', {}, h('div', {}, h('h1', { text: 'Log Aktivitas' }),
      h('p', { text: 'Siapa mengubah apa, dan kapan. Tindakan lewat service token tercatat sebagai "service-token" karena tidak ada orang yang bisa disebut.' }))),
    h('div.card', {}, h('table', {},
      h('thead', {}, h('tr', {}, h('th', { text: 'Waktu' }), h('th', { text: 'Aktor' }),
        h('th', { text: 'Tindakan' }), h('th', { text: 'Target' }), h('th', { text: 'Detail' }))),
      h('tbody', {}, (r.entries || []).length
        ? r.entries.map(e => h('tr', {},
            h('td.muted', { text: new Date(e.created_at).toLocaleString('id-ID') }),
            h('td', { text: e.actor }),
            h('td', {}, h('code', { text: e.action })),
            h('td.muted', { text: e.target || '—' }),
            h('td.muted', { text: e.detail && Object.keys(e.detail).length ? JSON.stringify(e.detail).slice(0, 120) : '' })))
        : h('tr', {}, h('td', { colspan: '5', text: 'Belum ada aktivitas.' }))))));
};

/* ---------- modal ---------- */

let modalEl = null;
function openModal(title, body, actions) {
  closeModal();
  const close = () => closeModal();
  const panel = h('div', {
    role: 'dialog', 'aria-modal': 'true', 'aria-label': title,
    style: 'background:var(--surface);border-radius:12px;max-width:760px;width:100%;max-height:88vh;overflow:auto;padding:20px;box-shadow:0 20px 60px rgba(0,0,0,.25)',
  },
    h('div.head', {}, h('h2', { text: title }), h('button.btn.ghost.sm', { text: 'Tutup', onclick: close })),
    body,
    h('div.row', { style: 'margin-top:18px;justify-content:flex-end' },
      h('button.btn.ghost', { text: 'Batal', onclick: close }), ...actions));

  modalEl = h('div', {
    style: 'position:fixed;inset:0;background:rgba(43,38,32,.5);display:flex;align-items:center;justify-content:center;padding:20px;z-index:40',
    onclick: e => { if (e.target === modalEl) close(); },
  }, panel);
  document.body.append(modalEl);
  document.addEventListener('keydown', onEsc);
  // Focus the first control so the dialog is usable from the keyboard, which
  // the captured site's own modal never was (A11Y-10).
  const first = panel.querySelector('input,select,textarea,button');
  if (first) first.focus();
}
function onEsc(e) { if (e.key === 'Escape') closeModal(); }
function closeModal() {
  if (modalEl) { modalEl.remove(); modalEl = null; }
  document.removeEventListener('keydown', onEsc);
}

/* ---------- date helpers ---------- */

function pad(n) { return String(n).padStart(2, '0'); }
function iso(d) { return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()); }
function addDays(isoStr, n) {
  const p = isoStr.split('-');
  const d = new Date(+p[0], +p[1] - 1, +p[2]);
  d.setDate(d.getDate() + n);
  return iso(d);
}
function nextMonday() {
  const d = new Date();
  // Sunday is 0; the coming Monday is 1 day away from Sunday and 7 from Monday.
  const delta = (8 - (d.getDay() || 7)) % 7 || 7;
  d.setDate(d.getDate() + delta);
  return iso(d);
}
function isoWeekOf(date) {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const dayNum = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
  return Math.ceil(((d - yearStart) / 86400000 + 1) / 7);
}

boot();
