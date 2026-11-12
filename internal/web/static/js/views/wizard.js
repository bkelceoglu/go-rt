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
    <div class="card" style="display:flex; justify-content:space-between; align-items:center; background:var(--bg-elevated); margin-bottom:12px; padding:10px 14px;">
      <div style="display:flex; align-items:center; gap:8px;">
        <span style="font-size:16px;">📂</span>
        <div>
          <strong style="font-size:13px;">Load Existing Configuration (.yaml / .yml)</strong>
          <div style="font-size:11.5px; color:var(--text-secondary);">Browse any YAML file from your computer or select a saved workspace file</div>
        </div>
      </div>
      <button type="button" id="btn-wiz-browse-file" class="btn btn-secondary" style="font-size:12px; padding:6px 12px;">📂 Browse Local YAML File...</button>
    </div>
    <div class="card">
      <div class="card-header"><span class="card-title">1. Assessment Metadata &amp; Template Settings</span></div>
      <div class="form-grid">
        <div class="form-group">
          <label>Description</label>
          <input id="inp-desc" class="input-text" value="${escapeAttr(cfg.description)}" placeholder="e.g. Enterprise Assistant Red Teaming &amp; Security Audit" />
        </div>
        <div class="form-group">
          <label>Max Concurrency (Parallel Workers)</label>
          <input id="inp-conc" type="number" min="1" max="32" class="input-text" value="${cfg.maxConcurrency || 2}" />
        </div>
      </div>
      <div class="form-group" style="margin-bottom:12px;">
        <label>Target System Purpose (Used by Attacker &amp; Judge LLMs for Context)</label>
        <input id="inp-purpose" class="input-text" value="${escapeAttr(cfg.purpose)}" placeholder="e.g. Internal enterprise assistant for customer policy and documentation QA" />
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
          <input id="inp-sqlite" class="input-text" value="${escapeAttr(cfg.storage?.sqlitePath || ".gort/gort.db")}" readOnly />
        </div>
      </div>
    </div>
  `;
  const bind = (sel, fn) => body.querySelector(sel).addEventListener("input", (e) => { fn(e.target.value); notify(); });
  body.querySelector("#btn-wiz-browse-file")?.addEventListener("click", () => {
    document.getElementById("config-file-input")?.click();
  });
  bind("#inp-desc", (v) => (cfg.description = v));
  bind("#inp-purpose", (v) => (cfg.purpose = v));
  bind("#inp-conc", (v) => (cfg.maxConcurrency = Number(v) || 1));
  bind("#inp-tmpl", (v) => (cfg.promptTemplate = v));
  bind("#inp-var", (v) => (cfg.injectVar = v));
  bind("#inp-sqlite", (v) => { cfg.storage = cfg.storage || {}; cfg.storage.sqlitePath = v; });
}

function mountStep2Providers(body, container, showToast) {
  const cfg = appState.config;
  const providers = appState.catalog.providers || [
    { type: "ollama", label: "Ollama (Local Daemon)", prefix: "ollama:chat:", defaultUrl: "http://localhost:11434" },
    { type: "llamacpp", label: "llama.cpp Server", prefix: "llamacpp:chat:", defaultUrl: "http://localhost:8080" },
    { type: "vllm", label: "vLLM / LiteLLM", prefix: "vllm:chat:", defaultUrl: "http://localhost:8000/v1" },
    { type: "localai", label: "LocalAI", prefix: "localai:chat:", defaultUrl: "http://localhost:8090/v1" },
    { type: "openai", label: "OpenAI / Cloud", prefix: "openai:chat:", defaultUrl: "https://api.openai.com/v1" },
    { type: "webhook", label: "Custom Webhook API", prefix: "webhook:", defaultUrl: "http://localhost:9000/api/chat" },
  ];
  const liveModels = appState.catalog.localModels || [];

  cfg.targets = cfg.targets || [];
  cfg.attacker = cfg.attacker || { id: "", baseUrl: "", think: false };
  cfg.judge = cfg.judge || { id: "", baseUrl: "", think: false, notContains: [], rubric: "" };

  body.innerHTML = `
    <div class="card">
      <div class="card-header">
        <span class="card-title">Target LLM(s) (Systems Under Test)</span>
        <button id="btn-add-target" class="btn btn-secondary">+ Add Target LLM</button>
      </div>
      <div id="targets-list"></div>
    </div>
    ${renderRoleCardHTML("attacker", "Attacker LLM (Synthesizer & Multi-Turn Crescendo Driver)", cfg.attacker, providers, liveModels, false)}
    ${renderRoleCardHTML("judge", "Judge LLM (Safety Rubric Evaluator)", cfg.judge, providers, liveModels, true)}
  `;

  const targetsList = body.querySelector("#targets-list");
  if (cfg.targets.length === 0) {
    const emptyNotice = document.createElement("div");
    emptyNotice.style.cssText = "color:var(--text-secondary);font-size:13px;padding:16px;text-align:center;border:1px dashed var(--border-color);border-radius:6px;margin-bottom:12px;";
    emptyNotice.innerHTML = `No Target LLMs added yet. Click <strong>+ Add Target LLM</strong> above to configure your system under test.`;
    targetsList.appendChild(emptyNotice);
  } else {
    cfg.targets.forEach((t, idx) => {
      const wrapper = document.createElement("div");
      wrapper.innerHTML = renderRoleCardHTML(`target-${idx}`, `Target #${idx + 1}`, t, providers, liveModels, false, idx);
      targetsList.appendChild(wrapper);
      bindProviderCardEvents(wrapper, t, container, showToast);
    });
  }

  body.querySelector("#btn-add-target").addEventListener("click", () => {
    cfg.targets.push({
      id: "ollama:chat:model",
      label: `Target LLM #${cfg.targets.length + 1}`,
      baseUrl: "http://localhost:11434",
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

function renderRoleCardHTML(roleKey, title, spec, providers, liveModels, isJudge, removeIdx = -1) {
  const thinkVal = spec.think === true ? "true" : spec.think === false ? "false" : "omit";
  const providerOptions = providers.map((p) => `
    <option value="${escapeAttr(p.prefix)}" data-default-url="${escapeAttr(p.defaultUrl)}">
      ${escapeAttr(p.label)} (${escapeAttr(p.defaultUrl)})
    </option>
  `).join("");
  const liveOptions = (liveModels || []).map((m) => `<option value="${escapeAttr(m)}">${escapeAttr(m)}</option>`).join("");

  return `
    <div class="card" data-role-card="${roleKey}" style="background:var(--bg-elevated);">
      <div class="card-header">
        <span class="card-title">${title}</span>
        <div style="display:flex; gap:8px; align-items:center;">
          <span class="health-badge">○ Untested</span>
          <button type="button" class="btn btn-outline btn-test-conn" style="padding:2px 8px; font-size:11px;">⚡ Test Ping</button>
          ${removeIdx >= 0 ? `<button class="btn btn-outline btn-remove-target" data-idx="${removeIdx}">Remove</button>` : ""}
        </div>
      </div>
      <div class="form-grid">
        <div class="form-group">
          <label>Protocol / Runtime Type</label>
          <select class="input-select sel-protocol">
            <option value="">-- Select Protocol Template --</option>
            ${providerOptions}
          </select>
        </div>
        <div class="form-group">
          <label>Provider &amp; Model ID (Editable)</label>
          <input class="input-text inp-id" list="list-models-${roleKey}" value="${escapeAttr(spec.id || "")}" placeholder="e.g. ollama:chat:llama3.1:8b" />
          <datalist id="list-models-${roleKey}">
            ${liveOptions}
          </datalist>
        </div>
        <div class="form-group">
          <label>Base URL (Default filled, fully editable)</label>
          <input class="input-text inp-url" value="${escapeAttr(spec.baseUrl || "")}" placeholder="e.g. http://localhost:11434" />
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
  const protoSel = root.querySelector(".sel-protocol");
  const idInp = root.querySelector(".inp-id");
  const urlInp = root.querySelector(".inp-url");
  const testBtn = root.querySelector(".btn-test-conn");
  const healthBadge = root.querySelector(".health-badge");

  if (protoSel) {
    protoSel.addEventListener("change", () => {
      const selected = protoSel.options[protoSel.selectedIndex];
      if (!selected || !selected.value) return;
      const prefix = selected.value;
      const defaultUrl = selected.dataset.defaultUrl || "";
      if (!idInp.value || idInp.value.includes(":")) {
        idInp.value = prefix + "model";
        spec.id = idInp.value;
      }
      urlInp.value = defaultUrl;
      spec.baseUrl = defaultUrl;
      notify();
    });
  }

  if (testBtn) {
    testBtn.addEventListener("click", async () => {
      healthBadge.className = "health-badge status-testing";
      healthBadge.textContent = "Testing...";
      try {
        const rawYaml = `targets:\n  - id: ${JSON.stringify(spec.id || "test")}\n    baseUrl: ${JSON.stringify(spec.baseUrl || "")}\n    apiKey: ${JSON.stringify(spec.apiKey || "")}\n`;
        const res = await (await fetch("/api/configs/ping-providers", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ rawYaml }),
        })).json();
        const first = (res.reachability && res.reachability[0]) || {};
        if (first.reachable) {
          healthBadge.className = "health-badge status-accepted";
          healthBadge.textContent = `● ${first.message}`;
          showToast(`Endpoint accepted: ${spec.id || spec.baseUrl}`);
        } else {
          healthBadge.className = "health-badge status-refused";
          healthBadge.textContent = `○ ${first.message || "Connection refused"}`;
          showToast(`Endpoint connection refused: ${first.message}`);
        }
      } catch (err) {
        healthBadge.className = "health-badge status-refused";
        healthBadge.textContent = "○ Connection refused (Check error)";
      }
    });
  }

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
          const dsName = d.datasetName || d.DatasetName || "";
          const count = d.promptCount ?? d.PromptCount ?? 0;
          const src = d.source || d.Source || "";
          const checked = cfg.datasets.includes(dsName);
          return `
            <div class="check-card ${checked ? "selected" : ""}">
              <input type="checkbox" class="chk-dataset" data-name="${escapeAttr(dsName)}" ${checked ? "checked" : ""} />
              <div class="check-card-body">
                <div class="check-card-title">${escapeAttr(dsName)}</div>
                <div class="check-card-desc">${count} prompts • ${escapeAttr(src)}</div>
                <button class="btn btn-outline btn-inspect-ds" data-name="${escapeAttr(dsName)}" style="margin-top:6px;padding:3px 8px;font-size:11px;">Inspect &amp; Pick Prompts</button>
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
                <td><code>${escapeAttr(r.category || r.Category || "")}</code></td>
                <td>${escapeAttr(r.attack_type || r.attackType || r.AttackType || "")}</td>
                <td>${escapeAttr(r.prompt || r.Prompt || "")}</td>
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

  // Expand any legacy collection IDs directly in redteam.plugins into member plugins
  const legacyCols = (cfg.redteam.plugins || []).filter((p) => p.id && p.id.startsWith("gort:collection:"));
  if (legacyCols.length > 0) {
    const toAdd = [];
    legacyCols.forEach((colEntry) => {
      const members = appState.catalog.collections[colEntry.id] || [];
      members.forEach((mId) => {
        if (!cfg.redteam.plugins.some((p) => p.id === mId)) {
          toAdd.push({ id: mId, numTests: colEntry.numTests || cfg.redteam.numTestsPerPlugin || 1 });
        }
      });
    });
    cfg.redteam.plugins = cfg.redteam.plugins.filter((p) => !p.id.startsWith("gort:collection:")).concat(toAdd);
  }

  const selectedMap = new Map((cfg.redteam.plugins || []).map((p) => [p.id, p]));
  const collections = Object.keys(appState.catalog.collections || {}).sort();
  const allPlugins = appState.catalog.plugins || [];
  const categories = ["all", ...new Set(allPlugins.map((p) => p.category || p.Category).filter(Boolean))];

  const filteredPlugins = allPlugins.filter((p) => {
    const pCat = p.category || p.Category || "";
    const pId = p.id || p.ID || "";
    const pDesc = p.description || p.Description || "";
    const catMatch = pluginCategoryFilter === "all" || pCat === pluginCategoryFilter;
    const qMatch = !pluginSearchQuery || pId.toLowerCase().includes(pluginSearchQuery) || pDesc.toLowerCase().includes(pluginSearchQuery);
    return catMatch && qMatch;
  });

  const totalActivePlugins = selectedMap.size;

  body.innerHTML = `
    <div class="card">
      <div class="card-header"><span class="card-title">4A. Synthesis Settings &amp; 14 Preset Collections (Selecting a bundle activates all its plugins in 4B)</span></div>
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
      <div class="SelectGrid" style="max-height:220px;">
        ${collections.map((col) => {
          const memberIds = appState.catalog.collections[col] || [];
          const count = memberIds.length;
          const selectedMembersCount = memberIds.filter((pid) => selectedMap.has(pid)).length;
          const allSelected = count > 0 && selectedMembersCount === count;
          const partial = selectedMembersCount > 0 && !allSelected;
          const cardClass = allSelected ? "selected" : (partial ? "partial" : "");
          return `
            <label class="check-card ${cardClass}" style="cursor:pointer;">
              <input type="checkbox" class="chk-collection-col" data-col="${escapeAttr(col)}" data-partial="${partial ? "true" : "false"}" ${allSelected ? "checked" : ""} />
              <div class="check-card-body">
                <div class="check-card-title">${escapeAttr(col)}</div>
                <div class="check-card-desc">
                  ${count === 0 ? "Empty bundle" : (allSelected ? `All ${count} plugins active` : (partial ? `${selectedMembersCount} / ${count} plugins active` : `Preset bundle (${count} plugins)`))}
                </div>
              </div>
            </label>
          `;
        }).join("")}
      </div>
    </div>

    <div class="card">
      <div class="card-header">
        <span class="card-title">4B. Individual Redteam Plugins (${totalActivePlugins} of ${allPlugins.length} selected)</span>
        <div style="display:flex;gap:8px;align-items:center;">
          <button id="btn-select-filtered" class="btn btn-outline" style="padding:4px 8px;font-size:11px;" title="Select all currently filtered plugins">+ Select Filtered (${filteredPlugins.length})</button>
          <button id="btn-deselect-all-pl" class="btn btn-outline" style="padding:4px 8px;font-size:11px;" title="Clear all plugin selections">Clear All</button>
          <select id="sel-plugin-cat" class="input-select" style="width:160px;">
            ${categories.map((c) => `<option value="${escapeAttr(c)}" ${c === pluginCategoryFilter ? "selected" : ""}>Category: ${escapeAttr(c)}</option>`).join("")}
          </select>
          <input id="inp-plugin-search" class="input-text" style="width:200px;" placeholder="Filter plugins..." value="${escapeAttr(pluginSearchQuery)}" />
        </div>
      </div>
      <div style="max-height:340px;overflow-y:auto;">
        <table class="data-table">
          <thead><tr><th>Use</th><th>Plugin ID</th><th>Category</th><th>Description</th><th>numTests</th><th>Severity</th></tr></thead>
          <tbody>
            ${filteredPlugins.map((p) => {
              const pId = p.id || p.ID || "";
              const pCat = p.category || p.Category || "";
              const pDesc = p.description || p.Description || "";
              const pSev = p.severity || p.Severity || "low";
              const spec = selectedMap.get(pId);
              const sev = spec?.severity || pSev;
              return `
                <tr>
                  <td><input type="checkbox" class="chk-plugin-id" data-id="${escapeAttr(pId)}" ${spec ? "checked" : ""} /></td>
                  <td><code>${escapeAttr(pId)}</code></td>
                  <td><span class="badge badge-info">${escapeAttr(pCat)}</span></td>
                  <td>${escapeAttr(pDesc)}</td>
                  <td style="width:90px;">
                    <input type="number" min="1" max="50" class="input-text inp-pl-num" data-id="${escapeAttr(pId)}" value="${spec?.numTests || cfg.redteam.numTestsPerPlugin || 1}" ${spec ? "" : "disabled"} />
                  </td>
                  <td style="width:120px;">
                    <select class="input-select sel-pl-sev" data-id="${escapeAttr(pId)}" ${spec ? "" : "disabled"}>
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

  body.querySelectorAll(".chk-collection-col").forEach((chk) => {
    if (chk.dataset.partial === "true") {
      chk.indeterminate = true;
    }
    chk.addEventListener("change", () => {
      const col = chk.dataset.col;
      const memberIds = appState.catalog.collections[col] || [];
      const currentSelected = new Set((cfg.redteam.plugins || []).map((p) => p.id));
      const allSelected = memberIds.length > 0 && memberIds.every((pid) => currentSelected.has(pid));

      if (chk.checked || !allSelected) {
        memberIds.forEach((pid) => {
          if (!currentSelected.has(pid)) {
            cfg.redteam.plugins.push({ id: pid, numTests: cfg.redteam.numTestsPerPlugin || 1 });
            currentSelected.add(pid);
          }
        });
        showToast(`Activated all ${memberIds.length} plugins in bundle '${col}'`);
      } else {
        const removeSet = new Set(memberIds);
        cfg.redteam.plugins = cfg.redteam.plugins.filter((p) => !removeSet.has(p.id));
        showToast(`Deactivated all plugins in bundle '${col}'`);
      }
      notify();
      mountStep4Plugins(body, container, showToast);
    });
  });

  body.querySelector("#btn-select-filtered")?.addEventListener("click", () => {
    const currentSet = new Set((cfg.redteam.plugins || []).map((p) => p.id));
    filteredPlugins.forEach((p) => {
      const pid = p.id || p.ID;
      if (!currentSet.has(pid)) {
        cfg.redteam.plugins.push({ id: pid, numTests: cfg.redteam.numTestsPerPlugin || 1 });
        currentSet.add(pid);
      }
    });
    notify();
    mountStep4Plugins(body, container, showToast);
    showToast(`Added ${filteredPlugins.length} filtered plugins to assessment!`);
  });

  body.querySelector("#btn-deselect-all-pl")?.addEventListener("click", () => {
    cfg.redteam.plugins = [];
    notify();
    mountStep4Plugins(body, container, showToast);
    showToast("Cleared all selected plugins");
  });

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
