// Assessment Runs & Deep Transcript Inspector View
import { fetchEvalRuns, fetchEvalRun, fetchRunResults, fetchResultDetail } from "../api.js";
import { escapeAttr, appState } from "../state.js";

let selectedEvalID = "";
let activeFilter = "all"; // "all" | "failed" | "passed"
let searchQuery = "";
let strategyFilter = "all";
let cachedResults = [];

export async function renderRunsView(container, showToast) {
  container.innerHTML = `
    <div style="padding:20px;text-align:center;color:var(--text-secondary);">
      Loading evaluation runs from SQLite...
    </div>
  `;

  try {
    const data = await fetchEvalRuns(50);
    const runs = data.runs || [];
    if (runs.length === 0) {
      renderEmptyRunsState(container);
      return;
    }

    if (appState.selectedEvalID && runs.some((r) => r.evalId === appState.selectedEvalID)) {
      selectedEvalID = appState.selectedEvalID;
      appState.selectedEvalID = ""; // reset once consumed
    } else if (!selectedEvalID || !runs.some((r) => r.evalId === selectedEvalID)) {
      selectedEvalID = runs[0].evalId;
    }

    await mountRunsViewLayout(container, runs, showToast);
  } catch (err) {
    container.innerHTML = `
      <div class="card" style="border-color:var(--danger);">
        <div class="card-header"><span class="card-title" style="color:var(--danger);">Error Loading Assessment Runs</span></div>
        <p style="color:var(--text-secondary);">${escapeAttr(err.message)}</p>
      </div>
    `;
  }
}

function renderEmptyRunsState(container) {
  container.innerHTML = `
    <div class="card" style="text-align:center;padding:40px 20px;">
      <h3 style="margin-bottom:8px;">No Assessment Runs Found</h3>
      <p style="color:var(--text-secondary);font-size:13px;max-width:500px;margin:0 auto 16px;">
        No red teaming assessments have been executed or stored in SQLite yet. Use the Config Builder Wizard or CLI to launch a security assessment.
      </p>
    </div>
  `;
}

async function mountRunsViewLayout(container, runs, showToast) {
  const currentRun = runs.find((r) => r.evalId === selectedEvalID) || runs[0];
  const passRate = currentRun.totalTests > 0 ? ((currentRun.passedTests / currentRun.totalTests) * 100).toFixed(0) : "100";
  const isPassGreen = Number(passRate) >= 80;

  container.innerHTML = `
    <!-- RUN SELECTOR & RUN OVERVIEW -->
    <div class="card" style="margin-bottom:18px;">
      <div class="card-header" style="flex-wrap:wrap;gap:12px;">
        <div style="display:flex;align-items:center;gap:10px;">
          <label class="inline-label" for="sel-run-id"><strong>Select Run:</strong></label>
          <select id="sel-run-id" class="input-select" style="min-width:320px;">
            ${runs.map((r) => `
              <option value="${escapeAttr(r.evalId)}" ${r.evalId === selectedEvalID ? "selected" : ""}>
                ${escapeAttr(r.evalId)} — Pass: ${r.passedTests} | Fail: ${r.failedTests} (${r.totalTests} tests)
              </option>
            `).join("")}
          </select>
        </div>
        <div style="display:flex;gap:8px;">
          <span class="badge ${isPassGreen ? "badge-pass" : "badge-fail"}" style="font-size:12px;padding:5px 10px;">
            ${passRate}% Defended (${currentRun.passedTests}/${currentRun.totalTests})
          </span>
          <span class="badge badge-info" style="font-size:12px;padding:5px 10px;">
            ⏱ ${((currentRun.durationMs || 0) / 1000).toFixed(1)}s
          </span>
        </div>
      </div>

      <div class="form-grid" style="margin-top:10px;font-size:12.5px;">
        <div><strong>Description:</strong> <span style="color:var(--text-secondary);">${escapeAttr(currentRun.description || "N/A")}</span></div>
        <div><strong>Target Model:</strong> <code>${escapeAttr((currentRun.targetProviders || []).join(", ") || "default")}</code></div>
        <div><strong>Trace ID:</strong> <code>${escapeAttr(currentRun.traceId)}</code></div>
        <div><strong>Judge Model:</strong> <code>${escapeAttr(currentRun.judgeProvider || "default")}</code></div>
      </div>
    </div>

    <!-- FILTER TOOLBAR & RESULTS TABLE CONTAINER -->
    <div class="card">
      <div class="card-header" style="flex-wrap:wrap;gap:10px;">
        <div style="display:flex;gap:6px;" id="filter-btn-group">
          <button class="btn ${activeFilter === "all" ? "btn-primary" : "btn-outline"} btn-filter" data-val="all">All Tests</button>
          <button class="btn ${activeFilter === "failed" ? "btn-primary" : "btn-outline"} btn-filter" data-val="failed" style="${currentRun.failedTests > 0 ? "border-color:#ef4444;" : ""}">
            Bypassed / Failed (${currentRun.failedTests})
          </button>
          <button class="btn ${activeFilter === "passed" ? "btn-primary" : "btn-outline"} btn-filter" data-val="passed">
            Defended (${currentRun.passedTests})
          </button>
        </div>

        <div style="display:flex;gap:8px;align-items:center;">
          <input id="inp-result-search" class="input-text" style="width:200px;" placeholder="Search prompt, plugin..." value="${escapeAttr(searchQuery)}" />
        </div>
      </div>

      <div id="run-results-table-box" style="overflow-x:auto;">
        <div style="padding:16px;text-align:center;color:var(--text-secondary);">Loading test case results...</div>
      </div>
    </div>

    <!-- MODAL CONTAINER -->
    <div id="transcript-modal-root"></div>
  `;

  container.querySelector("#sel-run-id").addEventListener("change", (e) => {
    selectedEvalID = e.target.value;
    renderRunsView(container, showToast);
  });

  container.querySelectorAll(".btn-filter").forEach((btn) => {
    btn.addEventListener("click", () => {
      activeFilter = btn.dataset.val;
      renderRunsView(container, showToast);
    });
  });

  container.querySelector("#inp-result-search").addEventListener("input", (e) => {
    searchQuery = e.target.value.toLowerCase();
    renderFilteredResultsTable(container.querySelector("#run-results-table-box"), container, showToast);
  });

  await loadAndRenderResults(selectedEvalID, container, showToast);
}

async function loadAndRenderResults(evalID, container, showToast) {
  const tableBox = container.querySelector("#run-results-table-box");
  try {
    const res = await fetchRunResults(evalID);
    cachedResults = res.results || [];
    renderFilteredResultsTable(tableBox, container, showToast);
  } catch (err) {
    tableBox.innerHTML = `<p style="color:var(--danger);padding:16px;">Failed to load results: ${escapeAttr(err.message)}</p>`;
  }
}

function renderFilteredResultsTable(tableBox, container, showToast) {
  const filtered = cachedResults.filter((r) => {
    if (activeFilter === "failed" && r.pass) return false;
    if (activeFilter === "passed" && !r.pass) return false;
    if (searchQuery) {
      const q = searchQuery;
      const matchPlugin = (r.pluginId || "").toLowerCase().includes(q);
      const matchCat = (r.category || "").toLowerCase().includes(q);
      const matchPrompt = (r.rawPrompt || "").toLowerCase().includes(q) || (r.renderedPrompt || "").toLowerCase().includes(q);
      const matchReason = (r.judgeReason || "").toLowerCase().includes(q);
      if (!matchPlugin && !matchCat && !matchPrompt && !matchReason) return false;
    }
    return true;
  });

  if (filtered.length === 0) {
    tableBox.innerHTML = `<div style="padding:24px;text-align:center;color:var(--text-secondary);">No test results match the selected filter.</div>`;
    return;
  }

  tableBox.innerHTML = `
    <table class="data-table">
      <thead>
        <tr>
          <th>Verdict</th>
          <th>Plugin ID</th>
          <th>Strategy</th>
          <th>Severity</th>
          <th>Latency</th>
          <th>Turns</th>
          <th>Verdict Summary</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        ${filtered.map((r) => {
          const isPass = r.pass;
          const multiTurns = (r.conversationHistory || []).length;
          return `
            <tr>
              <td>
                <span class="badge ${isPass ? "badge-pass" : "badge-fail"}">${isPass ? "DEFENDED" : "BYPASSED"}</span>
              </td>
              <td><code>${escapeAttr(r.pluginId)}</code></td>
              <td><span class="badge badge-info">${escapeAttr(r.strategyId)}</span></td>
              <td><span class="badge badge-${escapeAttr(r.severity || "medium")}">${escapeAttr(r.severity || "med")}</span></td>
              <td>${r.latencyMs || 0}ms</td>
              <td>${multiTurns > 0 ? `<span class="badge badge-info">${multiTurns} turns</span>` : `<span style="color:var(--text-muted);">Single</span>`}</td>
              <td style="max-width:280px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">${escapeAttr(r.judgeReason || r.failureReason || "N/A")}</td>
              <td>
                <button class="btn btn-outline btn-inspect-transcript" data-id="${escapeAttr(r.resultId)}" style="padding:3px 8px;font-size:11px;">
                  Inspect 🔍
                </button>
              </td>
            </tr>
          `;
        }).join("")}
      </tbody>
    </table>
  `;

  tableBox.querySelectorAll(".btn-inspect-transcript").forEach((btn) => {
    btn.addEventListener("click", () => {
      openTranscriptModal(selectedEvalID, btn.dataset.id, container, showToast);
    });
  });
}

async function openTranscriptModal(evalID, resultID, container, showToast) {
  const modalRoot = container.querySelector("#transcript-modal-root");
  modalRoot.innerHTML = `
    <div class="modal-backdrop">
      <div class="modal-dialog">
        <div class="modal-header">
          <span class="card-title">Loading Deep Transcript...</span>
          <button class="btn btn-outline btn-close-modal">✕</button>
        </div>
        <div class="modal-body" style="text-align:center;padding:30px;color:var(--text-secondary);">
          Fetching prompt mutations, target outputs, judge reasoning, and OTel spans...
        </div>
      </div>
    </div>
  `;

  modalRoot.querySelector(".btn-close-modal").addEventListener("click", () => {
    modalRoot.innerHTML = "";
  });

  try {
    const data = await fetchResultDetail(evalID, resultID);
    mountTranscriptModalContent(modalRoot, data.result, data.spans || []);
  } catch (err) {
    modalRoot.querySelector(".modal-body").innerHTML = `
      <p style="color:var(--danger);">Failed to load transcript detail: ${escapeAttr(err.message)}</p>
    `;
  }
}

function mountTranscriptModalContent(modalRoot, res, spans) {
  let activeTab = "overview";
  const turns = res.conversationHistory || [];

  const renderModal = () => {
    modalRoot.innerHTML = `
      <div class="modal-backdrop">
        <div class="modal-dialog">
          <div class="modal-header">
            <div style="display:flex;align-items:center;gap:10px;">
              <span class="badge ${res.pass ? "badge-pass" : "badge-fail"}" style="font-size:12px;padding:4px 8px;">
                ${res.pass ? "DEFENDED / PASS" : "BYPASSED / VULNERABLE"}
              </span>
              <span style="font-weight:700;font-size:14px;"><code>${escapeAttr(res.pluginId)}</code></span>
              <span class="badge badge-info">${escapeAttr(res.strategyId)}</span>
            </div>
            <button class="btn btn-outline btn-close-modal">✕</button>
          </div>

          <div class="tab-strip" style="padding:10px 20px 0;margin-bottom:0;">
            <button class="tab-btn ${activeTab === "overview" ? "active" : ""}" data-tab="overview">Prompts &amp; Response</button>
            <button class="tab-btn ${activeTab === "judge" ? "active" : ""}" data-tab="judge">Judge Verdict &amp; Rubric</button>
            <button class="tab-btn ${activeTab === "history" ? "active" : ""}" data-tab="history">
              Multi-Turn Crescendo (${turns.length})
            </button>
            <button class="tab-btn ${activeTab === "otel" ? "active" : ""}" data-tab="otel">OpenTelemetry &amp; Spans (${spans.length})</button>
          </div>

          <div class="modal-body">
            ${renderModalTabBody(activeTab, res, turns, spans)}
          </div>

          <div class="modal-footer">
            <button class="btn btn-secondary btn-close-modal">Close Inspector</button>
          </div>
        </div>
      </div>
    `;

    modalRoot.querySelectorAll(".btn-close-modal").forEach((b) => {
      b.addEventListener("click", () => (modalRoot.innerHTML = ""));
    });

    modalRoot.querySelectorAll(".tab-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        activeTab = btn.dataset.tab;
        renderModal();
      });
    });
  };

  renderModal();
}

function renderModalTabBody(tab, res, turns, spans) {
  if (tab === "overview") {
    return `
      <div style="display:flex;flex-direction:column;gap:16px;">
        <div>
          <label style="font-weight:700;color:var(--text-secondary);font-size:12px;display:block;margin-bottom:4px;">RAW SEED PROMPT:</label>
          <div class="code-box">${escapeAttr(res.rawPrompt)}</div>
        </div>
        <div>
          <label style="font-weight:700;color:var(--text-secondary);font-size:12px;display:block;margin-bottom:4px;">RENDERED / MUTATED PROMPT SENT TO TARGET:</label>
          <div class="code-box" style="border-color:rgba(56,189,248,0.4);">${escapeAttr(res.renderedPrompt)}</div>
        </div>
        <div>
          <label style="font-weight:700;color:var(--text-secondary);font-size:12px;display:block;margin-bottom:4px;">TARGET LLM OUTPUT:</label>
          <div class="code-box" style="max-height:260px;overflow-y:auto;">${escapeAttr(res.targetOutput || "(No output)")}</div>
        </div>
        <div class="form-grid" style="font-size:12px;">
          <div><strong>Latency:</strong> ${res.latencyMs || 0} ms</div>
          <div><strong>Tokens:</strong> Prompt: ${res.tokenUsage?.prompt || 0} • Completion: ${res.tokenUsage?.completion || 0} • Total: ${res.tokenUsage?.total || 0}</div>
        </div>
      </div>
    `;
  }

  if (tab === "judge") {
    return `
      <div style="display:flex;flex-direction:column;gap:16px;">
        <div style="background:#0f172a;padding:14px;border-radius:8px;border:1px solid var(--border-color);">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;">
            <span style="font-weight:700;">VERDICT OUTCOME</span>
            <span class="badge ${res.pass ? "badge-pass" : "badge-fail"}" style="font-size:13px;padding:4px 10px;">
              Score: ${res.score} (${res.pass ? "Refused Harmful Request" : "Complied with Harmful Attack"})
            </span>
          </div>
          <div style="font-size:13px;line-height:1.5;">${escapeAttr(res.judgeReason || res.failureReason || "No explanation provided.")}</div>
        </div>

        <div>
          <label style="font-weight:700;color:var(--text-secondary);font-size:12px;display:block;margin-bottom:4px;">EVALUATION RUBRIC APPLIED:</label>
          <div class="code-box">${escapeAttr(res.judgeRubric || "Standard refusal safety rubric")}</div>
        </div>
      </div>
    `;
  }

  if (tab === "history") {
    if (turns.length === 0) {
      return `<div style="text-align:center;padding:30px;color:var(--text-secondary);">This test case was a single-turn attack mutation (no multi-turn Crescendo history).</div>`;
    }
    return `
      <div class="chat-container">
        ${turns.map((t) => {
          const isUser = t.role === "user";
          const isBacktrack = t.backtrack;
          return `
            <div class="chat-bubble ${isUser ? "user" : "assistant"} ${isBacktrack ? "backtrack" : ""}">
              <div style="display:flex;justify-content:space-between;margin-bottom:4px;font-size:11px;font-weight:700;color:var(--text-muted);">
                <span>Turn #${t.turn} — ${isUser ? "ATTACKER" : "TARGET ASSISTANT"}</span>
                ${isBacktrack ? `<span style="color:#ef4444;">[BACKTRACKED / REJECTED]</span>` : ""}
              </div>
              <div style="white-space:pre-wrap;">${escapeAttr(isUser ? t.prompt : t.output)}</div>
            </div>
          `;
        }).join("")}
      </div>
    `;
  }

  if (tab === "otel") {
    return `
      <div style="display:flex;flex-direction:column;gap:14px;">
        <div class="form-grid" style="font-size:12px;">
          <div><strong>Trace ID:</strong> <code>${escapeAttr(res.traceId)}</code></div>
          <div><strong>Span ID:</strong> <code>${escapeAttr(res.spanId)}</code></div>
        </div>
        <div style="font-weight:700;font-size:12px;color:var(--text-secondary);">ASSOCIATED OPENTELEMETRY GENAI SPANS (${spans.length}):</div>
        ${spans.length === 0 ? `<p style="color:var(--text-secondary);font-size:12px;">No standalone span objects recorded for trace.</p>` : `
          <table class="data-table">
            <thead><tr><th>Span ID</th><th>Name</th><th>Kind</th><th>Status</th><th>Duration</th></tr></thead>
            <tbody>
              ${spans.map((sp) => `
                <tr>
                  <td><code>${escapeAttr(sp.spanId)}</code></td>
                  <td>${escapeAttr(sp.name)}</td>
                  <td><span class="badge badge-info">${escapeAttr(sp.kind)}</span></td>
                  <td>${escapeAttr(sp.statusCode)}</td>
                  <td>${Math.round((sp.endTimeUnixNano - sp.startTimeUnixNano) / 1000000)}ms</td>
                </tr>
              `).join("")}
            </tbody>
          </table>
        `}
      </div>
    `;
  }
}
