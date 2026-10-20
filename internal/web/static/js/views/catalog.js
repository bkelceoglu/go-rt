// Modular Plugins & Prompt Bank Explorer View
import { appState, notify } from "../state.js";
import { fetchDatasetPrompts } from "../api.js";

let activeSubTab = "plugins";
let searchFilter = "";
let previewDataset = "";
let previewRows = [];

export function renderCatalogView(container, showToast) {
  const plugins = appState.catalog.plugins || [];
  const datasets = appState.catalog.datasets || [];

  container.innerHTML = `
    <div class="wizard-stepper" style="max-width:520px;">
      <button class="step-tab ${activeSubTab === "plugins" ? "active" : ""}" id="tab-cat-plugins">
        <span class="step-num">155 Built-in</span>
        <span class="step-label">Redteam Plugins</span>
      </button>
      <button class="step-tab ${activeSubTab === "datasets" ? "active" : ""}" id="tab-cat-datasets">
        <span class="step-num">${datasets.length} Named Sets</span>
        <span class="step-label">SQLite Prompt Bank</span>
      </button>
    </div>
    <div id="catalog-subview"></div>
  `;

  container.querySelector("#tab-cat-plugins").addEventListener("click", () => {
    activeSubTab = "plugins";
    renderCatalogView(container, showToast);
  });
  container.querySelector("#tab-cat-datasets").addEventListener("click", () => {
    activeSubTab = "datasets";
    renderCatalogView(container, showToast);
  });

  const sub = container.querySelector("#catalog-subview");
  if (activeSubTab === "plugins") renderPluginsTable(sub, plugins, showToast);
  else renderDatasetsTable(sub, datasets, container, showToast);
}

function renderPluginsTable(sub, plugins, showToast) {
  const filtered = plugins.filter((p) =>
    !searchFilter || p.ID.toLowerCase().includes(searchFilter) || p.Category.toLowerCase().includes(searchFilter) || p.Description.toLowerCase().includes(searchFilter)
  );
  sub.innerHTML = `
    <div class="card">
      <div class="card-header">
        <span class="card-title">SQLite Redteam Plugin Catalog (${filtered.length} plugins)</span>
        <input id="cat-search" class="input-text" style="width:260px;" placeholder="Search plugin ID, category, desc..." value="${searchFilter}" />
      </div>
      <div style="max-height:540px;overflow-y:auto;">
        <table class="data-table">
          <thead><tr><th>#</th><th>Plugin ID</th><th>Category</th><th>Severity</th><th>Exempt</th><th>Description</th><th>Action</th></tr></thead>
          <tbody>
            ${filtered.map((p, idx) => `
              <tr>
                <td>${idx + 1}</td>
                <td><code>${p.ID}</code></td>
                <td><span class="badge badge-info">${p.Category}</span></td>
                <td><span class="badge badge-${p.Severity}">${p.Severity}</span></td>
                <td>${p.StrategyExempt ? "Yes" : "No"}</td>
                <td>${p.Description}</td>
                <td><button class="btn btn-outline btn-add-pl" data-id="${p.ID}" style="padding:3px 8px;font-size:11px;">+ Add to Config</button></td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
    </div>
  `;
  sub.querySelector("#cat-search").addEventListener("input", (e) => {
    searchFilter = e.target.value.toLowerCase();
    renderPluginsTable(sub, plugins, showToast);
  });
  sub.querySelectorAll(".btn-add-pl").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = btn.dataset.id;
      appState.config.redteam = appState.config.redteam || { numTestsPerPlugin: 2, languages: ["en"], plugins: [] };
      if (!appState.config.redteam.plugins.some((x) => x.id === id)) {
        appState.config.redteam.plugins.push({ id, numTests: appState.config.redteam.numTestsPerPlugin || 1 });
        notify();
        showToast(`Added ${id} to active config.yaml!`);
      }
    });
  });
}

function renderDatasetsTable(sub, datasets, container, showToast) {
  sub.innerHTML = `
    <div class="card">
      <div class="card-header"><span class="card-title">SQLite Prompt Bank Datasets (${datasets.length} datasets)</span></div>
      <table class="data-table">
        <thead><tr><th>#</th><th>Dataset Name</th><th>Prompts</th><th>Source</th><th>Actions</th></tr></thead>
        <tbody>
          ${datasets.map((d, idx) => `
            <tr>
              <td>${idx + 1}</td>
              <td><code>${d.DatasetName}</code></td>
              <td><strong>${d.PromptCount}</strong></td>
              <td>${d.Source}</td>
              <td style="display:flex;gap:8px;">
                <button class="btn btn-secondary btn-view-ds" data-name="${d.DatasetName}" style="padding:4px 10px;font-size:11.5px;">Browse Prompts</button>
                <button class="btn btn-outline btn-use-ds" data-name="${d.DatasetName}" style="padding:4px 10px;font-size:11.5px;">+ Add to Config</button>
              </td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    </div>
    <div id="ds-preview-box"></div>
  `;

  sub.querySelectorAll(".btn-view-ds").forEach((btn) => {
    btn.addEventListener("click", async () => {
      previewDataset = btn.dataset.name;
      const res = await fetchDatasetPrompts(previewDataset);
      previewRows = res.prompts || [];
      renderDatasetPreviewBox(sub.querySelector("#ds-preview-box"));
    });
  });
  sub.querySelectorAll(".btn-use-ds").forEach((btn) => {
    btn.addEventListener("click", () => {
      const name = btn.dataset.name;
      appState.config.datasets = appState.config.datasets || [];
      if (!appState.config.datasets.includes(name)) {
        appState.config.datasets.push(name);
        notify();
        showToast(`Added dataset '${name}' to config.yaml!`);
      }
    });
  });

  if (previewDataset && previewRows.length > 0) {
    renderDatasetPreviewBox(sub.querySelector("#ds-preview-box"));
  }
}

function renderDatasetPreviewBox(box) {
  box.innerHTML = `
    <div class="card">
      <div class="card-header"><span class="card-title">Prompts in <code>${previewDataset}</code> (Showing ${Math.min(100, previewRows.length)} of ${previewRows.length})</span></div>
      <div style="max-height:340px;overflow-y:auto;">
        <table class="data-table">
          <thead><tr><th>#</th><th>Category</th><th>Attack Type</th><th>Prompt</th></tr></thead>
          <tbody>
            ${previewRows.slice(0, 100).map((r, i) => `
              <tr>
                <td>${i + 1}</td>
                <td><code>${r.Category}</code></td>
                <td>${r.AttackType}</td>
                <td>${r.Prompt}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
    </div>
  `;
}
