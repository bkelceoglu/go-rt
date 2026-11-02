// Modular REST API Client for GORT Command Center

export async function fetchCatalog() {
  const res = await fetch("/api/catalog");
  if (!res.ok) throw new Error("Failed to load catalog");
  return res.json();
}

export async function fetchDatasetPrompts(datasetName) {
  const res = await fetch(`/api/datasets/${encodeURIComponent(datasetName)}/prompts`);
  if (!res.ok) throw new Error(`Failed to load prompts for ${datasetName}`);
  return res.json();
}

export async function fetchConfigList() {
  const res = await fetch("/api/configs");
  if (!res.ok) throw new Error("Failed to list config files");
  return res.json();
}

export async function loadConfigFile(filename) {
  const res = await fetch(`/api/configs/load?file=${encodeURIComponent(filename)}`);
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to load config file");
  return data;
}

export async function saveConfigFile(filename, rawYaml) {
  const res = await fetch("/api/configs/save", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ filename, rawYaml }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to save config file");
  return data;
}

export async function validateConfig(rawYaml, checkEndpoint = false) {
  const res = await fetch("/api/configs/validate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rawYaml, checkEndpoint }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to validate configuration");
  return data;
}

export async function pingProviders(rawYaml) {
  const res = await fetch("/api/configs/ping-providers", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rawYaml }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to ping providers");
  return data;
}

export async function autoFixConfig(payload) {
  const res = await fetch("/api/configs/autofix", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Auto-fix failed");
  return data;
}

export async function saveCustomDataset(name, records) {
  const res = await fetch("/api/datasets/custom", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, records }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to save custom dataset");
  return data;
}

export async function fetchDashboardStats(evalId = "") {
  const q = evalId ? `?evalId=${encodeURIComponent(evalId)}` : "";
  const res = await fetch(`/api/dashboard/stats${q}`);
  if (!res.ok) throw new Error("Failed to load dashboard statistics");
  return res.json();
}

export async function fetchEvalRuns(limit = 50) {
  const res = await fetch(`/api/runs?limit=${limit}`);
  if (!res.ok) throw new Error("Failed to list evaluation runs");
  return res.json();
}

export async function fetchEvalRun(evalID) {
  const res = await fetch(`/api/runs/${encodeURIComponent(evalID)}`);
  if (!res.ok) throw new Error(`Failed to load evaluation run ${evalID}`);
  return res.json();
}

export async function fetchRunResults(evalID, filters = {}) {
  const params = new URLSearchParams();
  if (filters.failedOnly) params.set("failedOnly", "true");
  if (filters.plugin) params.set("plugin", filters.plugin);
  if (filters.category) params.set("category", filters.category);
  if (filters.strategy) params.set("strategy", filters.strategy);
  if (filters.limit) params.set("limit", filters.limit);
  const q = params.toString() ? `?${params.toString()}` : "";
  const res = await fetch(`/api/runs/${encodeURIComponent(evalID)}/results${q}`);
  if (!res.ok) throw new Error(`Failed to load results for ${evalID}`);
  return res.json();
}

export async function fetchResultDetail(evalID, resultID) {
  const res = await fetch(`/api/runs/${encodeURIComponent(evalID)}/results/${encodeURIComponent(resultID)}`);
  if (!res.ok) throw new Error(`Failed to load result detail ${resultID}`);
  return res.json();
}

export async function startAssessment({ configFile = "", dataset = "", rawYaml = "" }) {
  const res = await fetch("/api/assessments/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ configFile, dataset, rawYaml }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to start assessment");
  return data;
}

export async function stopAssessment() {
  const res = await fetch("/api/assessments/stop", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Failed to stop assessment");
  return data;
}

export async function fetchAssessmentStatus() {
  const res = await fetch("/api/assessments/status");
  if (!res.ok) throw new Error("Failed to fetch assessment status");
  return res.json();
}

export async function fetchLogTail(lines = 100) {
  const res = await fetch(`/api/logs/tail?lines=${lines}`);
  if (!res.ok) throw new Error("Failed to fetch log tail");
  return res.json();
}

export function streamAssessmentLogs(onMessage, onError) {
  const es = new EventSource("/api/logs/stream");
  es.onmessage = (event) => {
    if (onMessage) onMessage(event.data);
  };
  es.onerror = (err) => {
    if (onError) onError(err);
  };
  return es;
}

export async function fetchExecutiveReport(evalId = "") {
  const q = evalId ? `?evalId=${encodeURIComponent(evalId)}` : "";
  const res = await fetch(`/api/reports/executive${q}`);
  if (!res.ok) throw new Error("Failed to load executive report");
  return res.json();
}
