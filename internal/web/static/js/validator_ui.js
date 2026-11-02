// Interactive YAML Validation, Attacker Logic Control & Auto-Fix Modal (CLI + Web Parity)
import { validateConfig, autoFixConfig, saveConfigFile } from "./api.js";
import { appState, notify, escapeAttr } from "./state.js";

let pendingAutoFixResult = null;

export async function openValidationModal(showToast, onValidatedCallback = null) {
  const modal = document.getElementById("validation-modal");
  const body = document.getElementById("validation-modal-body");
  if (!modal || !body) return false;

  modal.classList.remove("hidden");
  body.innerHTML = `<div class="stat-pill">Validating YAML syntax, SQLite catalog IDs, and Attacker LLM logic...</div>`;
  pendingAutoFixResult = null;

  const rawYaml = document.getElementById("yaml-preview-editor").value;
  try {
    const res = await validateConfig(rawYaml, true);
    renderValidationModalContent(body, res.report, res.config, showToast, onValidatedCallback);
    return res.report.valid && (res.report.issues || []).length === 0;
  } catch (err) {
    body.innerHTML = `<div class="badge badge-danger">Validation Error: ${escapeAttr(err.message)}</div>`;
    return false;
  }
}

export function bindValidationModalEvents(showToast, refreshMainView) {
  const closeBtn = document.getElementById("btn-close-validation-modal");
  const modal = document.getElementById("validation-modal");
  if (closeBtn && modal) {
    closeBtn.addEventListener("click", () => modal.classList.add("hidden"));
  }
  const validateBtn = document.getElementById("btn-validate-yaml");
  if (validateBtn) {
    validateBtn.addEventListener("click", () => {
      openValidationModal(showToast, () => {
        if (refreshMainView) refreshMainView();
      });
    });
  }
}

function renderValidationModalContent(container, report, parsedCfg, showToast, onValidatedCallback) {
  const issues = report.issues || [];
  const reachability = report.reachability || [];
  const isClean = report.valid && issues.length === 0 && !report.needsAttackerResolution;

  const statusHtml = isClean
    ? `<div style="display:flex; justify-content:space-between; align-items:center; padding:12px; border:1px solid var(--accent-emerald); background:rgba(16,185,129,0.08);">
         <div><strong style="color:var(--accent-emerald);">✓ CONFIGURATION VALID</strong> — All YAML syntax, SQLite catalog IDs, and Attacker LLM logic checks passed.</div>
         <button id="btn-reval-now" class="btn btn-secondary">↻ Re-Validate</button>
       </div>`
    : `<div style="display:flex; justify-content:space-between; align-items:center; padding:12px; border:1px solid var(--accent-amber); background:rgba(245,158,11,0.08);">
         <div><strong style="color:var(--accent-amber);">⚠ ${issues.length} VALIDATION / LOGIC ISSUE(S) FOUND</strong> — Review findings below or use Auto-Fix.</div>
         <button id="btn-reval-now" class="btn btn-secondary">↻ Re-Validate</button>
       </div>`;

  const reachabilityHtml = reachability.length > 0
    ? `<div class="card" style="padding:12px; border:1px solid var(--border-subtle); background:var(--bg-elevated);">
         <div style="font-weight:700; margin-bottom:6px; display:flex; justify-content:space-between;">
           <span>Pre-flight LLM Endpoint Reachability Checks</span>
           <span style="font-size:11px; color:var(--text-secondary);">Verified live network socket connection</span>
         </div>
         <div style="display:flex; flex-direction:column; gap:6px;">
           ${reachability.map((r) => `
             <div style="display:flex; justify-content:space-between; align-items:center; font-size:12px; padding:6px 10px; border-radius:4px; background:var(--bg-surface); border-left:3px solid ${r.reachable ? "var(--success)" : "var(--danger)"};">
               <div>
                 <span class="health-badge ${r.reachable ? "status-accepted" : "status-refused"}" style="margin-right:6px;">
                   ${r.reachable ? "● Accepted" : "○ Connection Refused"}
                 </span>
                 <span style="font-weight:600;">[${escapeAttr(r.role)}]</span>
                 <code style="margin-left:6px;">${escapeAttr(r.id)}</code>
               </div>
               <div style="color:var(--text-secondary); text-align:right;">
                 <span>${escapeAttr(r.message)}</span>
                 ${r.baseUrl ? `<span style="margin-left:6px; font-family:monospace; font-size:11px;">(${escapeAttr(r.baseUrl)})</span>` : ""}
               </div>
             </div>
           `).join("")}
         </div>
       </div>`
    : "";

  const issuesHtml = issues.length > 0
    ? `<div style="display:flex; flex-direction:column; gap:8px;">
         ${issues.map((iss, idx) => `
           <div style="padding:10px 12px; border:1px solid var(--border-subtle); background:var(--bg-elevated);">
             <div style="display:flex; gap:8px; align-items:center; margin-bottom:4px;">
               <span class="badge ${iss.severity === "error" ? "badge-danger" : "badge-warn"}">${escapeAttr(iss.severity.toUpperCase())}</span>
               <code>${escapeAttr(iss.code)}</code>
               <strong style="font-size:12px;">${escapeAttr(iss.field)}</strong>
             </div>
             <div style="font-size:13px;">${escapeAttr(iss.message)}</div>
             ${iss.suggestion ? `<div style="font-size:12px; color:var(--text-secondary); margin-top:4px;">Suggestion: ${escapeAttr(iss.suggestion)}</div>` : ""}
           </div>
         `).join("")}
       </div>`
    : "";

  const logicHtml = report.needsAttackerResolution
    ? renderAttackerLogicPanel(report)
    : "";

  const autoFixHtml = !isClean
    ? `<div class="card" style="padding:12px; border:1px solid var(--border-focus);">
         <div style="font-weight:700; margin-bottom:6px;">2. Automated Repair Engine (Requires Your Confirmation)</div>
         <p style="font-size:12px; color:var(--text-secondary); margin:0 0 10px 0;">
           Choose between the deterministic <strong>Mini Engine</strong> (&lt;1ms, offline, SQLite catalog-aware) or <strong>Judge LLM Repair</strong> (AI syntax repair + Mini Engine verification).
         </p>
         <div style="display:flex; gap:10px; flex-wrap:wrap;">
           <button id="btn-autofix-mini" class="btn btn-primary">⚡ Preview Fix via 2.1 Mini Engine (Recommended)</button>
           <button id="btn-autofix-judge" class="btn btn-outline">🤖 Preview Fix via 2.2 Judge LLM</button>
         </div>
         <div id="autofix-preview-zone" style="margin-top:12px;"></div>
       </div>`
    : "";

  container.innerHTML = statusHtml + reachabilityHtml + issuesHtml + logicHtml + autoFixHtml;
  wireModalActionButtons(container, showToast, onValidatedCallback);
}

function renderAttackerLogicPanel(report) {
  const strats = (report.attackerDependentStrategies || []).join(", ");
  const judgeLabel = report.judgeId || "configured Judge LLM";
  return `
    <div class="card" style="padding:12px; border:1px solid var(--accent-amber);">
      <div style="font-weight:700; margin-bottom:6px;">1. Logic Control — Missing Attacker LLM Dependency</div>
      <p style="font-size:12px; color:var(--text-secondary); margin:0 0 10px 0;">
        Active strategies (<code>${escapeAttr(strats)}</code>) require an Attacker LLM, which is currently empty. Do you want to configure an Attacker LLM?
      </p>
      <div style="display:flex; flex-direction:column; gap:8px; font-size:13px;">
        <label style="display:flex; gap:8px; align-items:flex-start; cursor:pointer;">
          <input type="radio" name="attacker-res-mode" value="use_judge_sequential" checked />
          <span><strong>1.2.2 No Attacker LLM — Reuse Judge LLM (${escapeAttr(judgeLabel)}) Sequentially:</strong> Attack generation and Judge grading will run strictly one after another (<code>sequentialSharedLlm: true</code>, <code>maxConcurrency: 1</code>).</span>
        </label>
        <label style="display:flex; gap:8px; align-items:flex-start; cursor:pointer;">
          <input type="radio" name="attacker-res-mode" value="remove_strategies" />
          <span><strong>1.2.1 No Attacker LLM — Remove Attacker-Dependent Strategies:</strong> Strip <code>${escapeAttr(strats)}</code> from the configuration.</span>
        </label>
        <label style="display:flex; gap:8px; align-items:flex-start; cursor:pointer;">
          <input type="radio" name="attacker-res-mode" value="set_attacker" />
          <span><strong>1.1 Yes — Configure a Dedicated Attacker LLM:</strong> Run Attacker and Judge independently.</span>
        </label>
        <div id="custom-attacker-inputs" style="display:grid; grid-template-columns:1fr 1fr; gap:8px; padding-left:22px; margin-top:4px;">
          <input type="text" id="val-custom-attacker-id" class="input-text" placeholder="Attacker ID (e.g. ollama:qwen3:8b)" />
          <input type="text" id="val-custom-attacker-url" class="input-text" placeholder="Base URL (e.g. http://localhost:11434)" />
        </div>
      </div>
    </div>
  `;
}

function wireModalActionButtons(container, showToast, onValidatedCallback) {
  const revalBtn = container.querySelector("#btn-reval-now");
  if (revalBtn) {
    revalBtn.addEventListener("click", () => openValidationModal(showToast, onValidatedCallback));
  }
  const miniBtn = container.querySelector("#btn-autofix-mini");
  if (miniBtn) {
    miniBtn.addEventListener("click", () => executeAutoFixPreview("mini", container, showToast, onValidatedCallback));
  }
  const judgeBtn = container.querySelector("#btn-autofix-judge");
  if (judgeBtn) {
    judgeBtn.addEventListener("click", () => executeAutoFixPreview("judge", container, showToast, onValidatedCallback));
  }
}

async function executeAutoFixPreview(engine, container, showToast, onValidatedCallback) {
  const previewZone = container.querySelector("#autofix-preview-zone");
  if (!previewZone) return;
  previewZone.innerHTML = `<div class="stat-pill">Running ${engine === "judge" ? "Judge LLM + Mini Engine" : "Mini Engine"} repair...</div>`;

  const selectedRadio = container.querySelector('input[name="attacker-res-mode"]:checked');
  const attackerResolution = selectedRadio ? selectedRadio.value : "";
  const customAttackerId = container.querySelector("#val-custom-attacker-id")?.value.trim() || "";
  const customAttackerUrl = container.querySelector("#val-custom-attacker-url")?.value.trim() || "";
  const rawYaml = document.getElementById("yaml-preview-editor").value;

  try {
    const res = await autoFixConfig({
      rawYaml,
      engine,
      attackerResolution,
      customAttackerId,
      customAttackerUrl,
    });
    pendingAutoFixResult = res;
    renderConfirmationPreview(previewZone, res, showToast, onValidatedCallback);
  } catch (err) {
    previewZone.innerHTML = `<div class="badge badge-danger">Auto-Fix Failed: ${escapeAttr(err.message)}</div>`;
  }
}

function renderConfirmationPreview(previewZone, res, showToast, onValidatedCallback) {
  const fixes = res.appliedFixes || [];
  previewZone.innerHTML = `
    <div style="padding:12px; border:1px solid var(--accent-emerald); background:var(--bg-elevated);">
      <div style="font-weight:700; margin-bottom:6px;">Proposed Changes (${escapeAttr(res.engineUsed)}):</div>
      <ul style="margin:0 0 10px 18px; padding:0; font-size:12px;">
        ${fixes.length > 0 ? fixes.map((f) => `<li>${escapeAttr(f)}</li>`).join("") : `<li>Normalized YAML formatting and defaults</li>`}
      </ul>
      <textarea readonly class="yaml-editor" style="height:150px; width:100%; margin-bottom:10px; font-size:11px;">${escapeAttr(res.fixedYaml)}</textarea>
      <div style="display:flex; gap:10px;">
        <button id="btn-confirm-apply-fix" class="btn btn-primary">✓ Confirm &amp; Apply Fixes to Config</button>
      </div>
    </div>
  `;
  const confirmBtn = previewZone.querySelector("#btn-confirm-apply-fix");
  if (confirmBtn) {
    confirmBtn.addEventListener("click", async () => {
      if (!pendingAutoFixResult) return;
      document.getElementById("yaml-preview-editor").value = pendingAutoFixResult.fixedYaml;
      appState.config = pendingAutoFixResult.config;
      const filename = document.getElementById("save-filename-input").value.trim() || "config.yaml";
      await saveConfigFile(filename, pendingAutoFixResult.fixedYaml);
      notify();
      showToast(`Applied ${fixes.length} fix(es) and saved ${filename}! Re-validating...`);
      if (onValidatedCallback) onValidatedCallback();
      await openValidationModal(showToast, onValidatedCallback);
    });
  }
}
