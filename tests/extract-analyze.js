// Pulls analyze() — the Daily Order pricing engine — VERBATIM out of the
// mirrored page and makes it callable from Node.
//
// The point is that the tests exercise the REAL code. Nothing is copy-pasted or
// reimplemented here: if site/index.html changes, these tests test the new
// behaviour automatically. site/index.html is never written to.
'use strict';
const fs = require('fs');
const path = require('path');

const MIRROR = path.join(__dirname, '..', 'site', 'index.html');

function extractBalanced(src, startIdx) {
  const open = src.indexOf('{', startIdx);
  let depth = 0;
  for (let i = open; i < src.length; i++) {
    const c = src[i];
    if (c === '{') depth++;
    else if (c === '}') {
      depth--;
      if (depth === 0) return src.slice(startIdx, i + 1);
    }
  }
  throw new Error('unbalanced braces while extracting');
}

function buildAnalyze() {
  const html = fs.readFileSync(MIRROR, 'utf8');

  // The page ships more than one inline <script> (the site router/nav first,
  // then the Order Online app). analyze() lives in the Order app, so pick the
  // block that actually defines it rather than assuming it is the first one.
  const script = [...html.matchAll(/<script[^>]*>([\s\S]*?)<\/script>/g)]
    .map(m => m[1])
    .find(src => src.includes('function analyze()'));
  if (script === undefined) throw new Error('no <script> in the mirror defines analyze()');

  const idx = script.indexOf('function analyze()');
  const analyzeSrc = extractBalanced(script, idx);

  // The helpers analyze() closes over, also taken verbatim.
  const toDateObjSrc = extractBalanced(script, script.indexOf('function toDateObj'));
  const toKeySrc = extractBalanced(script, script.indexOf('function toKey'));

  const factory = new Function(
    'datesArg', 'rates',
    `
    const DAY_MS = 24*60*60*1000;
    ${toDateObjSrc}
    ${toKeySrc}
    const dates = datesArg;
    ${analyzeSrc}
    return analyze();
    `
  );

  return (dates, rates) => factory(dates, rates);
}

// The four Daily Order rate tables, read out of the mirror's data-rates
// attributes rather than hard-coded.
function readRates() {
  const html = fs.readFileSync(MIRROR, 'utf8');
  const out = {};
  // Both attributes sit on the same .order-card tag, so match them adjacently:
  // a looser window would let a card without rates (Nasi Bento, Paket Acara)
  // borrow the next card's data-rates.
  const re = /data-sub="([^"]+)"\s+data-rates='([^']+)'/g;
  let m;
  while ((m = re.exec(html)) !== null) {
    out[m[1]] = JSON.parse(m[2].replace(/&quot;/g, '"'));
  }
  return out;
}

module.exports = { buildAnalyze, readRates, MIRROR };
