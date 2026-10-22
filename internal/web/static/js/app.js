// Main Modular Entry Point & Left Sidebar Router for GORT Command Center
import { fetchCatalog, fetchConfigList, loadConfigFile, saveConfigFile } from "./api.js";
import {
  appState,
  subscribe,
  notify,
  serializeConfigToYAML,
  estimateTotalTestCases,
  createBlankConfig,
  createDefaultConfig,
} from "./state.js";
import { renderWizardView } from "./views/wizard.js";
import { renderCatalogView } from "./views/catalog.js";
import { renderDashboardView } from "./views/dashboard.js";
import { renderRunsView } from "./views/runs.js";
import { renderLiveView } from "./views/live.js";
import { renderReportsView } from "./views/reports.js";

let currentRoute = "wizard";

const ROUTE_META = {
  wizard: {
    title: "Interactive Config Builder Wizard",
    subtitle: "Step-by-step configuration for Target/Attacker/Judge LLMs, Prompt Bank datasets, 155 plugins, and strategies",
  },
  catalog: {
    title: "SQLite Plugin Catalog & Prompt Bank Explorer",
    subtitle: "Explore all 155 built-in redteam plugins, 14 compliance collections, and 20 benchmark datasets",
  },
  dashboard: {
    title: "Executive Security Command Center",
    subtitle: "Posture KPIs, strategy bypass analytics, and OWASP LLM Top 10 risk heatmaps (Step 6.2)",
  },
  runs: {
    title: "Assessment Runs & Multi-Turn Transcripts",
    subtitle: "Inspect SQLite eval_runs, side-by-side prompt mutations, Crescendo [BACKTRACKED] turns, and OTel spans (Step 6.2)",
  },
  live: {
    title: "Live Attack Launcher & Real-Time Log Stream",
    subtitle: "Trigger assessments from the browser and stream .gort/gort.log via SSE (Step 6.3)",
  },
  reports: {
    title: "Executive Audit & Compliance Reports",
    subtitle: "Generate standalone printable HTML/PDF Executive Security Reports (Step 6.4)",
  },
};

function showToast(msg) {
  const banner = document.getElementById("toast-banner");
  banner.textContent = msg;
  banner.classList.remove("hidden");
  setTimeout(() => banner.classList.add("hidden"), 4000);
}

function navigateToRoute(route) {
  currentRoute = route;
  document.querySelectorAll(".nav-item").forEach((b) => {
    b.classList.toggle("active", b.dataset.route === route);
  });
  renderActiveRoute();
}

function renderActiveRoute() {
  const meta = ROUTE_META[currentRoute] || ROUTE_META.wizard;
  document.getElementById("page-title").textContent = meta.title;
  document.getElementById("page-subtitle").textContent = meta.subtitle;

  const container = document.getElementById("view-container");
  if (currentRoute === "wizard") {
    renderWizardView(container, showToast);
  } else if (currentRoute === "catalog") {
    renderCatalogView(container, showToast);
  } else if (currentRoute === "dashboard") {
    renderDashboardView(container, showToast, navigateToRoute);
  } else if (currentRoute === "runs") {
    renderRunsView(container, showToast);
  } else if (currentRoute === "live") {
    renderLiveView(container, showToast);
  } else if (currentRoute === "reports") {
    renderReportsView(container, showToast);
  } else {
    container.innerHTML = `
      <div class="card">
        <div class="card-header"><span class="card-title">${meta.title}</span></div>
        <p style="color:var(--text-secondary);">${meta.subtitle} — Ready to be activated in the next Phase 6 micro-step.</p>
      </div>
    `;
  }
}

function updateLiveYAMLPanel() {
  const editor = document.getElementById("yaml-preview-editor");
  editor.value = serializeConfigToYAML(appState.config);
  const est = estimateTotalTestCases(appState.config);
  document.getElementById("case-estimator").textContent =
    `${est.seeds} seeds × ${est.strategies} strategies = ${est.total} cases`;
}

async function refreshConfigDropdown() {
  const data = await fetchConfigList();
  appState.configFiles = data.files || [];
  const sel = document.getElementById("config-file-select");
  const loadBtn = document.getElementById("btn-reload-config");
  if (appState.configFiles.length === 0) {
    sel.innerHTML = `<option value="">(No YAML files on disk)</option>`;
    sel.disabled = true;
    if (loadBtn) loadBtn.disabled = true;
  } else {
    sel.disabled = false;
    if (loadBtn) loadBtn.disabled = false;
    sel.innerHTML = appState.configFiles
      .map((f) => `<option value="${f}" ${f === appState.activeFile ? "selected" : ""}>${f}</option>`)
      .join("");
  }
}

async function initApp() {
  subscribe(() => updateLiveYAMLPanel());

  document.querySelectorAll(".nav-item").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".nav-item").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      currentRoute = btn.dataset.route;
      renderActiveRoute();
    });
  });

  document.getElementById("btn-toggle-yaml").addEventListener("click", () => {
    document.querySelector(".app-shell").classList.toggle("yaml-collapsed");
  });

  document.getElementById("btn-reload-config").addEventListener("click", async () => {
    const file = document.getElementById("config-file-select").value;
    if (!file) return;
    try {
      const loaded = await loadConfigFile(file);
      appState.activeFile = loaded.filename;
      appState.config = loaded.config;
      document.getElementById("save-filename-input").value = loaded.filename;
      notify();
      renderActiveRoute();
      showToast(`Loaded configuration from ${loaded.filename}`);
    } catch (err) {
      showToast(`Error loading ${file}: ${err.message}`);
    }
  });

  document.getElementById("btn-new-blank-config").addEventListener("click", () => {
    appState.config = createBlankConfig();
    appState.activeFile = "config.yaml";
    document.getElementById("save-filename-input").value = "config.yaml";
    notify();
    renderActiveRoute();
    showToast("Started fresh blank configuration! (All fields cleared)");
  });

  document.getElementById("btn-starter-template").addEventListener("click", () => {
    appState.config = createDefaultConfig();
    notify();
    renderActiveRoute();
    showToast("Loaded recommended starter template (Qwen 2.5 7B, 2 plugins, 3 strategies)!");
  });

  document.getElementById("btn-save-yaml").addEventListener("click", async () => {
    const filename = document.getElementById("save-filename-input").value.trim() || "config.yaml";
    const rawYaml = document.getElementById("yaml-preview-editor").value;
    try {
      const res = await saveConfigFile(filename, rawYaml);
      appState.activeFile = res.saved;
      await refreshConfigDropdown();
      showToast(`Saved ${res.saved} (${res.bytes} bytes) to project workspace!`);
    } catch (err) {
      showToast(`Save failed: ${err.message}`);
    }
  });

  document.getElementById("btn-copy-yaml").addEventListener("click", async () => {
    const rawYaml = document.getElementById("yaml-preview-editor").value;
    await navigator.clipboard.writeText(rawYaml);
    showToast("Copied config.yaml to clipboard!");
  });

  appState.catalog = await fetchCatalog();
  const totalPrompts = (appState.catalog.datasets || []).reduce((acc, d) => acc + (d.promptCount ?? d.PromptCount ?? 0), 0);
  document.getElementById("sidebar-stats").innerHTML = `
    <div class="stat-pill">SQLite Ready: <strong>${(appState.catalog.plugins || []).length}</strong> plugins • <strong>${Object.keys(appState.catalog.collections || {}).length}</strong> cols • <strong>${(appState.catalog.datasets || []).length}</strong> sets (${totalPrompts} prompts)</div>
  `;

  await refreshConfigDropdown();

  // If config.yaml exists on disk, auto-load it; otherwise start clean blank
  if (appState.configFiles.includes("config.yaml")) {
    try {
      const loaded = await loadConfigFile("config.yaml");
      appState.activeFile = loaded.filename;
      appState.config = loaded.config;
      document.getElementById("save-filename-input").value = loaded.filename;
      showToast(`Auto-loaded active ${loaded.filename} from disk`);
    } catch (e) {
      console.warn("Auto-load config.yaml failed:", e);
      appState.config = createBlankConfig();
    }
  } else {
    appState.config = createBlankConfig();
    appState.activeFile = "config.yaml";
    document.getElementById("save-filename-input").value = "config.yaml";
  }

  notify();
  renderActiveRoute();
}

initApp();
