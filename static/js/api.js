function appURL(path) {
  const basePath = window.__UNISHARE_CONFIG__?.basePath || "";
  return `${basePath}${path}`;
}

export async function getSession() {
  return request(appURL("/api/session"));
}

export async function login(token) {
  return request(appURL("/api/login"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });
}

export async function logout() {
  return request(appURL("/api/logout"), { method: "POST" });
}

export async function listItems() {
  return request(appURL("/api/items"));
}

export async function createItem(form) {
  return request(appURL("/api/items"), { method: "POST", body: form });
}

export async function deleteItem(id) {
  const response = await fetch(appURL(`/api/items/${id}`), { method: "DELETE" });
  if (!response.ok) {
    throw new Error(await response.text());
  }
}

async function request(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || response.statusText);
  }
  return response.json();
}
