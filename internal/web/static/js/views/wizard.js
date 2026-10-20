// Modular Config Builder Wizard View (5 Interactive Steps)
import { appState, notify } from "../state.js";
import { fetchDatasetPrompts, saveCustomDataset, fetchCatalog } from "../api.js";

const STEPS = [
  { num: 1, label: "General & Template" },
  { num: 2, label: "Target, Attacker & Judge LLMs" },
  { num: 3, label: "DB Datasets & Prompt Picker" },
  { num: 4, label: "155 Plugins & 14 Collections" },
  { num: 5, label: "Attack Strategies" },
];

let pluginCategoryFilter = "all";
let pluginSearchQuery = "";
let inspectedDataset = "";
let inspectedPrompts = [];
let selectedPromptIndices = new Set();

export function renderWizardView(container, showToast) {
  const step = appState.wizardStep || 1;
  container.innerHTML = `
    <div class="wizard-stepper">
      ${STEPS.map((s) => `
        <button class="step-tab ${s.num === step ? "active" : ""}" data-step="${s.num}">
          <span class="step-num">Step ${s.num}</span>
          <span class="step-label">${s.label}</span>
        </button>
      `).join("")}
    </div>
    <div id="wizard-step-body"></div>
    <div class="card" style="display:flex;justify-content:space-between;align-items:center;">
      <button id="wiz-prev" class="btn btn-secondary" ${step <= 1 ? "disabled" : ""}>← Previous Step</button>
      <span style="color:var(--text-secondary);font-size:12.5px;">Step ${step} of ${STEPS.length} — Live YAML updates automatically on the right</span>
      <button id="wiz-next" class="btn btn-primary" ${step >= STEPS.length ? "disabled" : ""}>Next Step →</button>
    </div>
  `;

  container.querySelectorAll(".step-tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      appState.wizardStep = Number(btn.dataset.step);
      renderWizardView(container, showToast);
    });
  });
  container.querySelector("#wiz-prev")?.addEventListener("click", () => {
    if (appState.wizardStep > 1) {
      appState.wizardStep--;
      renderWizardView(container, showToast);
    }
  });
  container.querySelector("#wiz-next")?.addEventListener("click", () => {
    if (appState.wizardStep < STEPS.length) {
      appState.wizardStep++;
      renderWizardView(container, showToast);
    }
  });

  const body = container.querySelector("#wizard-step-body");
  if (step === 1) mountStep1General(body);
  if (step === 2) mountStep2Providers(body, container, showToast);
  if (step === 3) mountStep3Datasets(body, container, showToast);
  if (step === 4) mountStep4Plugins(body, container, showToast);
  if (step === 5) mountStep5Strategies(body, container, showToast);
}

function mountStep1General(body) {
  const cfg = appState.config;
  body.innerHTML = `
    <div class="card">
      <div class="card-header"><span class="card-title">1. Assessment Metadata &amp; Template Settings</span></div>
      <div class="form-grid">
        <div class="form-group">
          <label>Description</label>
          <input id="inp-desc" class="input-text" value="${escapeAttr(cfg.description)}" />
        </div>
        <div class="form-group">
          <label>Max Concurrency (Parallel Workers)</label>
          <input id="inp-conc" type="number" min="1" max="32" class="input-text" value="${cfg.maxConcurrency || 2}" />
        </div>
      </div>
      <div class="form-group" style="margin-bottom:12px;">
        <label>Target System Purpose (Used by Attacker &amp; Judge LLMs for Context)</label>
        <input id="inp-purpose" class="input-text" value="${escapeAttr(cfg.purpose)}" />
      </div>
      <div class="form-grid">
        <div class="form-group">
          <label>Prompt Template (SSTI-Safe Boundary)</label>
          <input id="inp-tmpl" class="input-text" value="${escapeAttr(cfg.promptTemplate || "{{prompt}}")}" />
        </div>
        <div class="form-group">
          <label>Inject Variable Name</label>
          <input id="inp-var" class="input-text" value="${escapeAttr(cfg.injectVar || "prompt")}" />
        </div>
        <div class="form-group">
          <label>SQLite Database Path</label>
          <input id="inp-sqlite" class="input-text" value="${escapeAttr(cfg.storage?.sqlitePath || ".gort/gort.db")}" />
        </div>
      </div>
    </div>
  `;
  const bind = (sel, fn) => body.querySelector(sel).addEventListener("input", (e) => { fn(e.target.value); notify(); });
  bind("#inp-desc", (v) => (cfg.description = v));
  bind("#inp-purpose", (v) => (cfg.purpose = v));
  bind("#inp-conc", (v) => (cfg.maxConcurrency = Number(v) || 1));
  bind("#inp-tmpl", (v) => (cfg.promptTemplate = v));
  bind("#inp-var", (v) => (cfg.injectVar = v));
  bind("#inp-sqlite", (v) => { cfg.storage = cfg.storage || {}; cfg.storage.sqlitePath = v; });
}

function mountStep2Providers(body, container, showToast) {
  const cfg = appState.config;
  const models = [
    ...(appState.catalog.localModels || []),
    "ollama:chat:llama3.1:8b",
    "vllm:chat:Qwen/Qwen2.5-7B-Instruct",
    "localai:chat:qwen3-4b",
    "openai:chat:gpt-4o-mini",
    "webhook:http://localhost:9000/api/chat",
  ];
  const modelOptions = models.map((m) => `<option value="${escapeAttr(m)}">${escapeAttr(m)}</option>`).join("");

  body.innerHTML = `
    <div class="card">
      <div class="card-header">
        <span class="card-title">Target LLM(s) (Systems Under Test)</span>
        <button id="btn-add-target" class="btn btn-secondary">+ Add Target LLM</button>
      </div>
      <div id="targets-list"></div>
    </div>
    ${renderRoleCardHTML("attacker", "Attacker LLM (Synthesizer & Multi-Turn Crescendo Driver)", cfg.attacker, modelOptions, false)}
    ${renderRoleCardHTML("judge", "Judge LLM (Safety Rubric Evaluator)", cfg.judge, modelOptions, true)}
  `;

  const targetsList = body.querySelector("#targets-list");
  (cfg.targets || []).forEach((t, idx) => {
    const wrapper = document.createElement("div");
    wrapper.innerHTML = renderRoleCardHTML(`target-${idx}`, `Target #${idx + 1}`, t, modelOptions, false, cfg.targets.length > 1 ? idx : -1);
    targetsList.appendChild(wrapper);
    bindProviderCardEvents(wrapper, t, container, showToast);
  });

  body.querySelector("#btn-add-target").addEventListener("click", () => {
    cfg.targets.push({
      id: "llamacpp:chat:Llama-3.2-3B-Instruct",
      label: "Secondary Target (Llama 3.2 3B)",
      baseUrl: "http://localhost:8080",
      temperature: 0.1,
      maxTokens: 512,
      think: false,
    });
    notify();
    renderWizardView(container, showToast);
  });

  bindProviderCardEvents(body.querySelector(`[data-role-card="attacker"]`), cfg.attacker, container, showToast);
  bindProviderCardEvents(body.querySelector(`[data-role-card="judge"]`), cfg.judge, container, showToast, true);
}

function renderRoleCardHTML(roleKey, title, spec, modelOptions, isJudge, removeIdx = -1) {
  const thinkVal = spec.think === true ? "true" : spec.think === false ? "false" : "omit";
  return `
    <div class="card" data-role-card="${roleKey}" style="background:#111827;">
      <div class="card-header">
        <span class="card-title">${title}</span>
        ${removeIdx >= 0 ? `<button class="btn btn-outline btn-remove-target" data-idx="${removeIdx}">Remove</button>` : ""}
      </div>
      <div class="form-grid">
        <div class="form-group">
          <label>Quick Preset / Discovered Model</label>
          <select class="input-select sel-preset">
            <option value="">-- Choose Model Preset --</option>
            ${modelOptions}
          </select>
        </div>
        <div class="form-group">
          <label>Provider &amp; Model ID</label>
          <input class="input-text inp-id" value="${escapeAttr(spec.id || "")}" />
        </div>
        <div class="form-group">
          <label>Base URL</label>
          <input class="input-text inp-url" value="${escapeAttr(spec.baseUrl || "http://localhost:8080")}" />
        </div>
      </div>
      <div class="form-grid">
        <div class="form-group">
          <label>Temperature</label>
          <input type="number" step="0.1" min="0" max="2" class="input-text inp-temp" value="${spec.temperature ?? 0.1}" />
        </div>
        <div class="form-group">
          <label>Max Tokens</label>
          <input type="number" min="16" max="8192" class="input-text inp-toks" value="${spec.maxTokens || 512}" />
        </div>
        <div class="form-group">
          <label>API Key (Optional or \${ENV_VAR})</label>
          <input class="input-text inp-key" value="${escapeAttr(spec.apiKey || "")}" placeholder="\${OPENAI_API_KEY}" />
        </div>
      </div>
      <div class="form-group" style="margin-bottom:10px;">
        <label>Thinking Directive (<code>think</code> parameter sent by go-rt)</label>
        <div class="radio-group">
          <label class="radio-pill">
            <input type="radio" name="think-${roleKey}" value="false" ${thinkVal === "false" ? "checked" : ""} />
            <span><code>false</code> → Prepends <code>/no_think</code></span>
          </label>
          <label class="radio-pill">
            <input type="radio" name="think-${roleKey}" value="true" ${thinkVal === "true" ? "checked" : ""} />
            <span><code>true</code> → Prepends <code>/think</code></span>
          </label>
          <label class="radio-pill">
            <input type="radio" name="think-${roleKey}" value="omit" ${thinkVal === "omit" ? "checked" : ""} />
            <span>Omitted → Send prompt untouched</span>
          </label>
        </div>
      </div>
      ${isJudge ? `
        <div class="form-group" style="margin-bottom:10px;">
          <label>Judge Evaluation Rubric</label>
          <textarea rows="2" class="input-textarea inp-rubric">${escapeAttr(spec.rubric || "")}</textarea>
        </div>
        <div class="form-group">
          <label>Deterministic Forbidden Substrings (<code>notContains</code>, comma-separated)</label>
          <input class="input-text inp-notcontains" value="${escapeAttr((spec.notContains || []).join(", "))}" />
        </div>
      ` : ""}
    </div>
  `;
}

function bindProviderCardEvents(root, spec, container, showToast, isJudge = false) {
  if (!root) return;
  const presetSel = root.querySelector(".sel-preset");
  const idInp = root.querySelector(".inp-id");
  const urlInp = root.querySelector(".inp-url");

  presetSel.addEventListener("change", () => {
    if (!presetSel.value) return;
    spec.id = presetSel.value;
    idInp.value = presetSel.value;
    if (presetSel.value.startsWith("llamacpp:")) spec.baseUrl = "http://localhost:8080";
    if (presetSel.value.startsWith("ollama:")) spec.baseUrl = "http://localhost:11434";
    if (presetSel.value.startsWith("vllm:")) spec.baseUrl = "http://localhost:8000";
    if (presetSel.value.startsWith("localai:")) spec.baseUrl = "http://localhost:8090";
    if (presetSel.value.startsWith("openai:")) spec.baseUrl = "https://api.openai.com/v1";
    urlInp.value = spec.baseUrl;
    notify();
  });

  idInp.addEventListener("input", (e) => { spec.id = e.target.value; notify(); });
  urlInp.addEventListener("input", (e) => { spec.baseUrl = e.target.value; notify(); });
  root.querySelector(".inp-temp").addEventListener("input", (e) => { spec.temperature = parseFloat(e.target.value) || 0; notify(); });
  root.querySelector(".inp-toks").addEventListener("input", (e) => { spec.maxTokens = parseInt(e.target.value, 10) || 256; notify(); });
  root.querySelector(".inp-key").addEventListener("input", (e) => { spec.apiKey = e.target.value; notify(); });

  root.querySelectorAll(`input[type="radio"]`).forEach((rad) => {
    rad.addEventListener("change", (e) => {
      const v = e.target.value;
      if (v === "true") spec.think = true;
      else if (v === "false") spec.think = false;
      else delete spec.think;
      notify();
    });
  });

  if (isJudge) {
    root.querySelector(".inp-rubric")?.addEventListener("input", (e) => { spec.rubric = e.target.value; notify(); });
    root.querySelector(".inp-notcontains")?.addEventListener("input", (e) => {
      spec.notContains = e.target.value.split(",").map((s) => s.trim()).filter(Boolean);
      notify();
    });
  }

  root.querySelector(".btn-remove-target")?.addEventListener("click", (e) => {
    const idx = Number(e.target.dataset.idx);
    appState.config.targets.splice(idx, 1);
    notify();
    renderWizardView(container, showToast);
  });
}

function mountStep3Datasets(body, container, showToast) {
  const cfg = appState.config;
  cfg.datasets = cfg.datasets || [];
  const datasets = appState.catalog.datasets || [];

  body.innerHTML = `
    <div class="card">
      <div class="card-header">
        <span class="card-title">3. SQLite Prompt Bank Datasets (Select 1 or Multiple, or Pick Individual Prompts)</span>
        <span class="badge badge-info">${cfg.datasets.length} Selected</span>
      </div>
      <div class="SelectGrid">
        ${datasets.map((d) => {
          const checked = cfg.datasets.includes(d.DatasetName);
          return `
            <div class="check-card ${checked ? "selected" : ""}">
              <input type="checkbox" class="chk-dataset" data-name="${escapeAttr(d.DatasetName)}" ${checked ? "checked" : ""} />
              <div class="check-card-body">
                <div class="check-card-title">${escapeAttr(d.DatasetName)}</div>
                <div class="check-card-desc">${d.PromptCount} prompts • ${escapeAttr(d.Source)}</div>
                <button class="btn btn-outline btn-inspect-ds" data-name="${escapeAttr(d.DatasetName)}" style="margin-top:6px;padding:3px 8px;font-size:11px;">Inspect &amp; Pick Prompts</button>
              </div>
            </div>
          `;
        }).join("")}
      </div>
    </div>
    <div id="prompt-picker-panel"></div>
  `;

  body.querySelectorAll(".chk-dataset").forEach((chk) => {
    chk.addEventListener("change", () => {
      const name = chk.dataset.name;
      if (chk.checked && !cfg.datasets.includes(name)) cfg.datasets.push(name);
      if (!chk.checked) cfg.datasets = cfg.datasets.filter((x) => x !== name);
      notify();
      renderWizardView(container, showToast);
    });
  });

  body.querySelectorAll(".btn-inspect-ds").forEach((btn) => {
    btn.addEventListener("click", async () => {
      inspectedDataset = btn.dataset.name;
      const data = await fetchDatasetPrompts(inspectedDataset);
      inspectedPrompts = data.prompts || [];
      selectedPromptIndices = new Set();
      renderPromptPickerModal(body.querySelector("#prompt-picker-panel"), container, showToast);
    });
  });

  if (inspectedDataset && inspectedPrompts.length > 0) {
    renderPromptPickerModal(body.querySelector("#prompt-picker-panel"), container, showToast);
  }
}

function renderPromptPickerModal(panel, container, showToast) {
  panel.innerHTML = `
    <div class="card">
      <div class="card-header">
        <span class="card-title">Prompt Picker: <code>${escapeAttr(inspectedDataset)}</code> (${inspectedPrompts.length} prompts)</span>
        <div style="display:flex;gap:8px;align-items:center;">
          <input id="inp-custom-ds-name" class="input-text" style="width:210px;" value="custom_${escapeAttr(inspectedDataset)}" placeholder="custom_dataset_name" />
          <button id="btn-save-custom-ds" class="btn btn-primary">Save Selected (${selectedPromptIndices.size}) as Dataset</button>
        </div>
      </div>
      <div style="max-height:300px;overflow-y:auto;">
        <table class="data-table">
          <thead><tr><th style="width:40px;"><input type="checkbox" id="chk-all-prompts" /></th><th>Category</th><th>Attack Type</th><th>Prompt</th></tr></thead>
          <tbody>
            ${inspectedPrompts.slice(0, 150).map((r, idx) => `
              <tr>
                <td><input type="checkbox" class="chk-prompt-row" data-idx="${idx}" ${selectedPromptIndices.has(idx) ? "checked" : ""} /></td>
                <td><code>${escapeAttr(r.Category)}</code></td>
                <td>${escapeAttr(r.AttackType)}</td>
                <td>${escapeAttr(r.Prompt)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
    </div>
  `;

  panel.querySelector("#chk-all-prompts")?.addEventListener("change", (e) => {
    const limit = Math.min(150, inspectedPrompts.length);
    for (let i = 0; i < limit; i++) {
      if (e.target.checked) selectedPromptIndices.add(i);
      else selectedPromptIndices.delete(i);
    }
    renderPromptPickerModal(panel, container, showToast);
  });

  panel.querySelectorAll(".chk-prompt-row").forEach((chk) => {
    chk.addEventListener("change", () => {
      const idx = Number(chk.dataset.idx);
      if (chk.checked) selectedPromptIndices.add(idx);
      else selectedPromptIndices.delete(idx);
      panel.querySelector("#btn-save-custom-ds").textContent = `Save Selected (${selectedPromptIndices.size}) as Dataset`;
    });
  });

  panel.querySelector("#btn-save-custom-ds")?.addEventListener("click", async () => {
    const name = panel.querySelector("#inp-custom-ds-name").value.trim();
    const chosen = Array.from(selectedPromptIndices).map((i) => inspectedPrompts[i]);
    if (!name || chosen.length === 0) return;
    const saved = await saveCustomDataset(name, chosen);
    appState.catalog = await fetchCatalog();
    if (!appState.config.datasets.includes(saved.saved)) {
      appState.config.datasets.push(saved.saved);
    }
    notify();
    showToast(`Saved custom dataset '${saved.saved}' (${saved.count} prompts) to SQLite & added to config!`);
    renderWizardView(container, showToast);
  });
}

function mountStep4Plugins(body, container, showToast) {
  const cfg = appState.config;
  cfg.redteam = cfg.redteam || { numTestsPerPlugin: 2, languages: ["en", "tr"], plugins: [] };
  const selectedMap = new Map((cfg.redteam.plugins || []).map((p) => [p.id, p]));
  const collections = Object.keys(appState.catalog.collections || {}).sort();
  const allPlugins = appState.catalog.plugins || [];
  const categories = ["all", ...new Set(allPlugins.map((p) => p.Category))];

  const filteredPlugins = allPlugins.filter((p) => {
    const catMatch = pluginCategoryFilter === "all" || p.Category === pluginCategoryFilter;
    const qMatch = !pluginSearchQuery || p.ID.toLowerCase().includes(pluginSearchQuery) || p.Description.toLowerCase().includes(pluginSearchQuery);
    return catMatch && qMatch;
  });

  body.innerHTML = `
    <div class="card">
      <div class="card-header"><span class="card-title">4A. Synthesis Settings &amp; 14 Preset Collections</span></div>
      <div class="form-grid">
        <div class="form-group">
          <label>Default <code>numTestsPerPlugin</code></label>
          <input id="inp-numtests" type="number" min="1" max="50" class="input-text" value="${cfg.redteam.numTestsPerPlugin || 2}" />
        </div>
        <div class="form-group">
          <label>Synthesis Languages (comma-separated, e.g. en, tr, de)</label>
          <input id="inp-langs" class="input-text" value="${escapeAttr((cfg.redteam.languages || ["en"]).join(", "))}" />
        </div>
      </div>
      <div class="SelectGrid" style="max-height:200px;">
        ${collections.map((col) => {
          const checked = selectedMap.has(col);
          const count = (appState.catalog.collections[col] || []).length;
          return `
            <label class="check-card ${checked ? "selected" : ""}">
              <input type="checkbox" class="chk-plugin-id" data-id="${escapeAttr(col)}" ${checked ? "checked" : ""} />
              <div class="check-card-body">
                <div class="check-card-title">${escapeAttr(col)}</div>
                <div class="check-card-desc">Preset bundle (${count} plugins)</div>
              </div>
            </label>
          `;
        }).join("")}
      </div>
    </div>

    <div class="card">
      <div class="card-header">
        <span class="card-title">4B. Individual Redteam Plugins (${filteredPlugins.length} of ${allPlugins.length})</span>
        <div style="display:flex;gap:8px;">
          <select id="sel-plugin-cat" class="input-select" style="width:160px;">
            ${categories.map((c) => `<option value="${escapeAttr(c)}" ${c === pluginCategoryFilter ? "selected" : ""}>Category: ${escapeAttr(c)}</option>`).join("")}
          </select>
          <input id="inp-plugin-search" class="input-text" style="width:220px;" placeholder="Filter plugins..." value="${escapeAttr(pluginSearchQuery)}" />
        </div>
      </div>
      <div style="max-height:340px;overflow-y:auto;">
        <table class="data-table">
          <thead><tr><th>Use</th><th>Plugin ID</th><th>Category</th><th>Description</th><th>numTests</th><th>Severity</th></tr></thead>
          <tbody>
            ${filteredPlugins.map((p) => {
              const spec = selectedMap.get(p.ID);
              const sev = spec?.severity || p.Severity;
              return `
                <tr>
                  <td><input type="checkbox" class="chk-plugin-id" data-id="${escapeAttr(p.ID)}" ${spec ? "checked" : ""} /></td>
                  <td><code>${escapeAttr(p.ID)}</code></td>
                  <td><span class="badge badge-info">${escapeAttr(p.Category)}</span></td>
                  <td>${escapeAttr(p.Description)}</td>
                  <td style="width:90px;">
                    <input type="number" min="1" max="50" class="input-text inp-pl-num" data-id="${escapeAttr(p.ID)}" value="${spec?.numTests || cfg.redteam.numTestsPerPlugin || 1}" ${spec ? "" : "disabled"} />
                  </td>
                  <td style="width:120px;">
                    <select class="input-select sel-pl-sev" data-id="${escapeAttr(p.ID)}" ${spec ? "" : "disabled"}>
                      ${["low", "medium", "high", "critical"].map((s) => `<option value="${s}" ${s === sev ? "selected" : ""}>${s}</option>`).join("")}
                    </select>
                  </td>
                </tr>
              `;
            }).join("")}
          </tbody>
        </table>
      </div>
    </div>
  `;

  body.querySelector("#inp-numtests").addEventListener("input", (e) => {
    cfg.redteam.numTestsPerPlugin = Number(e.target.value) || 1;
    notify();
  });
  body.querySelector("#inp-langs").addEventListener("input", (e) => {
    cfg.redteam.languages = e.target.value.split(",").map((s) => s.trim()).filter(Boolean);
    notify();
  });
  body.querySelector("#sel-plugin-cat").addEventListener("change", (e) => {
    pluginCategoryFilter = e.target.value;
    mountStep4Plugins(body, container, showToast);
  });
  body.querySelector("#inp-plugin-search").addEventListener("input", (e) => {
    pluginSearchQuery = e.target.value.toLowerCase();
    mountStep4Plugins(body, container, showToast);
  });

  body.querySelectorAll(".chk-plugin-id").forEach((chk) => {
    chk.addEventListener("change", () => {
      const id = chk.dataset.id;
      if (chk.checked && !selectedMap.has(id)) {
        cfg.redteam.plugins.push({ id, numTests: cfg.redteam.numTestsPerPlugin || 1 });
      } else if (!chk.checked) {
        cfg.redteam.plugins = cfg.redteam.plugins.filter((x) => x.id !== id);
      }
      notify();
      mountStep4Plugins(body, container, showToast);
    });
  });

  body.querySelectorAll(".inp-pl-num").forEach((inp) => {
    inp.addEventListener("input", () => {
      const item = cfg.redteam.plugins.find((x) => x.id === inp.dataset.id);
      if (item) { item.numTests = Number(inp.value) || 1; notify(); }
    });
  });
  body.querySelectorAll(".sel-pl-sev").forEach((sel) => {
    sel.addEventListener("change", () => {
      const item = cfg.redteam.plugins.find((x) => x.id === sel.dataset.id);
      if (item) { item.severity = sel.value; notify(); }
    });
  });
}

function mountStep5Strategies(body, container, showToast) {
  const cfg = appState.config;
  cfg.strategies = cfg.strategies || [];
  const selectedMap = new Map(cfg.strategies.map((s) => [s.id, s]));
  const catalogStrats = appState.catalog.strategies || [];

  body.innerHTML = `
    <div class="card">
      <div class="card-header"><span class="card-title">5. Attack Mutation &amp; Multi-Turn Strategies (<code>gort:strategy:*</code>)</span></div>
      <div style="display:flex;flex-direction:column;gap:10px;">
        ${catalogStrats.map((st) => {
          const active = selectedMap.get(st.id);
          const isMulti = st.kind === "multiturn";
          const isLayer = st.kind === "layer";
          return `
            <div class="check-card ${active ? "selected" : ""}" style="align-items:center;">
              <input type="checkbox" class="chk-strat" data-id="${escapeAttr(st.id)}" data-kind="${st.kind}" ${active ? "checked" : ""} />
              <div class="check-card-body">
                <div class="check-card-title">${escapeAttr(st.label)} (<code>${escapeAttr(st.id)}</code>)</div>
                <div class="check-card-desc">${escapeAttr(st.description)}</div>
              </div>
              ${isMulti && active ? `
                <div style="display:flex;gap:8px;align-items:center;">
                  <label class="inline-label">maxTurns:</label>
                  <input type="number" min="1" max="12" class="input-text inp-st-turns" data-id="${escapeAttr(st.id)}" style="width:70px;" value="${active.maxTurns || 3}" />
                  <label class="inline-label">maxBacktracks:</label>
                  <input type="number" min="0" max="8" class="input-text inp-st-back" data-id="${escapeAttr(st.id)}" style="width:70px;" value="${active.maxBacktracks ?? 2}" />
                </div>
              ` : ""}
              ${isLayer && active ? `
                <div style="display:flex;gap:8px;align-items:center;">
                  <label class="inline-label">Steps (comma):</label>
                  <input class="input-text inp-st-steps" data-id="${escapeAttr(st.id)}" style="width:310px;" value="${escapeAttr((active.steps || ["gort:strategy:jailbreak-templates", "gort:strategy:base64"]).join(", "))}" />
                </div>
              ` : ""}
            </div>
          `;
        }).join("")}
      </div>
    </div>
  `;

  body.querySelectorAll(".chk-strat").forEach((chk) => {
    chk.addEventListener("change", () => {
      const id = chk.dataset.id;
      const kind = chk.dataset.kind;
      if (chk.checked && !selectedMap.has(id)) {
        const entry = { id };
        if (kind === "multiturn") { entry.maxTurns = 3; entry.maxBacktracks = 2; }
        if (kind === "layer") { entry.steps = ["gort:strategy:jailbreak-templates", "gort:strategy:base64"]; }
        cfg.strategies.push(entry);
      } else if (!chk.checked) {
        cfg.strategies = cfg.strategies.filter((x) => x.id !== id);
      }
      notify();
      mountStep5Strategies(body, container, showToast);
    });
  });

  body.querySelectorAll(".inp-st-turns").forEach((inp) => {
    inp.addEventListener("input", () => {
      const s = cfg.strategies.find((x) => x.id === inp.dataset.id);
      if (s) { s.maxTurns = Number(inp.value) || 2; notify(); }
    });
  });
  body.querySelectorAll(".inp-st-back").forEach((inp) => {
    inp.addEventListener("input", () => {
      const s = cfg.strategies.find((x) => x.id === inp.dataset.id);
      if (s) { s.maxBacktracks = Number(inp.value) || 0; notify(); }
    });
  });
  body.querySelectorAll(".inp-st-steps").forEach((inp) => {
    inp.addEventListener("input", () => {
      const s = cfg.strategies.find((x) => x.id === inp.dataset.id);
      if (s) { s.steps = inp.value.split(",").map((v) => v.trim()).filter(Boolean); notify(); }
    });
  });
}

function escapeAttr(str) {
  return String(str ?? "")
    .replace(/&/g, "&amp;")
    .replace(/"/g, "&quot;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}
