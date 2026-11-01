// Live Attack Launcher & Real-Time Log Streaming View (Phase 6.3)
import {
  fetchConfigList,
  fetchCatalog,
  loadConfigFile,
  validateConfig,
  startAssessment,
  stopAssessment,
  fetchAssessmentStatus,
  streamAssessmentLogs,
  fetchLogTail,
} from "../api.js";
import { appState, serializeConfigToYAML } from "../state.js";
import { openValidationModal } from "../validator_ui.js";

let logLines = [];
let eventSource = null;
let pollInterval = null;
let autoScroll = true;
let filterQuery = "";
let showToastFn = null;

export async function renderLiveView(container, showToast) {
  showToastFn = showToast;
  teardownActiveStream();
  container.innerHTML = buildLiveHTML();
  initControls();
  await loadInitialData();
  await refreshStatus();
  initLogStream();
}

function buildLiveHTML() {
  return `
    <div class="card" style="margin-bottom: 16px;">
      <div class="card-header">
        <span class="card-title">Launch Controls</span>
        <div id="live-status-pill" class="pill pill-info">Status: Checking...</div>
      </div>
      <div class="form-grid" style="grid-template-columns: 1.5fr 1fr auto auto; align-items: flex-end;">
        <div class="form-group">
          <label for="live-config-select">Evaluation Configuration</label>
          <select id="live-config-select" class="input-select">
            <option value="__active_wizard__">Current Wizard Configuration (In-Memory)</option>
          </select>
        </div>
        <div class="form-group">
          <label for="live-dataset-select">Target Dataset Scope</label>
          <select id="live-dataset-select" class="input-select">
            <option value="config">Configured in YAML</option>
            <option value="all">All Available SQLite Datasets</option>
          </select>
        </div>
        <div>
          <button id="btn-start-live" class="btn btn-primary" style="min-width: 130px;">
            ▶ Launch Attack
          </button>
        </div>
        <div>
          <button id="btn-stop-live" class="btn btn-danger" style="min-width: 90px;" disabled>
            ■ Stop
          </button>
        </div>
      </div>
    </div>

    <!-- Live Telemetry / HUD Card -->
    <div class="kpi-grid" style="grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); margin-bottom: 16px;">
      <div class="kpi-card">
        <span class="kpi-label">Active Target</span>
        <span id="hud-target" class="kpi-value" style="font-size: 16px; color: var(--accent-primary);">—</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Target Dataset</span>
        <span id="hud-dataset" class="kpi-value" style="font-size: 16px;">—</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Cases Completed</span>
        <span id="hud-progress" class="kpi-value" style="font-size: 16px;">0 / 0</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Pass / Fail</span>
        <span id="hud-scores" class="kpi-value" style="font-size: 16px; color: #10b981;">0 / 0</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Elapsed Time</span>
        <span id="hud-duration" class="kpi-value" style="font-size: 16px;">0s</span>
      </div>
    </div>

    <!-- Live Terminal Stream Console -->
    <div class="terminal-window">
      <div class="terminal-header">
        <div class="terminal-title">
          <span id="terminal-dot" class="terminal-dot idle"></span>
          <span>Live Console Stream (.gort/gort.log)</span>
        </div>
        <div style="display:flex; align-items:center; gap:10px;">
          <input
            id="terminal-filter"
            type="text"
            placeholder="Filter logs (e.g. FAIL, Crescendo)..."
            class="input-text"
            style="width:240px; padding:4px 8px; font-size:11px;"
          />
          <button id="btn-toggle-scroll" class="btn btn-secondary" style="padding:4px 10px; font-size:11px;">
            Auto-Scroll: ON
          </button>
          <button id="btn-clear-terminal" class="btn btn-secondary" style="padding:4px 10px; font-size:11px;">
            Clear
          </button>
        </div>
      </div>
      <div id="terminal-body" class="terminal-body">
        <div class="terminal-line system">[SYSTEM] Connecting to live event stream...</div>
      </div>
    </div>
  `;
}

function initControls() {
  document.getElementById("btn-start-live").addEventListener("click", onStartAssessment);
  document.getElementById("btn-stop-live").addEventListener("click", onStopAssessment);
  document.getElementById("btn-clear-terminal").addEventListener("click", onClearTerminal);
  document.getElementById("btn-toggle-scroll").addEventListener("click", onToggleAutoScroll);
  document.getElementById("terminal-filter").addEventListener("input", onFilterChange);
}

async function loadInitialData() {
  await Promise.all([populateConfigFiles(), populateDatasets()]);
}

async function populateConfigFiles() {
  try {
    const data = await fetchConfigList();
    const sel = document.getElementById("live-config-select");
    (data.files || []).forEach((f) => {
      const opt = document.createElement("option");
      opt.value = f;
      opt.textContent = `File: ${f}`;
      sel.appendChild(opt);
    });
  } catch (err) {
    console.error("Failed to load configs", err);
  }
}

async function populateDatasets() {
  try {
    const data = await fetchCatalog();
    const sel = document.getElementById("live-dataset-select");
    (data.datasets || []).forEach((d) => {
      const opt = document.createElement("option");
      opt.value = d.name;
      opt.textContent = `Dataset: ${d.name} (${d.prompt_count} prompts)`;
      sel.appendChild(opt);
    });
  } catch (err) {
    console.error("Failed to load datasets", err);
  }
}

async function onStartAssessment() {
  const configSel = document.getElementById("live-config-select").value;
  const datasetSel = document.getElementById("live-dataset-select").value;
  const payload = { dataset: datasetSel };

  try {
    let yamlToCheck = "";
    if (configSel === "__active_wizard__") {
      yamlToCheck = document.getElementById("yaml-preview-editor")?.value || serializeConfigToYAML(appState.config);
      payload.rawYaml = yamlToCheck;
    } else {
      const loaded = await loadConfigFile(configSel);
      yamlToCheck = loaded.rawYaml;
      payload.configFile = configSel;
    }
    const valRes = await validateConfig(yamlToCheck);
    if (!valRes.report.valid || valRes.report.needsAttackerResolution) {
      document.getElementById("yaml-preview-editor").value = yamlToCheck;
      await openValidationModal(showToastFn);
      return;
    }

    const res = await startAssessment(payload);
    showToastFn(res.message || "Assessment launched successfully!");
    appendLog(`[SYSTEM] Assessment launched with dataset: ${datasetSel}`);
    await refreshStatus();
  } catch (err) {
    showToastFn(`Launch failed: ${err.message}`);
  }
}

async function onStopAssessment() {
  try {
    const res = await stopAssessment();
    showToastFn(res.message || "Assessment stopped.");
    appendLog("[SYSTEM] Assessment stopped by operator.");
    await refreshStatus();
  } catch (err) {
    showToastFn(`Stop failed: ${err.message}`);
  }
}

function onClearTerminal() {
  logLines = [];
  renderLogs();
}

function onToggleAutoScroll() {
  autoScroll = !autoScroll;
  const btn = document.getElementById("btn-toggle-scroll");
  btn.textContent = `Auto-Scroll: ${autoScroll ? "ON" : "OFF"}`;
  btn.classList.toggle("btn-primary", autoScroll);
  btn.classList.toggle("btn-secondary", !autoScroll);
}

function onFilterChange(e) {
  filterQuery = (e.target.value || "").toLowerCase();
  renderLogs();
}

async function refreshStatus() {
  try {
    const st = await fetchAssessmentStatus();
    updateHUD(st);
    updateButtonsAndPill(st);
    managePolling(st.running);
  } catch (err) {
    console.error("Failed to fetch status", err);
  }
}

function updateHUD(st) {
  document.getElementById("hud-target").textContent = st.targetId || "—";
  document.getElementById("hud-dataset").textContent = st.dataset || "—";
  document.getElementById("hud-progress").textContent = `${st.currentCase} / ${st.totalCases}`;
  document.getElementById("hud-scores").textContent = `${st.passedTests} P / ${st.failedTests} F`;
  document.getElementById("hud-duration").textContent = formatDuration(st.durationMs);
}

function updateButtonsAndPill(st) {
  const startBtn = document.getElementById("btn-start-live");
  const stopBtn = document.getElementById("btn-stop-live");
  const pill = document.getElementById("live-status-pill");
  const dot = document.getElementById("terminal-dot");

  startBtn.disabled = st.running;
  stopBtn.disabled = !st.running;

  if (st.running) {
    pill.className = "pill pill-success";
    pill.textContent = "Status: Running Attack";
    dot.className = "terminal-dot active";
  } else if (st.errorMessage) {
    pill.className = "pill pill-danger";
    pill.textContent = `Status: Error (${st.errorMessage})`;
    dot.className = "terminal-dot idle";
  } else {
    pill.className = "pill pill-info";
    pill.textContent = "Status: Idle";
    dot.className = "terminal-dot idle";
  }
}

function managePolling(isRunning) {
  if (isRunning && !pollInterval) {
    pollInterval = setInterval(refreshStatus, 1500);
  } else if (!isRunning && pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }
}

function formatDuration(ms) {
  if (!ms || ms <= 0) return "0s";
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  const remSec = sec % 60;
  return `${min}m ${remSec}s`;
}

function initLogStream() {
  loadTailLogs();
  eventSource = streamAssessmentLogs(
    (msg) => appendLog(msg),
    (err) => console.warn("SSE connection retry...", err)
  );
}

async function loadTailLogs() {
  try {
    const data = await fetchLogTail(80);
    if (data.lines && data.lines.length > 0) {
      logLines = data.lines.slice();
      renderLogs();
    }
  } catch (err) {
    console.error("Failed to load initial log tail", err);
  }
}

function appendLog(line) {
  logLines.push(line);
  if (logLines.length > 2000) {
    logLines.shift();
  }
  appendSingleLineToDOM(line);
}

function appendSingleLineToDOM(line) {
  if (filterQuery && !line.toLowerCase().includes(filterQuery)) {
    return;
  }
  const body = document.getElementById("terminal-body");
  if (!body) return;
  const lineEl = createLineElement(line);
  body.appendChild(lineEl);
  if (autoScroll) {
    body.scrollTop = body.scrollHeight;
  }
}

function renderLogs() {
  const body = document.getElementById("terminal-body");
  if (!body) return;
  body.innerHTML = "";
  const filtered = filterQuery
    ? logLines.filter((l) => l.toLowerCase().includes(filterQuery))
    : logLines;

  const frag = document.createDocumentFragment();
  filtered.forEach((l) => frag.appendChild(createLineElement(l)));
  body.appendChild(frag);
  if (autoScroll) {
    body.scrollTop = body.scrollHeight;
  }
}

function createLineElement(line) {
  const el = document.createElement("div");
  el.className = `terminal-line ${classifyLine(line)}`;
  el.textContent = line;
  return el;
}

function classifyLine(line) {
  if (line.includes("[FAIL]") || line.includes("FAILED")) return "fail";
  if (line.includes("[PASS]") || line.includes("PASSED")) return "pass";
  if (line.includes("[ERROR]") || line.includes("error:")) return "error";
  if (line.includes("[WARN]")) return "warn";
  if (line.includes("[INFO]")) return "info";
  if (line.includes("[SYSTEM]")) return "system";
  return "";
}

function teardownActiveStream() {
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }
}
