package dashboard

// DashboardTemplate is the full HTML dashboard with Go text/template placeholders.
// The JS constants (ASSIGNEE_DATA, PROJECT_DATA, STATUS_DATA, PRIORITY_DATA,
// OLDEST_DATA, CALLOUTS_DATA, ISSUES) are injected as raw JSON by the Go
// program — no JS changes needed to track new team members or projects.
const DashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Team Jira Dashboard</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
  <style>
    @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap');
    body { font-family: 'Inter', system-ui, sans-serif; background: #060910; color: #e2e8f0; }
    .gradient-text { background: linear-gradient(135deg, #76b900 0%, #00c4ff 60%, #a78bfa 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
    .card { background: rgba(255,255,255,0.035); border: 1px solid rgba(255,255,255,0.07); border-radius: 12px; }
    .kpi-card { background: linear-gradient(135deg, rgba(255,255,255,0.04) 0%, rgba(255,255,255,0.02) 100%); border: 1px solid rgba(255,255,255,0.08); border-radius: 14px; transition: border-color 0.2s; }
    .kpi-card:hover { border-color: rgba(118,185,0,0.3); }
    a.kpi-card { display: block; text-decoration: none; color: inherit; cursor: pointer; }
    .scope-pill { display: inline-flex; align-items: center; font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 99px; font-family: 'JetBrains Mono', monospace; }
    .badge { font-size: 10px; font-weight: 700; padding: 2px 7px; border-radius: 4px; text-transform: uppercase; letter-spacing: 0.05em; }
    .priority-highest, .priority-high { background: rgba(239,68,68,0.15); color: #fca5a5; border: 1px solid rgba(239,68,68,0.25); }
    .priority-medium { background: rgba(251,191,36,0.12); color: #fde68a; border: 1px solid rgba(251,191,36,0.2); }
    .priority-low, .priority-lowest { background: rgba(56,189,248,0.12); color: #7dd3fc; border: 1px solid rgba(56,189,248,0.2); }
    .status-open  { background: rgba(239,68,68,0.12); color: #f87171; border: 1px solid rgba(239,68,68,0.2); }
    .status-progress { background: rgba(251,191,36,0.12); color: #fbbf24; border: 1px solid rgba(251,191,36,0.2); }
    .status-review { background: rgba(56,189,248,0.12); color: #38bdf8; border: 1px solid rgba(56,189,248,0.2); }
    .status-done  { background: rgba(118,185,0,0.12); color: #84cc16; border: 1px solid rgba(118,185,0,0.2); }
    table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
    thead th { background: rgba(255,255,255,0.04); padding: 10px 14px; text-align: left; font-weight: 600; font-size: 11px; text-transform: uppercase; letter-spacing: 0.06em; color: #94a3b8; border-bottom: 1px solid rgba(255,255,255,0.07); white-space: nowrap; }
    thead th.sortable { cursor: pointer; user-select: none; }
    thead th.sortable:hover { color: #e2e8f0; }
    .sort-ind { display: inline-block; width: 12px; color: #76b900; }
    tbody tr { border-bottom: 1px solid rgba(255,255,255,0.04); transition: background 0.1s; }
    tbody tr:hover { background: rgba(255,255,255,0.03); }
    tbody td { padding: 9px 14px; vertical-align: middle; }
    .mono { font-family: 'JetBrains Mono', monospace; }
    .overdue-dot { width: 6px; height: 6px; border-radius: 50%; background: #ef4444; display: inline-block; animation: pulse-dot 1.4s infinite; }
    @keyframes pulse-dot { 0%,100%{opacity:1} 50%{opacity:.3} }
    .callout-card { background: linear-gradient(135deg, rgba(239,68,68,0.08) 0%, rgba(239,68,68,0.02) 100%); border: 1px solid rgba(239,68,68,0.25); border-radius: 12px; }
    input[type=text] { background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); color: #e2e8f0; border-radius: 8px; padding: 7px 14px; font-size: 13px; outline: none; }
    input[type=text]:focus { border-color: rgba(118,185,0,0.4); }
    select { background: rgba(20,20,30,0.95); border: 1px solid rgba(255,255,255,0.1); color: #e2e8f0; border-radius: 8px; padding: 7px 10px; font-size: 13px; outline: none; cursor: pointer; }
    ::-webkit-scrollbar { width: 6px; height: 6px; }
    ::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.12); border-radius: 3px; }
  </style>
</head>
<body>
<div class="min-h-screen px-6 py-5 max-w-[1600px] mx-auto">

  <header class="flex items-center justify-between mb-8 gap-4 flex-wrap">
    <div class="flex items-center gap-4">
      <div class="w-10 h-10 rounded-xl flex items-center justify-center font-black" style="background:linear-gradient(135deg,#76b900,#00c4ff)">
        <span style="-webkit-text-fill-color:#000;font-size:9px;letter-spacing:-.5px;font-weight:900">JWT</span>
      </div>
      <div>
        <h1 class="text-xl font-bold tracking-tight gradient-text">Team Jira Dashboard</h1>
        <p class="text-xs text-slate-500 mt-0.5">Home projects <code class="mono text-slate-400">{{.HomeProjects}}</code> &middot; Team <code class="mono text-slate-400">{{.TeamLabel}}</code></p>
      </div>
    </div>
    <div class="flex items-center gap-3 text-xs text-slate-500">
      <span>Fetched {{.FetchedAt}}</span>
      <span class="text-slate-600">&middot;</span>
      <a href="{{.BaseURL}}/issues/?jql={{.JQLEncoded}}" target="_blank"
         class="ml-2 px-3 py-1.5 rounded-lg text-xs font-semibold border border-slate-700 text-slate-300 hover:border-lime-600 hover:text-lime-400 transition-colors">
        Open in Jira &rarr;
      </a>
    </div>
  </header>

  <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-6">
    <a href="{{.BaseURL}}/issues/?jql={{.JQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Total Open</div>
      <div class="text-3xl font-black text-white">{{.TotalOpen}}</div>
      <div class="text-xs text-slate-600 mt-1">unresolved, all projects</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.HomeJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Home Backlog</div>
      <div class="text-3xl font-black" style="color:#76b900">{{.HomeCount}}</div>
      <div class="text-xs text-slate-600 mt-1">in {{.HomeProjects}}</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.ExternalJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">External Asks</div>
      <div class="text-3xl font-black text-sky-400">{{.ExternalCount}}</div>
      <div class="text-xs text-slate-600 mt-1">outside home projects</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.OverdueJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Overdue</div>
      <div class="text-3xl font-black text-red-400">{{.OverdueCount}}</div>
      <div class="text-xs text-slate-600 mt-1">past due date</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.UnassignedJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Unassigned</div>
      <div class="text-3xl font-black text-amber-400">{{.UnassignedCount}}</div>
      <div class="text-xs text-slate-600 mt-1">no assignee set</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.JQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Projects</div>
      <div class="text-3xl font-black text-violet-400">{{.ProjectCount}}</div>
      <div class="text-xs text-slate-600 mt-1">touched by the team</div>
    </a>
  </div>

  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
    <a href="{{.BaseURL}}/issues/?jql={{.AgeJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Oldest Open</div>
      <div class="text-3xl font-black text-rose-400">{{.OldestDays}}<span class="text-base font-semibold text-slate-500">d</span></div>
      <div class="text-xs text-slate-600 mt-1">age of the oldest issue</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.AgeJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Avg Age</div>
      <div class="text-3xl font-black text-slate-200">{{.AvgAgeDays}}<span class="text-base font-semibold text-slate-500">d</span></div>
      <div class="text-xs text-slate-600 mt-1">across all open issues</div>
    </a>
    <a href="{{.BaseURL}}/issues/?jql={{.HighPriorityJQLEncoded}}" target="_blank" class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">High Priority+</div>
      <div class="text-3xl font-black text-red-400">{{.HighPriorityCount}}</div>
      <div class="text-xs text-slate-600 mt-1">Highest/High open issues</div>
    </a>
    <div class="kpi-card p-4">
      <div class="text-xs text-slate-500 font-semibold uppercase tracking-widest mb-1.5">Overloaded</div>
      <div class="text-3xl font-black text-amber-400">{{.OverloadedCount}}</div>
      <div class="text-xs text-slate-600 mt-1">people carrying outsized load</div>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
    <div class="lg:col-span-2 card p-5">
      <h2 class="text-sm font-semibold text-slate-300 mb-4">Open Work by Assignee <span class="text-slate-500 font-normal text-xs">— who is doing what</span></h2>
      <div style="position:relative;height:280px"><canvas id="assigneeChart"></canvas></div>
    </div>
    <div class="card p-5">
      <h2 class="text-sm font-semibold text-slate-300 mb-4">Home vs External</h2>
      <div style="position:relative;height:240px"><canvas id="scopeChart"></canvas></div>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
    <div class="card p-5">
      <h2 class="text-sm font-semibold text-slate-300 mb-4">Distribution by Project</h2>
      <div style="position:relative;height:220px"><canvas id="projectChart"></canvas></div>
    </div>
    <div class="card p-5">
      <h2 class="text-sm font-semibold text-slate-300 mb-4">Distribution by Status</h2>
      <div style="position:relative;height:220px"><canvas id="statusChart"></canvas></div>
    </div>
    <div class="card p-5">
      <h2 class="text-sm font-semibold text-slate-300 mb-4">Distribution by Priority</h2>
      <div style="position:relative;height:220px"><canvas id="priorityChart"></canvas></div>
    </div>
  </div>

  <div class="mb-4">
    <h2 class="text-sm font-semibold text-slate-300 mb-3">Workload Callouts <span class="text-slate-500 font-normal text-xs">— assignees carrying well above the team's average open load</span></h2>
    <div id="calloutGrid" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3"></div>
  </div>

  <div class="card p-5 mb-4">
    <h2 class="text-sm font-semibold text-slate-300 mb-4">Assignee Breakdown <span class="text-slate-500 font-normal text-xs">— open workload per person</span></h2>
    <div class="overflow-x-auto">
      <table>
        <thead id="assigneeHead"><tr>
          <th class="sortable" data-field="name">Assignee<span class="sort-ind"></span></th>
          <th class="sortable" data-field="count">Open<span class="sort-ind"></span></th>
          <th>Projects</th>
          <th class="sortable" data-field="overdue">Overdue<span class="sort-ind"></span></th>
          <th>% of Total</th>
        </tr></thead>
        <tbody id="assigneeTable"></tbody>
      </table>
    </div>
  </div>

  <div class="card p-5 mb-4">
    <h2 class="text-sm font-semibold text-slate-300 mb-4">Oldest Open Items <span class="text-slate-500 font-normal text-xs">— top 10 by age, oldest first</span></h2>
    <div class="overflow-x-auto">
      <table>
        <thead id="oldestHead"><tr>
          <th class="sortable" data-field="key">Key<span class="sort-ind"></span></th>
          <th class="sortable" data-field="summary">Summary<span class="sort-ind"></span></th>
          <th class="sortable" data-field="project">Project<span class="sort-ind"></span></th>
          <th class="sortable" data-field="assignee">Assignee<span class="sort-ind"></span></th>
          <th class="sortable" data-field="priority">Priority<span class="sort-ind"></span></th>
          <th class="sortable" data-field="created">Created<span class="sort-ind"></span></th>
          <th class="sortable" data-field="ageDays">Age<span class="sort-ind"></span></th>
          <th></th>
        </tr></thead>
        <tbody id="oldestTable"></tbody>
      </table>
    </div>
  </div>

  <div class="card p-5">
    <div class="flex flex-wrap items-center justify-between gap-3 mb-4">
      <h2 class="text-sm font-semibold text-slate-300">Issue Log</h2>
      <div class="flex gap-2 flex-wrap">
        <input type="text" id="searchInput" placeholder="Filter by key, summary, assignee…" style="width:250px">
        <select id="scopeFilter" onchange="PAGE=0;renderTable()">
          <option value="">Home + External</option>
          <option value="home">Home only</option>
          <option value="external">External only</option>
        </select>
        <select id="assigneeFilter" onchange="PAGE=0;renderTable()"><option value="">All Assignees</option></select>
        <select id="statusFilter" onchange="PAGE=0;renderTable()"><option value="">All Statuses</option></select>
      </div>
    </div>
    <div class="overflow-x-auto">
      <table>
        <thead id="issueHead"><tr>
          <th class="sortable" data-field="key">Key<span class="sort-ind"></span></th>
          <th class="sortable" data-field="summary">Summary<span class="sort-ind"></span></th>
          <th class="sortable" data-field="project">Project<span class="sort-ind"></span></th>
          <th class="sortable" data-field="scope">Scope<span class="sort-ind"></span></th>
          <th class="sortable" data-field="assignee">Assignee<span class="sort-ind"></span></th>
          <th class="sortable" data-field="status">Status<span class="sort-ind"></span></th>
          <th class="sortable" data-field="priority">Priority<span class="sort-ind"></span></th>
          <th class="sortable" data-field="due">Due<span class="sort-ind"></span></th>
          <th></th>
        </tr></thead>
        <tbody id="issueTable"></tbody>
      </table>
      <div id="tablePager" class="flex items-center justify-between mt-3 text-xs text-slate-500 px-1"></div>
    </div>
  </div>

  <footer class="text-center mt-8 text-xs text-slate-700 pb-4">
    Team Jira Dashboard &middot; Generated {{.FetchedAt}} &middot;
    <a href="{{.BaseURL}}" class="hover:text-slate-400">{{.BaseURL}}</a>
  </footer>
</div>

<script>
// ── Data injected by Go ───────────────────────────────────────────────────────
const ASSIGNEE_DATA = {{.AssigneesJSON}};
const PROJECT_DATA   = {{.ProjectsJSON}};
const STATUS_DATA    = {{.StatusJSON}};
const PRIORITY_DATA  = {{.PriorityJSON}};
const OLDEST_DATA    = {{.OldestJSON}};
const CALLOUTS_DATA  = {{.CalloutsJSON}};
const ISSUES         = {{.IssuesJSON}};
const BASE_URL       = {{.BaseURLJSON}};

// ── Colour palette ───────────────────────────────────────────────────────────
const PALETTE = ['#38bdf8','#a78bfa','#84cc16','#fbbf24','#fb923c','#f472b6','#2dd4bf','#76b900','#f87171','#c084fc'];
function colorFor(key, list) { const i = list.indexOf(key); return PALETTE[i % PALETTE.length] || '#64748b'; }
function keyLink(key) {
  return '<a href="'+BASE_URL+'/browse/'+key+'" target="_blank" class="mono text-slate-300 hover:text-lime-400 hover:underline" style="font-size:inherit">'+key+'</a>';
}

// ── Sorting ────────────────────────────────────────────────────────────────────
const PRIORITY_ORDER = ['Highest','High','Medium','Low','Lowest'];
function priorityRank(p) {
  const i = PRIORITY_ORDER.findIndex(x => x.toLowerCase() === (p||'').toLowerCase());
  return i === -1 ? PRIORITY_ORDER.length : i;
}

// sortRows returns a new array sorted by field/dir. valueFn maps (row, field) -> comparable value;
// priority fields sort by rank order (Highest first), everything else falls back to string/number compare.
function sortRows(rows, field, dir, valueFn) {
  return rows.slice().sort((a, b) => {
    const av = valueFn(a, field), bv = valueFn(b, field);
    const cmp = (typeof av === 'number' && typeof bv === 'number')
      ? av - bv
      : String(av).localeCompare(String(bv));
    return cmp * dir;
  });
}

// attachSort wires click handlers onto a <thead>'s sortable <th data-field> cells. state is a
// mutable {field, dir} object; clicking the active column flips direction, clicking a new column
// resets to ascending. renderFn is called after every change to redraw the owning table.
function attachSort(theadId, state, renderFn) {
  const ths = document.querySelectorAll('#'+theadId+' th[data-field]');
  ths.forEach(th => {
    th.addEventListener('click', () => {
      const field = th.dataset.field;
      if (state.field === field) { state.dir *= -1; } else { state.field = field; state.dir = 1; }
      ths.forEach(h => { const ind = h.querySelector('.sort-ind'); if (ind) ind.textContent = ''; });
      const ind = th.querySelector('.sort-ind');
      if (ind) ind.textContent = state.dir === 1 ? '▲' : '▼';
      renderFn();
    });
  });
}

// ── Build filter dropdowns from data ──────────────────────────────────────────
(function() {
  const aSel = document.getElementById('assigneeFilter');
  ASSIGNEE_DATA.slice().sort((a,b) => b.count-a.count).forEach(a => {
    const o = document.createElement('option');
    o.value = a.name; o.textContent = a.name + ' (' + a.count + ')';
    aSel.appendChild(o);
  });
  const sSel = document.getElementById('statusFilter');
  Object.keys(STATUS_DATA).sort((a,b) => STATUS_DATA[b]-STATUS_DATA[a]).forEach(s => {
    const o = document.createElement('option');
    o.value = s; o.textContent = s;
    sSel.appendChild(o);
  });
})();

// ── Chart defaults ────────────────────────────────────────────────────────────
Chart.defaults.color = '#94a3b8';
Chart.defaults.font  = {family:"'Inter',system-ui,sans-serif", size:11};

// Assignee bar
(function(){
  const rows = ASSIGNEE_DATA.slice().sort((a,b) => b.count-a.count);
  const names = rows.map(a => a.name);
  new Chart(document.getElementById('assigneeChart').getContext('2d'), {
    type: 'bar',
    data: {
      labels: names,
      datasets: [{
        data: rows.map(a => a.count),
        backgroundColor: names.map(n => colorFor(n, names) + '44'),
        borderColor:     names.map(n => colorFor(n, names)),
        borderWidth: 1.5, borderRadius: 4,
      }]
    },
    options: {
      indexAxis: 'y', responsive: true, maintainAspectRatio: false,
      plugins: { legend: {display:false}, tooltip: {callbacks:{label:c=>' ' + c.raw + ' open issues'}} },
      scales: {
        x: { grid:{color:'rgba(255,255,255,0.04)'}, ticks:{color:'#64748b'} },
        y: { grid:{display:false}, ticks:{color:'#94a3b8', font:{size:10.5,family:"'JetBrains Mono',monospace"}} }
      }
    }
  });
})();

// Home vs External doughnut
(function(){
  const home = ISSUES.filter(i => i.scope === 'home').length;
  const ext  = ISSUES.filter(i => i.scope === 'external').length;
  new Chart(document.getElementById('scopeChart').getContext('2d'), {
    type: 'doughnut',
    data: {
      labels: ['Home', 'External'],
      datasets: [{
        data: [home, ext],
        backgroundColor: ['#76b90088', '#38bdf888'],
        borderColor:     ['#76b900', '#38bdf8'],
        borderWidth: 1.5, hoverOffset: 6,
      }]
    },
    options: {
      responsive: true, maintainAspectRatio: false, cutout: '65%',
      plugins: {
        legend: {position:'right', labels:{boxWidth:10, boxHeight:10, padding:12}},
        tooltip: {callbacks:{label:c=>' ' + c.label + ': ' + c.raw + ' issues'}}
      }
    }
  });
})();

// Project doughnut
(function(){
  const keys = Object.keys(PROJECT_DATA);
  new Chart(document.getElementById('projectChart').getContext('2d'), {
    type: 'doughnut',
    data: {
      labels: keys,
      datasets: [{
        data: keys.map(k => PROJECT_DATA[k]),
        backgroundColor: keys.map(k => colorFor(k, keys) + '88'),
        borderColor:     keys.map(k => colorFor(k, keys)),
        borderWidth: 1.5, hoverOffset: 6,
      }]
    },
    options: {
      responsive: true, maintainAspectRatio: false, cutout: '60%',
      plugins: {
        legend: {position:'right', labels:{boxWidth:10, boxHeight:10, padding:10, font:{size:10.5,family:"'JetBrains Mono',monospace"}}},
        tooltip: {callbacks:{label:c=>' ' + c.label + ': ' + c.raw + ' issues'}}
      }
    }
  });
})();

// Status bar
(function(){
  const keys = Object.keys(STATUS_DATA).sort((a,b) => STATUS_DATA[b]-STATUS_DATA[a]);
  new Chart(document.getElementById('statusChart').getContext('2d'), {
    type: 'bar',
    data: {
      labels: keys,
      datasets: [{
        data: keys.map(k => STATUS_DATA[k]),
        backgroundColor: keys.map(k => colorFor(k, keys) + '44'),
        borderColor:     keys.map(k => colorFor(k, keys)),
        borderWidth: 1.5, borderRadius: 4,
      }]
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: {display:false}, tooltip: {callbacks:{label:c=>' ' + c.raw + ' issues'}} },
      scales: {
        x: { grid:{display:false}, ticks:{color:'#94a3b8', maxRotation:20} },
        y: { grid:{color:'rgba(255,255,255,0.04)'}, ticks:{color:'#64748b'}, beginAtZero:true }
      }
    }
  });
})();

// Priority bar
(function(){
  const keys = Object.keys(PRIORITY_DATA).sort((a,b) => priorityRank(a) - priorityRank(b));
  const col = k => { const l = k.toLowerCase();
    if (l.includes('highest') || l === 'high') return '#f87171';
    if (l.includes('medium')) return '#fbbf24';
    return '#38bdf8'; };
  new Chart(document.getElementById('priorityChart').getContext('2d'), {
    type: 'bar',
    data: {
      labels: keys,
      datasets: [{
        data: keys.map(k => PRIORITY_DATA[k]),
        backgroundColor: keys.map(k => col(k) + '44'),
        borderColor:     keys.map(k => col(k)),
        borderWidth: 1.5, borderRadius: 4,
      }]
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: {display:false}, tooltip: {callbacks:{label:c=>' ' + c.raw + ' issues'}} },
      scales: {
        x: { grid:{display:false}, ticks:{color:'#94a3b8'} },
        y: { grid:{color:'rgba(255,255,255,0.04)'}, ticks:{color:'#64748b'}, beginAtZero:true }
      }
    }
  });
})();

// Workload callouts
(function(){
  const grid = document.getElementById('calloutGrid');
  if (CALLOUTS_DATA.length === 0) {
    grid.innerHTML = '<div class="card p-4 text-xs text-slate-500">No assignee is carrying an outsized share of open work right now.</div>';
    return;
  }
  grid.innerHTML = CALLOUTS_DATA.map(c => {
    return '<div class="callout-card p-4">'
      + '<div class="flex items-center justify-between mb-1.5">'
      +   '<span class="text-sm font-semibold text-slate-100">'+c.name+'</span>'
      +   '<span class="badge priority-highest">+'+c.pctAboveAvg+'%</span>'
      + '</div>'
      + '<div class="text-2xl font-black text-red-400">'+c.count+'<span class="text-xs font-medium text-slate-500 ml-1">open issues</span></div>'
      + '<div class="text-xs text-slate-500 mt-1">team average is '+c.teamAvg+' &middot; '+c.projects.join(', ')+'</div>'
      + '</div>';
  }).join('');
})();

// Oldest open items table
const OLDEST_SORT = { field: null, dir: 1 };
function oldestSortValue(i, field) {
  if (field === 'priority') return priorityRank(i.priority);
  if (field === 'ageDays') return i.ageDays;
  return (i[field] || '').toString().toLowerCase();
}
function renderOldestTable() {
  const rows = OLDEST_SORT.field
    ? sortRows(OLDEST_DATA, OLDEST_SORT.field, OLDEST_SORT.dir, oldestSortValue)
    : OLDEST_DATA;
  document.getElementById('oldestTable').innerHTML = rows.map(i => {
    return '<tr>'
      + '<td class="text-xs">'+keyLink(i.key)+'</td>'
      + '<td class="text-slate-200 text-xs font-medium" style="max-width:320px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">'+i.summary+'</td>'
      + '<td class="mono text-slate-400 text-xs">'+i.project+'</td>'
      + '<td class="text-slate-300 text-xs">'+i.assignee+'</td>'
      + '<td><span class="badge '+priorityClass(i.priority)+'">'+i.priority+'</span></td>'
      + '<td class="text-slate-400 text-xs mono">'+i.created+'</td>'
      + '<td class="text-rose-400 text-xs mono font-semibold">'+i.ageDays+'d</td>'
      + '<td><a href="'+BASE_URL+'/browse/'+i.key+'" target="_blank" style="color:#334155;font-size:12px" onmouseover="this.style.color=\'#84cc16\'" onmouseout="this.style.color=\'#334155\'">&#8599;</a></td>'
      + '</tr>';
  }).join('');
}
renderOldestTable();
attachSort('oldestHead', OLDEST_SORT, renderOldestTable);

// Assignee breakdown table
const ASSIGNEE_SORT = { field: null, dir: 1 };
function assigneeSortValue(a, field) {
  if (field === 'count' || field === 'overdue') return a[field];
  return (a[field] || '').toString().toLowerCase();
}
function renderAssigneeTable() {
  const tot = ASSIGNEE_DATA.reduce((s,a) => s + a.count, 0) || 1;
  const base = ASSIGNEE_DATA.slice().sort((a,b) => b.count-a.count);
  const rows = ASSIGNEE_SORT.field
    ? sortRows(base, ASSIGNEE_SORT.field, ASSIGNEE_SORT.dir, assigneeSortValue)
    : base;
  document.getElementById('assigneeTable').innerHTML = rows.map(a => {
    const pct = Math.round(a.count / tot * 100);
    const overdueHtml = a.overdue > 0
      ? '<span class="overdue-dot mr-1.5"></span><span style="color:#f87171;font-weight:600">'+a.overdue+'</span>'
      : '<span style="color:#334155">—</span>';
    return '<tr>'
      + '<td class="mono text-slate-200 font-medium">'+a.name+'</td>'
      + '<td class="text-white font-bold text-base">'+a.count+'</td>'
      + '<td class="mono text-slate-400 text-xs">'+a.projects.join(', ')+'</td>'
      + '<td>'+overdueHtml+'</td>'
      + '<td><div class="flex items-center gap-2">'
      +   '<div style="width:80px;height:6px;border-radius:3px;background:rgba(255,255,255,0.06);overflow:hidden">'
      +     '<div style="height:100%;width:'+pct+'%;background:linear-gradient(90deg,#76b900,#00c4ff);border-radius:3px"></div>'
      +   '</div>'
      +   '<span style="color:#64748b;font-size:11px">'+pct+'%</span>'
      + '</div></td>'
      + '</tr>';
  }).join('');
}
renderAssigneeTable();
attachSort('assigneeHead', ASSIGNEE_SORT, renderAssigneeTable);

// Issue log
let PAGE = 0;
const PAGE_SIZE = 25;

function getFiltered() {
  const q        = document.getElementById('searchInput').value.toLowerCase();
  const scope    = document.getElementById('scopeFilter').value;
  const assignee = document.getElementById('assigneeFilter').value;
  const status   = document.getElementById('statusFilter').value;
  return ISSUES.filter(i => {
    const hit = !q || i.key.toLowerCase().includes(q) || i.summary.toLowerCase().includes(q)
              || i.assignee.toLowerCase().includes(q) || i.project.toLowerCase().includes(q);
    return hit && (!scope || i.scope === scope) && (!assignee || i.assignee === assignee) && (!status || i.status === status);
  });
}

function statusClass(s) {
  const l = s.toLowerCase();
  if (l.includes('progress') || l.includes('review')) return l.includes('review') ? 'status-review' : 'status-progress';
  if (l.includes('done') || l.includes('closed') || l.includes('resolved')) return 'status-done';
  return 'status-open';
}
function priorityClass(p) {
  const l = p.toLowerCase();
  if (l.includes('highest') || l.includes('high')) return 'priority-highest';
  if (l.includes('medium')) return 'priority-medium';
  return 'priority-low';
}

const ISSUE_SORT = { field: null, dir: 1 };
function issueSortValue(i, field) {
  if (field === 'priority') return priorityRank(i.priority);
  if (field === 'due') return i.due || '9999-12-31';
  return (i[field] || '').toString().toLowerCase();
}

function renderTable() {
  let filtered = getFiltered();
  if (ISSUE_SORT.field) filtered = sortRows(filtered, ISSUE_SORT.field, ISSUE_SORT.dir, issueSortValue);
  const page = filtered.slice(PAGE * PAGE_SIZE, (PAGE+1) * PAGE_SIZE);

  document.getElementById('issueTable').innerHTML = page.map(i => {
    const scopeCol = i.scope === 'home' ? '#76b900' : '#38bdf8';
    const overdue = i.overdue ? '<span class="overdue-dot mr-1"></span>' : '';
    return '<tr>'
      + '<td class="text-xs">'+keyLink(i.key)+'</td>'
      + '<td class="text-slate-200 text-xs font-medium" style="max-width:320px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">'+i.summary+'</td>'
      + '<td class="mono text-slate-400 text-xs">'+i.project+'</td>'
      + '<td><span class="scope-pill" style="background:'+scopeCol+'22;color:'+scopeCol+';border:1px solid '+scopeCol+'44">'+i.scope+'</span></td>'
      + '<td class="text-slate-300 text-xs">'+i.assignee+'</td>'
      + '<td><span class="badge '+statusClass(i.status)+'">'+i.status+'</span></td>'
      + '<td><span class="badge '+priorityClass(i.priority)+'">'+i.priority+'</span></td>'
      + '<td class="text-slate-400 text-xs mono">'+overdue+(i.due||'—')+'</td>'
      + '<td><a href="'+BASE_URL+'/browse/'+i.key+'" target="_blank" style="color:#334155;font-size:12px" onmouseover="this.style.color=\'#84cc16\'" onmouseout="this.style.color=\'#334155\'">&#8599;</a></td>'
      + '</tr>';
  }).join('');

  const s = PAGE*PAGE_SIZE+1, e = Math.min((PAGE+1)*PAGE_SIZE, filtered.length);
  document.getElementById('tablePager').innerHTML =
    '<span>'+filtered.length+' issues &middot; showing '+(filtered.length ? s : 0)+'&ndash;'+e+'</span>'
    + '<div style="display:flex;gap:8px">'
    + (PAGE > 0 ? '<button onclick="changePage(-1)" style="padding:4px 12px;border:1px solid #334155;border-radius:6px;color:#94a3b8;background:none;cursor:pointer">&#8592; Prev</button>' : '')
    + (e < filtered.length ? '<button onclick="changePage(1)" style="padding:4px 12px;border:1px solid #334155;border-radius:6px;color:#94a3b8;background:none;cursor:pointer">Next &#8594;</button>' : '')
    + '</div>';
}

function changePage(d) { PAGE = Math.max(0, PAGE+d); renderTable(); }
document.getElementById('searchInput').addEventListener('input', () => { PAGE=0; renderTable(); });
attachSort('issueHead', ISSUE_SORT, () => { PAGE = 0; renderTable(); });
renderTable();
</script>
</body>
</html>`
