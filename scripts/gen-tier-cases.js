#!/usr/bin/env node
// Generates the differential-test fixture for the Go pricing engine.
//
// It runs the REAL analyze() — extracted verbatim from site/index.html by
// tests/extract-analyze.js — over a wide spread of date shapes, and records what
// it decided. The Go port is then asserted against that record.
//
// The point is that neither side is trusted: the fixture is produced by the code
// the customer's browser actually runs, and regenerating it after a re-capture
// immediately shows whether upstream changed the pricing rules.
//
//   node scripts/gen-tier-cases.js > server/internal/domain/pricing/testdata/tier_cases.json
'use strict';
const { buildAnalyze, readRates } = require('../tests/extract-analyze.js');

const analyze = buildAnalyze();
const RATES = readRates();

// A tiny deterministic PRNG (mulberry32). Math.random() would make the fixture
// different on every run, so a regenerated file would show spurious diffs and
// nobody could tell a real upstream change from noise.
function rng(seed) {
  return function () {
    seed |= 0; seed = (seed + 0x6D2B79F5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function key(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}
function addDays(iso, n) {
  const d = new Date(iso + 'T00:00:00');
  d.setDate(d.getDate() + n);
  return key(d);
}

const cases = [];
function push(name, dates, planName) {
  const plan = planName || 'Healthy Meal';
  const r = analyze(dates, RATES[plan]);
  cases.push({
    name,
    plan,
    dates,
    want: { n: r.n, consecutive: r.consecutive, span: r.span, tier_key: r.tier.key, rate: r.tier.rate },
  });
}

// 1. Consecutive runs of every length 1..32, started on each day of the week.
//    This walks straight through every boundary in the ladder: 5, 14, 15, 19, 20.
const anchors = [
  ['Mon', '2026-08-17'], ['Tue', '2026-08-18'], ['Wed', '2026-08-19'],
  ['Thu', '2026-08-20'], ['Fri', '2026-08-21'], ['Sat', '2026-08-22'], ['Sun', '2026-08-23'],
];
for (const [dayName, start] of anchors) {
  for (let n = 1; n <= 32; n++) {
    const dates = [];
    for (let i = 0; i < n; i++) dates.push(addDays(start, i));
    push(`consecutive ${n}d from ${dayName}`, dates);
  }
}

// 2. Mon–Fri routines: skip weekends, 1..26 working days, from each of 6 start
//    Mondays. Exercises BR-3.11's clean-gap and span rules.
for (let wk = 0; wk < 6; wk++) {
  const monday = addDays('2026-08-17', wk * 7);
  for (let n = 1; n <= 26; n++) {
    const dates = [];
    let cursor = 0;
    while (dates.length < n) {
      const d = new Date(addDays(monday, cursor) + 'T00:00:00');
      if (d.getDay() >= 1 && d.getDay() <= 5) dates.push(key(d));
      cursor++;
    }
    push(`mon-fri ${n}d week+${wk}`, dates);
  }
}

// 3. Mon–Sat routines: skip Sundays only.
for (let wk = 0; wk < 4; wk++) {
  const monday = addDays('2026-08-17', wk * 7);
  for (let n = 1; n <= 26; n++) {
    const dates = [];
    let cursor = 0;
    while (dates.length < n) {
      const d = new Date(addDays(monday, cursor) + 'T00:00:00');
      if (d.getDay() !== 0) dates.push(key(d));
      cursor++;
    }
    push(`mon-sat ${n}d week+${wk}`, dates);
  }
}

// 4. The two worked examples from the source's own comment about BR-3.12.
push('BR-3.12 no week reaches 5 -> stays Flexi',
  ['2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21', '2026-08-24']);
push('BR-3.12 second week has 5 -> whole order Mingguan',
  ['2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21',
   '2026-08-24', '2026-08-26', '2026-08-27', '2026-08-28', '2026-08-29']);

// 5. Scattered sets: random picks from a widening window, which is where the
//    span-based Flexi branches live.
const rand = rng(20260827);
for (let i = 0; i < 260; i++) {
  const window = 7 + Math.floor(rand() * 55);
  const count = 1 + Math.floor(rand() * Math.min(window, 28));
  const offsets = new Set();
  while (offsets.size < count) offsets.add(Math.floor(rand() * window));
  const dates = [...offsets].sort((a, b) => a - b).map((o) => addDays('2026-08-17', o));
  push(`scattered ${count}d in ${window}d #${i}`, dates);
}

// 6. Every plan, so the rate carried on the tier is checked too, not just the
//    classification.
for (const plan of Object.keys(RATES)) {
  push(`${plan} mon-fri week`, ['2026-08-17', '2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21'], plan);
  push(`${plan} 20 consecutive`, Array.from({ length: 20 }, (_, i) => addDays('2026-08-17', i)), plan);
  push(`${plan} 3 scattered`, ['2026-08-17', '2026-08-20', '2026-08-25'], plan);
}

process.stdout.write(JSON.stringify({
  generated_from: 'site/index.html',
  note: 'Produced by scripts/gen-tier-cases.js from the analyze() in the captured mirror. Regenerate after any re-capture.',
  rates: RATES,
  cases,
}, null, 1) + '\n');
