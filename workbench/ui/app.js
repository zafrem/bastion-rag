'use strict';
// Every piece of dynamic text is placed with textContent (via h()), never innerHTML,
// so model output, prompts and service messages can never inject markup.

const $ = (id) => document.getElementById(id);
function h(tag, props, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(props || {})) {
    if (v == null || v === false) continue;
    if (k === 'class') el.className = v;
    else if (k === 'text') el.textContent = v;
    else if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v === true ? '' : v);
  }
  for (const kid of kids.flat()) if (kid != null && kid !== false) el.append(kid.nodeType ? kid : document.createTextNode(String(kid)));
  return el;
}
function clear(el) { while (el.firstChild) el.removeChild(el.firstChild); return el; }

async function api(path, body) {
  const opt = body === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) };
  const r = await fetch(path, opt);
  const data = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(data.error || ('HTTP ' + r.status));
  return data;
}

const ICON = { passed: '✓', reused: '↺', modified: '✎', flagged: '⚑', would_block: '⚠', blocked: '✕', error: '!', skipped: '–' };
const LABEL = { passed: 'Passed', reused: 'Shared answer', modified: 'Changed text', flagged: 'Flagged', would_block: 'Would block', blocked: 'Blocked', error: 'Error', skipped: 'Skipped' };
const STAGES = ['sentinel-in', 'context', 'llm', 'sentinel-out', 'anchor'];

let profiles = [], cases = [];

// ── status pills ──────────────────────────────────────────────────────────────
async function refreshStatus() {
  try {
    const list = await api('/api/status');
    const box = clear($('status'));
    for (const s of list) {
      box.append(h('span', { class: 'pill ' + (s.Up || s.up ? 'up' : 'down'), title: s.detail }, h('i'), s.name, ' ', s.detail && !s.up ? '(' + s.detail.slice(0, 40) + ')' : ''));
    }
  } catch (e) { clear($('status')).append(h('span', { class: 'pill down' }, h('i'), 'workbench unreachable')); }
}

// ── tabs ──────────────────────────────────────────────────────────────────────
$('tabs').addEventListener('click', (e) => {
  const b = e.target.closest('button[data-tab]'); if (!b) return;
  for (const t of document.querySelectorAll('#tabs button')) t.setAttribute('aria-selected', String(t === b));
  for (const s of document.querySelectorAll('.tab')) s.hidden = s.id !== 'tab-' + b.dataset.tab;
});

// ── shared pieces ─────────────────────────────────────────────────────────────
function splitContext(text) { return text.split(/\n\s*\n/).map((s) => s.trim()).filter(Boolean); }

function fillExamples(box, onPick) {
  clear(box);
  for (const c of cases) box.append(h('button', { class: 'chip', type: 'button', title: c.why, onclick: () => onPick(c) }, c.title));
}

function busy(btn, on, label) { btn.disabled = on; btn.textContent = on ? 'Working…' : label; }

function stageCard(s, opts = {}) {
  const card = h('button', { class: 'stage st-' + s.status + (opts.diverge ? ' diverge' : ''), type: 'button', onclick: () => openStage(s) },
    s.duration_ms ? h('span', { class: 'ms', text: s.duration_ms.toFixed(1) + ' ms' }) : null,
    h('div', { class: 'mod', text: s.module }),
    h('div', { class: 'name', text: s.name }),
    h('span', { class: 'tag' }, ICON[s.status] || '?', ' ', LABEL[s.status] || s.status),
    h('div', { class: 'sum', text: s.summary || '' }));
  if (opts.diverge) card.title = 'The profiles first behave differently here';
  return card;
}

// Word-level diff. left = the model's text with removed words marked; right = what the user gets with inserted words marked.
function diffPair(a, b) {
  const x = a.split(/(\s+)/).filter((t) => t !== ''), y = b.split(/(\s+)/).filter((t) => t !== '');
  if (x.length * y.length > 160000) return { left: [document.createTextNode(a)], right: [document.createTextNode(b)] };
  const L = Array.from({ length: x.length + 1 }, () => new Uint16Array(y.length + 1));
  for (let i = x.length - 1; i >= 0; i--) for (let j = y.length - 1; j >= 0; j--)
    L[i][j] = x[i] === y[j] ? L[i + 1][j + 1] + 1 : Math.max(L[i + 1][j], L[i][j + 1]);
  const left = [], right = [];
  const add = (arr, tag, t) => { const last = arr[arr.length - 1]; if (last && last.tag === tag) last.t += t; else arr.push({ tag, t }); };
  let i = 0, j = 0;
  while (i < x.length && j < y.length) {
    if (x[i] === y[j]) { add(left, '=', x[i]); add(right, '=', y[j]); i++; j++; }
    else if (L[i + 1][j] >= L[i][j + 1]) { add(left, 'del', x[i]); i++; }
    else { add(right, 'ins', y[j]); j++; }
  }
  while (i < x.length) add(left, 'del', x[i++]);
  while (j < y.length) add(right, 'ins', y[j++]);
  const nodes = (arr) => arr.map((o) => o.tag === '=' ? document.createTextNode(o.t) : h(o.tag, { text: o.t }));
  return { left: nodes(left), right: nodes(right) };
}

function answers(res) {
  const raw = res.raw_answer, fin = res.final_answer;
  const d = res.altered ? diffPair(raw, fin) : null;
  const left = h('div', { class: 'answer' }, h('h3', { text: 'What the model wrote' }),
    raw ? h('p', {}, ...(d ? d.left : [raw])) : h('p', { class: 'none', text: 'The model was not called.' }));
  const right = h('div', { class: 'answer' }, h('h3', { text: 'What the user receives' }),
    fin ? h('p', {}, ...(d ? d.right : [fin])) : h('p', { class: 'none', text: res.outcome === 'delivered' ? 'Nothing to show.' : 'No answer delivered.' }));
  return h('div', { class: 'answers' }, left, right);
}

function banner(res) {
  let cls = res.outcome, text;
  if (res.outcome === 'delivered') {
    cls = res.altered ? 'altered' : 'delivered';
    text = res.altered ? 'Delivered, but the text was changed' : 'Delivered unchanged';
  } else if (res.outcome === 'blocked') text = 'Blocked at ' + stageName(res, res.blocked_at);
  else text = 'Stopped: ' + stageName(res, res.blocked_at) + ' could not run (fail closed)';
  const extra = res.would_block_at ? 'Observe mode: would have been blocked at ' + stageName(res, res.would_block_at) : '';
  return h('div', { class: 'banner ' + cls }, text, extra ? h('small', { text: extra }) : null, h('small', { text: res.total_ms.toFixed(0) + ' ms total' }));
}
function stageName(res, id) { const s = res.stages.find((x) => x.id === id); return s ? s.name : id || 'a stage'; }

function openStage(s) {
  const c = clear($('drawer-content'));
  c.append(h('h2', { text: s.name }), h('p', { class: 'hint', text: s.module + ' · ' + (LABEL[s.status] || s.status) }), h('p', { text: s.summary || '' }));
  if (s.notice) c.append(h('h3', { text: 'What the user is told' }), h('p', { text: s.notice.message || '' }),
    kv({ code: s.notice.code, action: s.notice.action, count: s.notice.count }));
  if (s.detail) { c.append(h('h3', { text: 'Detail' }), kv(s.detail)); }
  $('drawer').hidden = false; $('drawer-close').focus();
}
function kv(obj) {
  const dl = h('dl', { class: 'kv' });
  for (const [k, v] of Object.entries(obj)) {
    if (v == null || (Array.isArray(v) && !v.length) || v === '') continue;
    dl.append(h('dt', { text: k.replace(/_/g, ' ') }), h('dd', { text: typeof v === 'object' ? JSON.stringify(v) : String(v) }));
  }
  return dl;
}
$('drawer-close').addEventListener('click', () => { $('drawer').hidden = true; });
$('drawer').addEventListener('click', (e) => { if (e.target.id === 'drawer') $('drawer').hidden = true; });
document.addEventListener('keydown', (e) => { if (e.key === 'Escape') $('drawer').hidden = true; });

function errBox(e) { return h('div', { class: 'banner error', role: 'alert' }, 'Failed: ' + e.message); }

// ── Run ───────────────────────────────────────────────────────────────────────
function pickCase(c, promptEl, ctxEl, ansEl) {
  promptEl.value = c.prompt; ctxEl.value = (c.context || []).join('\n\n'); if (ansEl) ansEl.value = '';
}
$('run-go').addEventListener('click', async () => {
  const btn = $('run-go'), out = clear($('run-out'));
  if (!$('run-prompt').value.trim()) { out.append(errBox(new Error('Enter a prompt first'))); return; }
  busy(btn, true); out.append(h('p', {}, h('span', { class: 'spin' }), 'Running…'));
  try {
    const res = await api('/api/run', { prompt: $('run-prompt').value, profile: $('run-profile').value,
      context: splitContext($('run-context').value), observe: $('run-observe').checked, answer: $('run-answer').value.trim() });
    clear(out).append(banner(res), h('div', { class: 'strip' }, res.stages.map((s) => stageCard(s))), answers(res));
  } catch (e) { clear(out).append(errBox(e)); } finally { busy(btn, false, 'Run through pipeline'); }
});

// ── Compare ───────────────────────────────────────────────────────────────────
function checkedProfiles(boxId) { return [...document.querySelectorAll('#' + boxId + ' input:checked')].map((i) => i.value); }
$('cmp-go').addEventListener('click', async () => {
  const btn = $('cmp-go'), out = clear($('cmp-out')), picked = checkedProfiles('cmp-profiles');
  if (!$('cmp-prompt').value.trim()) { out.append(errBox(new Error('Enter a prompt first'))); return; }
  if (picked.length < 2 || picked.length > 4) { out.append(errBox(new Error('Choose 2 to 4 profiles'))); return; }
  busy(btn, true); out.append(h('p', {}, h('span', { class: 'spin' }), 'Comparing…'));
  try {
    const r = await api('/api/compare', { prompt: $('cmp-prompt').value, profiles: picked, context: splitContext($('cmp-context').value),
      observe: $('cmp-observe').checked, answer: $('cmp-answer').value.trim() });
    clear(out);
    out.append(h('div', { class: 'answer shared' }, h('h3', { text: 'The one model answer every profile received' }), h('p', { text: r.shared_answer || '(none)' })));
    if (r.first_divergence) out.append(h('p', { class: 'hint', text: 'Dashed outline: the first stage where the profiles behave differently.' }));
    else out.append(h('p', { class: 'hint', text: 'The profiles behaved identically at every stage.' }));
    out.append(h('div', { class: 'cols' }, r.results.map((res) => {
      const p = profiles.find((x) => x.name === res.profile) || {};
      return h('div', { class: 'col' }, h('h3', { text: res.profile }), h('div', { class: 'desc', text: p.description || '' }), banner(res),
        h('div', { class: 'strip vertical' }, res.stages.map((s) => stageCard(s, { diverge: s.id === r.first_divergence }))),
        h('div', { class: 'answer' }, h('h3', { text: 'User receives' }), res.final_answer ? h('p', {}, ...(res.altered ? diffPair(res.raw_answer, res.final_answer).right : [res.final_answer])) : h('p', { class: 'none', text: 'No answer delivered.' })));
    })));
  } catch (e) { clear(out).append(errBox(e)); } finally { busy(btn, false, 'Compare'); }
});

// ── Suite ─────────────────────────────────────────────────────────────────────
function cellView(c, row) {
  const kind = c.outcome === 'delivered' ? (c.altered ? 'altered' : 'delivered') : c.outcome;
  const label = c.outcome === 'delivered' ? (c.altered ? 'Delivered, altered' : 'Delivered') : c.outcome === 'blocked' ? 'Blocked at ' + ({ 'sentinel-in': 'input check', 'sentinel-out': 'output check' }[c.at] || c.at) : 'Error';
  return h('button', { class: 'cell ' + kind, type: 'button', onclick: () => openCell(c, row) },
    h('span', { class: 'res', text: label }), h('span', { class: 'why', text: c.summary || (c.would_block_at ? 'would block at ' + c.would_block_at : '') }),
    h('div', { class: 'dots', 'aria-hidden': 'true' }, STAGES.map((id) => h('i', { class: 'd-' + (c.stages[id] || 'skipped') }))));
}
function openCell(c, row) {
  const d = clear($('drawer-content'));
  d.append(h('h2', { text: row.case.title }), h('p', { class: 'hint', text: c.profile + ' · ' + row.case.why }),
    h('h3', { text: 'Prompt' }), h('p', { text: row.case.prompt }),
    h('h3', { text: 'Model answer' }), h('p', { text: row.raw_answer || '' }),
    h('h3', { text: 'User receives' }), h('p', { text: c.final || '(nothing delivered)' }),
    h('h3', { text: 'Stages' }), kv(c.stages));
  $('drawer').hidden = false; $('drawer-close').focus();
}
$('suite-go').addEventListener('click', async () => {
  const btn = $('suite-go'), out = clear($('suite-out')), picked = checkedProfiles('suite-profiles');
  if (!picked.length) { out.append(errBox(new Error('Choose at least one profile'))); return; }
  busy(btn, true); out.append(h('p', {}, h('span', { class: 'spin' }), $('suite-llm').checked ? 'Running with the live model, this takes a while…' : 'Running…'));
  try {
    const rows = await api('/api/suite', { profiles: picked, observe: $('suite-observe').checked, use_llm: $('suite-llm').checked });
    clear(out);
    const harmless = rows.filter((r) => r.case.id.startsWith('benign') && r.cells.some((c) => c.altered));
    if (harmless.length) out.append(h('div', { class: 'callout', role: 'note' }, h('b', { text: 'Harmless answers were changed' }),
      harmless.map((r) => r.case.title).join('; ') + '. Open a cell to see which rule altered the text; a normal answer should arrive as the model wrote it.'));
    out.append(h('p', { class: 'legend' }, h('span', { text: 'Dots = Input · Context · Model · Output · Bias' }), h('span', { text: 'Click a cell for detail' })));
    const head = h('tr', {}, h('th', { text: 'Scenario' }), picked.map((p) => h('th', { text: p })));
    out.append(h('table', { class: 'matrix' }, h('thead', {}, head),
      h('tbody', {}, rows.map((r) => h('tr', {}, h('td', { class: 'case' }, h('b', { text: r.case.title }), h('span', { text: r.case.why })), r.cells.map((c) => h('td', {}, cellView(c, r))))))));
  } catch (e) { clear(out).append(errBox(e)); } finally { busy(btn, false, 'Run suite'); }
});

// ── Profiles ──────────────────────────────────────────────────────────────────
$('prof-save').addEventListener('click', async () => {
  const msg = $('prof-msg');
  try {
    const list = JSON.parse($('prof-json').value);
    profiles = await api('/api/profiles', list); renderProfiles(); msg.className = ''; msg.textContent = 'Saved.';
  } catch (e) { msg.className = 'err'; msg.textContent = e.message; }
});
function renderProfiles() {
  $('prof-json').value = JSON.stringify(profiles, null, 2);
  const sel = clear($('run-profile')); for (const p of profiles) sel.append(h('option', { value: p.name, text: p.name + ' — ' + p.description }));
  for (const id of ['cmp-profiles', 'suite-profiles']) {
    const box = clear($(id));
    profiles.forEach((p, i) => box.append(h('label', { class: 'check' }, h('input', { type: 'checkbox', value: p.name, checked: i < (id === 'cmp-profiles' ? 2 : 3) }), p.name)));
  }
}

// ── boot ──────────────────────────────────────────────────────────────────────
(async function () {
  refreshStatus(); setInterval(refreshStatus, 10000);
  try {
    [profiles, cases] = await Promise.all([api('/api/profiles'), api('/api/cases')]);
    renderProfiles();
    fillExamples($('run-examples'), (c) => pickCase(c, $('run-prompt'), $('run-context'), $('run-answer')));
    fillExamples($('cmp-examples'), (c) => pickCase(c, $('cmp-prompt'), $('cmp-context'), $('cmp-answer')));
  } catch (e) { $('run-out').append(errBox(e)); }
})();
