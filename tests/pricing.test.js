// Boundary tests for the Daily Order pricing engine, BR-3.1 – BR-3.12.
// The code under test is extracted verbatim from site/index.html at run time.
'use strict';
const { test, describe } = require('node:test');
const assert = require('node:assert');
const { buildAnalyze, readRates } = require('./extract-analyze.js');

const analyze = buildAnalyze();

// Healthy Meal, straight out of the mirror.
const RATES = readRates()['Healthy Meal'];

// 2026-08-17 is a Monday (the page labels it "Senin 17 Agu").
function run(dates) { return analyze(dates, RATES); }

// Format a Date as YYYY-MM-DD in LOCAL time. Never use toISOString() here: this
// machine is Asia/Jakarta (UTC+7), so local midnight is 17:00 UTC the previous
// day and toISOString() would silently shift every date back by one, turning a
// Mon-Fri run into Sun-Thu. The page's own toKey() formats locally for exactly
// this reason.
function key(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function seq(startISO, n) {
  const out = [];
  const d = new Date(startISO + 'T00:00:00');
  for (let i = 0; i < n; i++) {
    out.push(key(d));
    d.setDate(d.getDate() + 1);
  }
  return out;
}

// Guard the helper itself, so a timezone change can never silently invalidate
// every date-shape assertion below.
test('helper sanity: 2026-08-17 is a Monday and seq() does not drift', () => {
  assert.strictEqual(new Date('2026-08-17T00:00:00').getDay(), 1);
  assert.deepStrictEqual(seq('2026-08-17', 3), ['2026-08-17', '2026-08-18', '2026-08-19']);
});

describe('rate table integrity', () => {
  test('Healthy Meal rates match the documented catalogue', () => {
    assert.deepStrictEqual(RATES, {
      daily: 50000, weeklyPerDay: 38000, monthlyPerDay: 35000,
      flexiWeeklyPerDay: 40000, flexiMonthlyPerDay: 38000,
    });
  });

  test('all four Daily Order cards expose the same five keys', () => {
    const all = readRates();
    assert.deepStrictEqual(Object.keys(all).sort(),
      ['Bulking Extra', 'Healthy Meal', 'Kids Meal', 'Regular Catering']);
    for (const [name, r] of Object.entries(all)) {
      assert.deepStrictEqual(Object.keys(r).sort(),
        ['daily', 'flexiMonthlyPerDay', 'flexiWeeklyPerDay', 'monthlyPerDay', 'weeklyPerDay'],
        `${name} rate keys`);
      assert.ok(r.monthlyPerDay <= r.weeklyPerDay, `${name}: monthly must not exceed weekly`);
      assert.ok(r.weeklyPerDay <= r.daily, `${name}: weekly must not exceed daily`);
    }
  });
});

describe('BR-3.5 — fewer than 5 consecutive days is Harian', () => {
  for (const n of [1, 2, 3, 4]) {
    test(`${n} consecutive day(s) -> daily rate`, () => {
      const r = run(seq('2026-08-17', n));
      assert.strictEqual(r.tier.key, 'daily');
      assert.strictEqual(r.tier.rate, RATES.daily);
      assert.strictEqual(r.consecutive, true);
    });
  }
});

describe('BR-3.4 — 5+ consecutive inside one calendar week is Mingguan', () => {
  test('Mon-Fri (17-21 Aug) -> weekly', () => {
    const r = run(seq('2026-08-17', 5));
    assert.strictEqual(r.tier.key, 'weekly');
    assert.strictEqual(r.tier.rate, RATES.weeklyPerDay);
  });

  test('Mon-Sat (17-22 Aug) -> weekly, still one calendar week', () => {
    const r = run(seq('2026-08-17', 6));
    assert.strictEqual(r.tier.key, 'weekly');
  });
});

describe('BR-3.2 — a consecutive run straddling two calendar weeks is Flexi Mingguan', () => {
  test('Thu-Mon (20-24 Aug) -> flexi-weekly, not weekly', () => {
    const r = run(seq('2026-08-20', 5));
    assert.strictEqual(r.consecutive, true);
    assert.strictEqual(r.tier.key, 'flexi-weekly');
    assert.strictEqual(r.tier.rate, RATES.flexiWeeklyPerDay);
  });

  test('the customer pays MORE than an aligned 5-day week', () => {
    const aligned = run(seq('2026-08-17', 5));
    const straddling = run(seq('2026-08-20', 5));
    assert.ok(straddling.tier.rate > aligned.tier.rate,
      'straddling a week boundary costs more per day');
  });
});

describe('BR-3.3 — the 15-19 consecutive-day dead zone pays FULL price', () => {
  for (const n of [15, 16, 17, 18, 19]) {
    test(`${n} consecutive days -> daily rate (more per day than 5 days)`, () => {
      const r = run(seq('2026-08-17', n));
      assert.strictEqual(r.tier.key, 'flexi');
      assert.strictEqual(r.tier.rate, RATES.daily);
    });
  }

  test('19 consecutive days costs more PER DAY than 14', () => {
    assert.ok(run(seq('2026-08-17', 19)).tier.rate > run(seq('2026-08-17', 14)).tier.rate);
  });

  test('the cliff edges: 14 discounted, 15 not, 20 cheapest', () => {
    assert.strictEqual(run(seq('2026-08-17', 14)).tier.rate, RATES.flexiWeeklyPerDay);
    assert.strictEqual(run(seq('2026-08-17', 15)).tier.rate, RATES.daily);
    assert.strictEqual(run(seq('2026-08-17', 20)).tier.rate, RATES.monthlyPerDay);
  });
});

describe('BR-3.1 — 20+ consecutive days is Bulanan', () => {
  for (const n of [20, 25, 30]) {
    test(`${n} consecutive days -> monthly`, () => {
      const r = run(seq('2026-08-17', n));
      assert.strictEqual(r.tier.key, 'monthly');
      assert.strictEqual(r.tier.rate, RATES.monthlyPerDay);
    });
  }
});

describe('BR-3.12 — weekly routine needs one calendar week holding 5+ dates', () => {
  // Both examples are taken from the comments in the source itself.
  test('18,19,20,21,24 Aug -> NOT weekly (no single week has 5)', () => {
    const r = run(['2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21', '2026-08-24']);
    assert.strictEqual(r.consecutive, false);
    assert.strictEqual(r.tier.key, 'flexi-weekly');
  });

  test('18,19,20,21,24,26,27,28,29 Aug -> weekly (second week holds 5)', () => {
    const r = run(['2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21',
                   '2026-08-24', '2026-08-26', '2026-08-27', '2026-08-28', '2026-08-29']);
    assert.strictEqual(r.consecutive, false);
    assert.strictEqual(r.tier.key, 'weekly');
    assert.strictEqual(r.tier.rate, RATES.weeklyPerDay);
  });

  test('adding dates can promote the WHOLE order to a cheaper rate', () => {
    const before = run(['2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21', '2026-08-24']);
    const after = run(['2026-08-18', '2026-08-19', '2026-08-20', '2026-08-21',
                       '2026-08-24', '2026-08-26', '2026-08-27', '2026-08-28', '2026-08-29']);
    assert.ok(after.tier.rate < before.tier.rate);
  });
});

describe('BR-3.11 — a 20+ day Mon-Fri routine is Bulanan even when not consecutive', () => {
  test('20 weekdays across 4 clean Mon-Fri weeks -> monthly', () => {
    const dates = [];
    const d = new Date('2026-08-17T00:00:00');
    while (dates.length < 20) {
      const dow = d.getDay();
      if (dow >= 1 && dow <= 5) dates.push(key(d));
      d.setDate(d.getDate() + 1);
    }
    const r = run(dates);
    assert.strictEqual(r.consecutive, false);
    assert.strictEqual(r.tier.key, 'monthly');
    assert.strictEqual(r.tier.rate, RATES.monthlyPerDay);
  });
});

describe('BR-3.8 — 20+ scattered dates inside 45 days is Flexi Bulanan', () => {
  test('20 dates spread over ~40 days with dirty gaps -> flexi-monthly', () => {
    const dates = [];
    const d = new Date('2026-08-17T00:00:00');
    while (dates.length < 20) {
      dates.push(key(d));
      d.setDate(d.getDate() + 2); // every other day: never a clean weekday routine
    }
    const r = run(dates);
    assert.strictEqual(r.tier.key, 'flexi-monthly');
    assert.strictEqual(r.tier.rate, RATES.flexiMonthlyPerDay);
    assert.ok(r.span <= 45);
  });
});

describe('BR-3.10 — fewer than 5 scattered dates is full price', () => {
  test('3 random dates -> daily rate', () => {
    const r = run(['2026-08-18', '2026-08-25', '2026-09-03']);
    assert.strictEqual(r.consecutive, false);
    assert.strictEqual(r.tier.key, 'flexi');
    assert.strictEqual(r.tier.rate, RATES.daily);
  });
});

describe('shape of the returned analysis', () => {
  test('empty selection returns null', () => {
    assert.strictEqual(run([]), null);
  });

  test('n, consecutive and span are reported correctly', () => {
    const r = run(['2026-08-17', '2026-08-19', '2026-08-21']);
    assert.strictEqual(r.n, 3);
    assert.strictEqual(r.consecutive, false);
    assert.strictEqual(r.span, 5); // 17..21 inclusive
  });
});

describe('BR-2.1 — the item total is unitPrice x qty x dates', () => {
  test('3 pax, Mon-Fri Healthy Meal = 570000', () => {
    const r = run(seq('2026-08-17', 5));
    assert.strictEqual(r.tier.rate * 3 * r.n, 570000);
  });

  test('4 scattered dates cost more than 5 aligned ones', () => {
    const aligned = run(seq('2026-08-17', 5));
    const scattered = run(['2026-08-18', '2026-08-20', '2026-08-25', '2026-08-27']);
    assert.ok(scattered.tier.rate * 3 * scattered.n > aligned.tier.rate * 3 * aligned.n);
  });
});
