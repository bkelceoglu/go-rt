// Executive Security Command Center Dashboard View
import { fetchDashboardStats } from "../api.js";
import { escapeAttr, appState } from "../state.js";

let currentScope = "latest"; // Default: "Sadece Son Oturum" (Latest Run)

export async function renderDashboardView(container, showToast, navigateToRoute, scope = currentScope) {
  currentScope = scope;
  container.innerHTML = `
    <div style="padding:20px;text-align:center;color:var(--text-secondary);">
      Loading executive security posture metrics from SQLite...
    </div>
  `;

  try {
    const stats = await fetchDashboardStats(currentScope);
    mountDashboardContent(container, stats, showToast, navigateToRoute);
  } catch (err) {
    container.innerHTML = `
      <div class="card" style="border-color:var(--danger);">
        <div class="card-header"><span class="card-title" style="color:var(--danger);">Failed to Load Dashboard</span></div>
        <p style="color:var(--text-secondary);">${escapeAttr(err.message)}</p>
      </div>
    `;
  }
}

function mountDashboardContent(container, stats, showToast, navigateToRoute) {
  const passRate = stats.totalTests > 0 ? (stats.passRate || 0).toFixed(1) : "100.0";
  const passClass = Number(passRate) >= 80 ? "success" : Number(passRate) >= 50 ? "warning" : "danger";
  const stratEntries = Object.entries(stats.strategyStats || {});
  const runs = stats.recentRuns || [];
  const latestRun = runs.length > 0 ? runs[0] : null;
  const isAllTime = stats.scope === "all";

  container.innerHTML = `
    <!-- SCOPE SELECTOR CONTROL BAR -->
    <div class="card" style="margin-bottom:18px;">
      <div class="card-header" style="flex-wrap:wrap;gap:12px;">
        <div style="display:flex;align-items:center;gap:10px;">
          <label class="inline-label" for="sel-dash-scope"><strong>Görüntüleme Kapsamı:</strong></label>
          <select id="sel-dash-scope" class="input-select" style="min-width:340px;">
            <option value="latest" ${currentScope === "latest" ? "selected" : ""}>
              ⚡ Sadece Son Oturum ${latestRun ? `(En Son: ${latestRun.evalId})` : ""}
            </option>
            <option value="all" ${currentScope === "all" ? "selected" : ""}>
              🌐 Tüm Zamanlar / All-Time Cumulative (${runs.length} Oturum)
            </option>
            ${runs.map((r, idx) => `
              <option value="${escapeAttr(r.evalId)}" ${currentScope === r.evalId ? "selected" : ""}>
                Oturum #${idx + 1}: ${escapeAttr(r.evalId)} (${r.passedTests}/${r.totalTests} Pass • ${escapeAttr(r.description || "Assessment")})
              </option>
            `).join("")}
          </select>
        </div>
        <div style="display:flex;gap:8px;align-items:center;">
          <span class="badge ${isAllTime ? "badge-info" : "badge-medium"}" style="font-size:12px;padding:5px 10px;">
            ${isAllTime ? `🌐 TÜM ZAMANLAR (${runs.length} OTURUM)` : `⚡ ${stats.selectedEvalId ? `OTURUM: ${stats.selectedEvalId}` : "SON OTURUM"}`}
          </span>
        </div>
      </div>
    </div>

    <!-- TOP ROW: POSTURE KPIS -->
    <div class="kpi-grid">
      <div class="kpi-card">
        <div class="kpi-label">${isAllTime ? "Overall Security Posture" : "Session Defense Rate"}</div>
        <div class="kpi-value" style="color:${passClass === "success" ? "#10b981" : passClass === "warning" ? "#f59e0b" : "#ef4444"};">${passRate}%</div>
        <div class="meter-bar">
          <div class="meter-fill ${passClass}" style="width:${passRate}%;"></div>
        </div>
        <div class="kpi-meta" style="margin-top:6px;">
          ${isAllTime ? "Tüm geçmiş testlerde başarı oranı" : `Seçili oturumda (${stats.passedTests}/${stats.totalTests}) test savunuldu`}
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-label">${isAllTime ? "Assessment Runs" : "Target System"}</div>
        <div class="kpi-value" style="font-size:${isAllTime ? "24px" : "18px"};">
          ${isAllTime ? `${stats.totalRuns || 0} Runs` : (latestRun?.targetProviders?.[0] || "Target Model")}
        </div>
        <div class="kpi-meta">${stats.totalTests || 0} test vakası değerlendirildi</div>
      </div>

      <div class="kpi-card">
        <div class="kpi-label">Vulnerabilities (Bypasses)</div>
        <div class="kpi-value" style="color:${stats.failedTests > 0 ? "#ef4444" : "#10b981"};">${stats.failedTests || 0}</div>
        <div class="kpi-meta">${stats.passedTests || 0} saldırı başarıyla savuşturuldu</div>
      </div>

      <div class="kpi-card">
        <div class="kpi-label">Compute &amp; Latency</div>
        <div class="kpi-value">${Math.round((stats.totalTokens || 0) / 1000)}k <span style="font-size:14px;color:var(--text-secondary);">toks</span></div>
        <div class="kpi-meta">Gecikme süresi: ${((stats.avgDurationMs || 0) / 1000).toFixed(1)}s</div>
      </div>
    </div>

    <!-- MIDDLE ROW: 2-COLUMN ANALYTICS -->
    <div style="display:grid;grid-template-columns:1fr 1fr;gap:18px;margin-bottom:20px;">
      <!-- LEFT: STRATEGY BYPASS ANALYTICS -->
      <div class="card" style="margin-bottom:0;">
        <div class="card-header">
          <span class="card-title">Attack Strategy Bypass Effectiveness</span>
          <span class="badge badge-info">${stratEntries.length} Strategies</span>
        </div>
        <div style="display:flex;flex-direction:column;gap:12px;padding:4px 0;">
          ${stratEntries.length === 0 ? `<p style="color:var(--text-secondary);font-size:13px;">No strategy data recorded for this scope.</p>` : ""}
          ${stratEntries.map(([sId, sStat]) => {
            const bypassRate = (sStat.bypassRate || 0).toFixed(1);
            const isDanger = Number(bypassRate) > 30;
            return `
              <div>
                <div style="display:flex;justify-content:space-between;align-items:center;font-size:12.5px;margin-bottom:4px;">
                  <span><code>${escapeAttr(sId || "direct")}</code></span>
                  <span><strong>${bypassRate}% Bypass</strong> (${sStat.failed || 0} / ${sStat.total || 0} breaches)</span>
                </div>
                <div class="meter-bar">
                  <div class="meter-fill ${isDanger ? "danger" : "warning"}" style="width:${bypassRate}%;"></div>
                </div>
              </div>
            `;
          }).join("")}
        </div>
      </div>

      <!-- RIGHT: SEVERITY & TOP FAILING PLUGINS -->
      <div class="card" style="margin-bottom:0;">
        <div class="card-header">
          <span class="card-title">Severity Impact &amp; Top Failing Plugins</span>
        </div>
        <div style="display:flex;gap:10px;margin-bottom:14px;">
          <div style="flex:1;background:#0f172a;padding:10px;border-radius:6px;border:1px solid var(--border-color);text-align:center;">
            <span class="badge badge-critical">Critical</span>
            <div style="font-size:18px;font-weight:700;margin-top:4px;">${stats.severityStats?.critical || 0}</div>
          </div>
          <div style="flex:1;background:#0f172a;padding:10px;border-radius:6px;border:1px solid var(--border-color);text-align:center;">
            <span class="badge badge-high">High</span>
            <div style="font-size:18px;font-weight:700;margin-top:4px;">${stats.severityStats?.high || 0}</div>
          </div>
          <div style="flex:1;background:#0f172a;padding:10px;border-radius:6px;border:1px solid var(--border-color);text-align:center;">
            <span class="badge badge-medium">Medium</span>
            <div style="font-size:18px;font-weight:700;margin-top:4px;">${stats.severityStats?.medium || 0}</div>
          </div>
          <div style="flex:1;background:#0f172a;padding:10px;border-radius:6px;border:1px solid var(--border-color);text-align:center;">
            <span class="badge badge-low">Low</span>
            <div style="font-size:18px;font-weight:700;margin-top:4px;">${stats.severityStats?.low || 0}</div>
          </div>
        </div>

        <div style="font-size:12px;font-weight:600;color:var(--text-secondary);margin-bottom:8px;">MOST EXPLOITED PLUGINS:</div>
        <div style="max-height:160px;overflow-y:auto;">
          ${(stats.topFailingPlugins || []).length === 0 ? `<p style="color:var(--text-secondary);font-size:12px;">Zero vulnerabilities recorded in this scope!</p>` : ""}
          ${(stats.topFailingPlugins || []).map((tf) => `
            <div style="display:flex;justify-content:space-between;align-items:center;padding:5px 0;border-bottom:1px solid rgba(255,255,255,0.05);font-size:12px;">
              <div><code>${escapeAttr(tf.pluginId || "unknown")}</code> <span class="badge badge-info" style="font-size:10px;">${escapeAttr(tf.category || "general")}</span></div>
              <span style="color:#ef4444;font-weight:700;">${tf.failures} bypasses</span>
            </div>
          `).join("")}
        </div>
      </div>
    </div>

    <!-- BOTTOM ROW: RECENT RUNS TABLE -->
    <div class="card">
      <div class="card-header">
        <span class="card-title">Recent Security Assessment Runs (${runs.length})</span>
        <button id="btn-dash-view-all-runs" class="btn btn-secondary">View All Runs in Runs Tab →</button>
      </div>
      <div style="overflow-x:auto;">
        <table class="data-table">
          <thead>
            <tr><th>Eval ID</th><th>Purpose / Target</th><th>Tests</th><th>Passed</th><th>Failed</th><th>Tokens</th><th>Duration</th><th>Action</th></tr>
          </thead>
          <tbody>
            ${runs.map((r) => {
              return `
                <tr>
                  <td><code>${escapeAttr(r.evalId)}</code></td>
                  <td>${escapeAttr(r.description || r.purpose || "Security Assessment")}</td>
                  <td><strong>${r.totalTests}</strong></td>
                  <td><span style="color:#10b981;">${r.passedTests}</span></td>
                  <td><span style="color:${r.failedTests > 0 ? "#ef4444" : "var(--text-muted);"};">${r.failedTests}</span></td>
                  <td>${r.totalTokens || 0}</td>
                  <td>${((r.durationMs || 0) / 1000).toFixed(1)}s</td>
                  <td>
                    <button class="btn btn-outline btn-dash-inspect-run" data-id="${escapeAttr(r.evalId)}" style="padding:3px 8px;font-size:11px;">Inspect →</button>
                  </td>
                </tr>
              `;
            }).join("")}
          </tbody>
        </table>
      </div>
    </div>
  `;

  container.querySelector("#sel-dash-scope")?.addEventListener("change", (e) => {
    currentScope = e.target.value;
    renderDashboardView(container, showToast, navigateToRoute, currentScope);
  });

  container.querySelector("#btn-dash-view-all-runs")?.addEventListener("click", () => {
    navigateToRoute("runs");
  });

  container.querySelectorAll(".btn-dash-inspect-run").forEach((btn) => {
    btn.addEventListener("click", () => {
      appState.selectedEvalID = btn.dataset.id;
      navigateToRoute("runs");
    });
  });
}
