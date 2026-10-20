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
