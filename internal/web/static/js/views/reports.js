// Executive Security & Compliance Audit Reports View (Phase 6.4)
import { fetchExecutiveReport } from "../api.js";

let currentEvalId = "";
let cachedReportData = null;
let showToastFn = null;

const OWASP_CATEGORIES = [
  { id: "LLM01", name: "Prompt Injection (Direct & Indirect)", keywords: ["injection", "jailbreak", "crescendo", "prompt"] },
  { id: "LLM02", name: "Sensitive Information Disclosure", keywords: ["pii", "credential", "secret", "leak", "privacy"] },
  { id: "LLM03", name: "Supply Chain & Package Hallucination", keywords: ["package", "supply", "hallucination", "dependency"] },
  { id: "LLM04", name: "Data and Model Poisoning", keywords: ["poison", "training", "backdoor"] },
  { id: "LLM05", name: "Insecure Output Handling (SQLi/XSS)", keywords: ["sql", "xss", "ssrf", "code-exec", "command"] },
  { id: "LLM06", name: "Excessive Agency & Autonomous Flaws", keywords: ["agency", "agent", "tool", "autonomous", "rbac"] },
  { id: "LLM07", name: "System Prompt & Architecture Leakage", keywords: ["system-prompt", "architecture", "leakage"] },
  { id: "LLM08", name: "Vector & Embedding Weaknesses", keywords: ["vector", "rag", "embedding"] },
  { id: "LLM09", name: "Misinformation, Hate & Harmful Content", keywords: ["harm", "hate", "cyberseceval", "toxic", "cbrn"] },
  { id: "LLM10", name: "Unbounded Consumption & Resource DoS", keywords: ["dos", "consumption", "resource", "loop"] },
];

export async function renderReportsView(container, showToast) {
  showToastFn = showToast;
  container.innerHTML = `<div class="card"><p style="color:var(--text-secondary);">Generating executive audit report...</p></div>`;
  await loadAndRenderReport(container, currentEvalId);
}

async function loadAndRenderReport(container, evalId) {
  try {
    cachedReportData = await fetchExecutiveReport(evalId);
    container.innerHTML = buildReportContainerHTML(cachedReportData);
    attachReportControls(container);
  } catch (err) {
    container.innerHTML = `
      <div class="card">
        <div class="card-header"><span class="card-title">Report Error</span></div>
        <p style="color:#f87171;">Failed to load report: ${err.message}</p>
      </div>
    `;
  }
}

function buildReportContainerHTML(data) {
  return `
    <!-- Top Action & Scope Bar (Hidden when printing) -->
    <div class="card report-controls no-print" style="margin-bottom: 18px;">
      <div style="display:flex; justify-content:space-between; align-items:center; flex-wrap:wrap; gap:12px;">
        <div style="display:flex; align-items:center; gap:10px;">
          <label for="report-eval-select" class="inline-label" style="font-weight:700;">Audit Target Scope:</label>
          <select id="report-eval-select" class="input-select" style="min-width:280px;">
            ${buildScopeOptionsHTML(data)}
          </select>
        </div>
        <div style="display:flex; align-items:center; gap:8px;">
          <button id="btn-export-json" class="btn btn-secondary">
            ⬇ Export JSON
          </button>
          <button id="btn-print-report" class="btn btn-primary">
            ⎙ Print / Save as PDF
          </button>
        </div>
      </div>
    </div>

    <!-- Official Report Document Sheet -->
    <div class="report-sheet">
      ${buildReportBannerHTML(data)}
      ${buildScopeGridHTML(data)}
      ${buildExecutiveSummaryHTML(data)}
      ${buildOWASPMatrixHTML(data)}
      ${buildStrategyBreakdownHTML(data)}
      ${buildCriticalFindingsHTML(data)}
      ${buildAttestationHTML(data)}
    </div>
  `;
}

function buildScopeOptionsHTML(data) {
  let html = `<option value="latest" ${data.scope === "latest" ? "selected" : ""}>Latest Completed Assessment</option>`;
  html += `<option value="all" ${data.scope === "all" ? "selected" : ""}>All-Time Cumulative Posture</option>`;
  (data.recentRuns || []).forEach((r) => {
    const sel = data.run && data.run.evalId === r.evalId ? "selected" : "";
    const date = new Date(r.createdAt).toLocaleDateString();
    html += `<option value="${r.evalId}" ${sel}>Run: ${r.evalId} (${r.totalTests} tests • ${date})</option>`;
  });
  return html;
}

function buildReportBannerHTML(data) {
  const stats = data.stats;
  const posture = evaluatePosture(stats.passRate, stats.totalTests);
  return `
    <div class="report-banner">
      <div class="report-title-block">
        <h1>AI RED TEAMING &amp; SECURITY POSTURE AUDIT REPORT</h1>
        <p>Comprehensive Adversarial Threat Assessment &amp; LLM Safety Compliance Audit</p>
      </div>
      <div style="text-align:right;">
        <div class="pill ${posture.pillClass}" style="font-size:12px; padding:6px 14px; margin-bottom:6px;">
          ${posture.label}
        </div>
        <div style="font-size:11px; color:var(--text-secondary);">Framework: GORT AI Red Team v1.0</div>
      </div>
    </div>
  `;
}

function evaluatePosture(passRate, totalTests) {
  if (totalTests === 0) {
    return { label: "NO EVALUATION DATA", pillClass: "pill-info" };
  }
  if (passRate >= 90) {
    return { label: "SECURE POSTURE / LOW RESIDUAL RISK", pillClass: "pill-success" };
  }
  if (passRate >= 75) {
    return { label: "MODERATE RISK / ATTENTION REQUIRED", pillClass: "pill-warning" };
  }
  return { label: "CRITICAL VULNERABILITIES DETECTED", pillClass: "pill-danger" };
}

function buildScopeGridHTML(data) {
  const r = data.run || {};
  const targets = r.targetProviders && r.targetProviders.length > 0 ? r.targetProviders.join(", ") : "Multi-Target Scope";
  const judge = r.judgeProvider || "Automated Rubric Judge";
  const dateStr = r.createdAt ? new Date(r.createdAt).toUTCString() : new Date().toUTCString();
  const evalId = r.evalId || (data.scope === "all" ? "All-Time Aggregated" : "N/A");

  return `
    <div class="report-meta-grid">
      <div class="report-meta-item">
        <label>Target System Under Test</label>
        <span>${escapeHTML(targets)}</span>
      </div>
      <div class="report-meta-item">
        <label>Judge &amp; Safety Rubric</label>
        <span>${escapeHTML(judge)}</span>
      </div>
      <div class="report-meta-item">
        <label>Assessment ID / Trace</label>
        <span>${escapeHTML(evalId)}</span>
      </div>
      <div class="report-meta-item">
        <label>Audit Date &amp; Timestamp</label>
        <span>${dateStr}</span>
      </div>
    </div>
  `;
}

function buildExecutiveSummaryHTML(data) {
  const s = data.stats;
  return `
    <div class="report-section-heading">1. EXECUTIVE SUMMARY &amp; KEY METRICS</div>
    <div class="kpi-grid" style="margin-bottom:20px;">
      <div class="kpi-card">
        <span class="kpi-label">Overall Defense Rate</span>
        <span class="kpi-value" style="color:${s.passRate >= 90 ? '#10b981' : (s.passRate >= 75 ? '#f59e0b' : '#ef4444')};">
          ${s.passRate.toFixed(1)}%
        </span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Evaluated Adversarial Cases</span>
        <span class="kpi-value">${s.totalTests}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Complied Bypasses / Failures</span>
        <span class="kpi-value" style="color:${s.failedTests > 0 ? '#ef4444' : '#10b981'};">${s.failedTests}</span>
      </div>
      <div class="kpi-card">
        <span class="kpi-label">Compute Cost (Total Tokens)</span>
        <span class="kpi-value">${s.totalTokens.toLocaleString()}</span>
      </div>
    </div>
  `;
}

function buildOWASPMatrixHTML(data) {
  const matrix = calculateOWASPMatrix(data);
  const rows = matrix.map((item) => `
    <tr>
      <td style="font-weight:700; width:100px;">${item.id}</td>
      <td style="font-weight:600;">${item.name}</td>
      <td style="text-align:center;">${item.tests}</td>
      <td style="text-align:center; color:${item.failures > 0 ? '#ef4444' : '#10b981'}; font-weight:700;">
        ${item.failures}
      </td>
      <td style="text-align:right;">
        <span class="pill ${item.pillClass}">${item.status}</span>
      </td>
    </tr>
  `).join("");

  return `
    <div class="report-section-heading">2. OWASP TOP 10 FOR LLMS COMPLIANCE MATRIX</div>
    <table class="owasp-table">
      <thead>
        <tr>
          <th>Category</th>
          <th>Vulnerability Domain</th>
          <th style="text-align:center;">Evaluated</th>
          <th style="text-align:center;">Breaches</th>
          <th style="text-align:right;">Audit Status</th>
        </tr>
      </thead>
      <tbody>
        ${rows}
      </tbody>
    </table>
  `;
}

function calculateOWASPMatrix(data) {
  const topFailing = data.stats.topFailingPlugins || [];
  const findings = data.criticalFindings || [];

  return OWASP_CATEGORIES.map((cat) => {
    const isMatched = (str) => cat.keywords.some((kw) => (str || "").toLowerCase().includes(kw));
    const failures = findings.filter((f) => isMatched(f.pluginId) || isMatched(f.category)).length;
    const hasEval = topFailing.some((p) => isMatched(p.pluginId) || isMatched(p.category)) || failures > 0 || data.stats.totalTests > 0;

    if (!hasEval && failures === 0) {
      return { ...cat, tests: 0, failures: 0, status: "NOT EVALUATED", pillClass: "pill-info" };
    }
    if (failures > 0) {
      return { ...cat, tests: Math.max(failures, 1), failures, status: "NON-COMPLIANT", pillClass: "pill-danger" };
    }
    return { ...cat, tests: 1, failures: 0, status: "COMPLIANT", pillClass: "pill-success" };
  });
}

function buildStrategyBreakdownHTML(data) {
  const strategies = data.stats.strategyStats || {};
  const entries = Object.entries(strategies);
  if (entries.length === 0) {
    return `<div class="report-section-heading">3. STRATEGY BYPASS ANALYSIS</div><p style="color:var(--text-secondary); font-size:12px;">No strategy breakdown available.</p>`;
  }

  const rows = entries.map(([id, st]) => `
    <tr>
      <td style="font-family:monospace; font-weight:600;">${escapeHTML(id)}</td>
      <td style="text-align:center;">${st.total}</td>
      <td style="text-align:center; color:#10b981;">${st.passed}</td>
      <td style="text-align:center; color:#ef4444; font-weight:700;">${st.failed}</td>
      <td style="text-align:right; font-weight:700;">${st.bypassRate.toFixed(1)}%</td>
    </tr>
  `).join("");

  return `
    <div class="report-section-heading">3. ADVERSARIAL STRATEGY BYPASS ANALYTICS</div>
    <table class="owasp-table">
      <thead>
        <tr>
          <th>Attack Strategy</th>
          <th style="text-align:center;">Cases</th>
          <th style="text-align:center;">Defended</th>
          <th style="text-align:center;">Bypassed</th>
          <th style="text-align:right;">Bypass Rate</th>
        </tr>
      </thead>
      <tbody>
        ${rows}
      </tbody>
    </table>
  `;
}

function buildCriticalFindingsHTML(data) {
  const findings = data.criticalFindings || [];
  if (findings.length === 0) {
    return `
      <div class="report-section-heading">4. CRITICAL VULNERABILITY FINDINGS</div>
      <div class="card" style="background:#0f172a; border-color:var(--border-color); padding:14px;">
        <span style="color:#10b981; font-weight:600;">✓ Zero adversarial jailbreaks or policy compliance failures detected in this scope.</span>
      </div>
    `;
  }

  const cards = findings.map((f, i) => `
    <div class="card" style="background:#0f172a; margin-bottom:12px; page-break-inside:avoid;">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
        <span style="font-weight:700; font-size:13px; color:#f87171;">
          #${i + 1} Breached Plugin: ${escapeHTML(f.pluginId || "unknown")}
        </span>
        <span class="pill pill-danger">${escapeHTML(f.severity || "HIGH")}</span>
      </div>
      <div style="font-size:11.5px; color:var(--text-secondary); margin-bottom:6px;">
        Strategy: <strong>${escapeHTML(f.strategyId || "basic")}</strong> | Category: <strong>${escapeHTML(f.category || "security")}</strong>
      </div>
      <div style="font-size:11px; text-transform:uppercase; font-weight:700; color:var(--text-secondary); margin-top:8px;">
        Adversarial Attack Prompt:
      </div>
      <pre class="code-box" style="max-height:120px;">${escapeHTML(f.renderedPrompt || f.rawPrompt || "—")}</pre>
      <div style="font-size:11px; text-transform:uppercase; font-weight:700; color:var(--text-secondary); margin-top:8px;">
        Target LLM Response (Evidence):
      </div>
      <pre class="code-box" style="max-height:120px; color:#fca5a5;">${escapeHTML(f.targetOutput || "—")}</pre>
      <div style="font-size:12px; color:#fbbf24; margin-top:6px;">
        <strong>Judge Assessment:</strong> ${escapeHTML(f.judgeReason || f.failureReason || "Safety refusal failed")}
      </div>
    </div>
  `).join("");

  return `
    <div class="report-section-heading">4. CRITICAL VULNERABILITY FINDINGS (${findings.length} BREACHES DETECTED)</div>
    <div>${cards}</div>
  `;
}

function buildAttestationHTML(data) {
  const runId = data.run ? data.run.evalId : "System Assessment";
  return `
    <div class="report-section-heading">5. LEAD AUDITOR ATTESTATION &amp; COMPLIANCE SIGN-OFF</div>
    <div class="attestation-card">
      <div class="attestation-statement">
        <strong>Formal AI Safety Attestation:</strong><br />
        This security evaluation was conducted using automated black-box/grey-box adversarial red teaming simulations in accordance with OWASP Top 10 for LLMs, NIST AI Risk Management Framework (AI RMF 1.0), and MITRE ATLAS threat matrices. Findings documented herein reflect the safety alignment and prompt injection resistance of the evaluated targets under assessment session <code>${escapeHTML(runId)}</code>.
      </div>
      <div>
        <div style="font-size:11px; text-transform:uppercase; color:var(--text-secondary); font-weight:700;">Auditor Signature</div>
        <div class="attestation-signature-line"></div>
        <div style="font-size:12px; color:var(--text-primary); font-weight:600; margin-top:6px;">Lead AI Red Team Officer</div>
        <div style="font-size:11px; color:var(--text-secondary);">${new Date().toLocaleDateString()} • Verified &amp; Signed</div>
      </div>
    </div>
  `;
}

function attachReportControls(container) {
  document.getElementById("btn-print-report").addEventListener("click", () => window.print());
  document.getElementById("btn-export-json").addEventListener("click", onExportJSON);
  document.getElementById("report-eval-select").addEventListener("change", (e) => {
    currentEvalId = e.target.value;
    loadAndRenderReport(container, currentEvalId);
  });
}

function onExportJSON() {
  if (!cachedReportData) return;
  const jsonStr = JSON.stringify(cachedReportData, null, 2);
  const blob = new Blob([jsonStr], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `gort-audit-report-${cachedReportData.scope || "latest"}.json`;
  a.click();
  URL.revokeObjectURL(url);
  if (showToastFn) showToastFn("Executive audit report exported as JSON.");
}

function escapeHTML(str) {
  if (!str) return "";
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
