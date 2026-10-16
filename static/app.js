"use strict";
const $ = (id) => document.getElementById(id);
const state = {
  user: null,
  mode: "login",
  view: "all",
  page: 1,
  total: 0,
  notes: [],
  current: null,
  dirty: false,
  busy: false,
  loading: false,
  query: "",
  controller: null,
  generation: 0,
};
let searchTimer, toastTimer;
const labels = {
  all: "All notes",
  pinned: "Pinned",
  archive: "Archive",
  trash: "Trash",
};
function announce(message, error = false) {
  clearTimeout(toastTimer);
  $("toast").textContent = message;
  $("toast").classList.toggle("error", error);
  $("toast").hidden = false;
  toastTimer = setTimeout(() => {
    $("toast").hidden = true;
  }, 4500);
}
async function api(path, options = {}) {
  let response;
  try {
    const timeout = AbortSignal.timeout(20000);
    response = await fetch("/v1" + path, {
      credentials: "same-origin",
      ...options,
      signal: options.signal
        ? AbortSignal.any([options.signal, timeout])
        : timeout,
      headers: { "Content-Type": "application/json", ...options.headers },
    });
  } catch (error) {
    if (error.name === "AbortError") throw error;
    throw new Error(
      error.name === "TimeoutError"
        ? "The server took too long to respond. Try again."
        : "Could not connect. Check your connection and try again.",
    );
  }
  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    const error = new Error(
      payload.error || "The request failed. Please try again.",
    );
    error.status = response.status;
    throw error;
  }
  return response.status === 204 ? null : response.json();
}
function showError(id, error) {
  const box = $(id);
  box.textContent =
    error.message || "Could not connect. Check your connection and try again.";
  if (error.status === 401 && id === "editor-error") {
    box.append(document.createTextNode(" "));
    const a = document.createElement("a");
    a.href = "/";
    a.target = "_blank";
    a.rel = "noopener";
    a.textContent = "Sign in in a new tab, then save again.";
    box.append(a);
  }
}
function ask(title, copy, action = "Discard changes", cancel = "Keep editing") {
  return new Promise((resolve) => {
    const d = $("decision"),
      trigger = document.activeElement;
    $("decision-title").textContent = title;
    $("decision-copy").textContent = copy;
    $("decision-confirm").textContent = action;
    $("decision-cancel").textContent = cancel;
    let result = false;
    const finish = () => {
      d.removeEventListener("close", finish);
      $("decision-confirm").onclick = null;
      $("decision-cancel").onclick = null;
      if (trigger?.isConnected) trigger.focus();
      resolve(result);
    };
    d.addEventListener("close", finish);
    $("decision-confirm").onclick = () => {
      result = true;
      d.close();
    };
    $("decision-cancel").onclick = () => d.close();
    d.showModal();
    $("decision-cancel").focus();
  });
}
async function canLeave() {
  if (state.busy) return false;
  return (
    !state.dirty ||
    (await ask(
      "Discard unsaved changes?",
      "Your unsaved edits will be lost. Download your draft first if you want to keep them.",
    ))
  );
}
function authMode(mode) {
  state.mode = mode;
  const register = mode === "register",
    legacy = mode === "claim";
  $("name-row").hidden = !register;
  $("legacy-row").hidden = !legacy;
  $("login-mode").classList.toggle("selected", mode === "login");
  $("register-mode").classList.toggle("selected", register);
  $("auth-title").textContent = legacy
    ? "Bring your notes along."
    : register
      ? "A notebook of your own."
      : "Welcome back.";
  $("auth-description").textContent = legacy
    ? "Keep your Notely notes and set up a new sign-in."
    : register
      ? "Start with a name and a thought."
      : "Open your notebook and pick up where you left off.";
  $("auth-submit").textContent = legacy
    ? "Move my notes"
    : register
      ? "Create notebook"
      : "Open notebook";
  $("password").autocomplete =
    mode === "login" ? "current-password" : "new-password";
  $("auth-error").textContent = "";
  for (const id of ["name", "username", "password", "legacy-key"])
    $(id).removeAttribute("aria-invalid");
}
$("login-mode").onclick = () => authMode("login");
$("register-mode").onclick = () => authMode("register");
$("legacy-mode").onclick = () => authMode("claim");
for (const button of document.querySelectorAll(".reveal"))
  button.onclick = () => {
    const input = $(button.dataset.for);
    const visible = input.type === "password";
    input.type = visible ? "text" : "password";
    button.textContent = visible ? "Hide" : "Show";
    button.setAttribute("aria-pressed", String(visible));
    button.setAttribute(
      "aria-label",
      (visible ? "Hide " : "Show ") +
        (button.dataset.for === "password" ? "password" : "Notely API key"),
    );
  };
$("auth-form").onsubmit = async (event) => {
  event.preventDefault();
  if (state.busy || event.isComposing) return;
  const mode = state.mode;
  const username = $("username").value.trim().toLowerCase(),
    password = $("password").value,
    name = $("name").value.trim();
  $("auth-error").textContent = "";
  for (const id of ["name", "username", "password", "legacy-key"])
    $(id).removeAttribute("aria-invalid");
  function invalid(id, text) {
    $(id).setAttribute("aria-invalid", "true");
    $("auth-error").textContent = text;
    $(id).focus();
  }
  if (mode === "register" && !name) return invalid("name", "Enter your name.");
  if (!/^[a-z0-9_-]{3,40}$/.test(username))
    return invalid(
      "username",
      "Use 3–40 letters, numbers, dashes or underscores.",
    );
  if (
    !password ||
    (mode !== "login" && new TextEncoder().encode(password).length < 12)
  )
    return invalid(
      "password",
      mode === "login"
        ? "Enter your password."
        : "Use at least 12 characters for your password.",
    );
  if (mode === "claim" && !$("legacy-key").value.trim())
    return invalid("legacy-key", "Enter your existing Notely API key.");
  state.busy = true;
  for (const control of document.querySelectorAll("#auth input, #auth button"))
    control.disabled = true;
  const submitLabel = $("auth-submit").textContent;
  $("auth-submit").textContent =
    mode === "login" ? "Signing in…" : "Creating notebook…";
  $("auth-form").setAttribute("aria-busy", "true");
  try {
    const data = { username, password };
    if (mode === "register") data.name = name;
    if (mode === "claim") data.legacy_key = $("legacy-key").value.trim();
    await api(
      "/auth/" +
        (mode === "claim"
          ? "claim"
          : mode === "register"
            ? "register"
            : "login"),
      { method: "POST", body: JSON.stringify(data) },
    );
    $("password").value = "";
    $("legacy-key").value = "";
    await enterWorkspace();
  } catch (error) {
    showError("auth-error", error);
    $("username").focus();
  } finally {
    state.busy = false;
    for (const control of document.querySelectorAll(
      "#auth input, #auth button",
    ))
      control.disabled = false;
    $("auth-submit").textContent = submitLabel;
    $("auth-form").removeAttribute("aria-busy");
    if ($("auth-error").textContent && !$("auth").hidden) $("username").focus();
  }
};
async function enterWorkspace() {
  state.user = await api("/users");
  $("auth").hidden = true;
  $("workspace").hidden = false;
  $("account-name").textContent = state.user.name;
  $("avatar").textContent = state.user.name.slice(0, 1).toUpperCase();
  document.title = "Leafmark — All notes";
  await loadNotes();
}
function formatDate(value) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(new Date(value));
}
function renderList() {
  const list = $("note-list");
  list.replaceChildren();
  $("note-count").textContent = String(state.total);
  $("view-title").textContent = labels[state.view];
  $("page-label").textContent =
    `Page ${state.page} of ${Math.max(1, Math.ceil(state.total / 30))}`;
  $("previous").disabled = state.page <= 1 || state.loading;
  $("next").disabled = state.page * 30 >= state.total || state.loading;
  for (const n of state.notes) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "note-card";
    button.classList.toggle("selected", n.id === state.current?.id);
    button.setAttribute("aria-pressed", String(n.id === state.current?.id));
    if (n.pinned) {
      const pin = document.createElement("span");
      pin.className = "pin-mark";
      pin.textContent = "☆";
      pin.setAttribute("aria-label", "Pinned");
      button.append(pin);
    }
    const title = document.createElement("strong");
    title.textContent = n.title || n.note.split("\n")[0] || "Untitled note";
    const preview = document.createElement("p");
    preview.textContent = n.note.slice(0, 240) || "No text yet.";
    const time = document.createElement("time");
    time.dateTime = n.updated_at;
    time.title = new Date(n.updated_at).toLocaleString();
    time.textContent = formatDate(n.updated_at);
    button.append(title, preview, time);
    button.onclick = async () => {
      if (await canLeave()) openNote(n);
    };
    list.append(button);
  }
  if (!state.notes.length) {
    const box = document.createElement("div");
    box.className = "empty-list";
    const heading = document.createElement("strong");
    heading.textContent = state.query
      ? "No matching thoughts."
      : state.view === "trash"
        ? "Nothing in the trash."
        : state.view === "archive"
          ? "A little room for later."
          : state.view === "pinned"
            ? "Keep a thought close."
            : "Your notebook starts here.";
    const p = document.createElement("p");
    p.textContent = state.query
      ? "Try another word or clear your search."
      : state.view === "trash"
        ? "Notes you move here can be restored."
        : state.view === "archive"
          ? "Archived notes stay here until you need them."
          : state.view === "pinned"
            ? "Pin a note to find it here."
            : "Choose New note and put something into words.";
    box.append(heading, p);
    list.append(box);
  }
}
async function loadNotes() {
  state.controller?.abort();
  const controller = new AbortController();
  state.controller = controller;
  const generation = ++state.generation;
  state.loading = true;
  $("list-status").textContent = "Loading notes…";
  $("note-list").setAttribute("aria-busy", "true");
  $("previous").disabled = true;
  $("next").disabled = true;
  try {
    const result = await api(
      `/notes?view=${state.view}&page=${state.page}&q=${encodeURIComponent(state.query)}`,
      { signal: controller.signal },
    );
    if (generation !== state.generation) return;
    if (state.page > 1 && !result.notes.length) {
      state.page = Math.max(1, Math.ceil(result.total / 30));
      return await loadNotes();
    }
    state.notes = result.notes;
    state.total = result.total;
    $("list-status").textContent = "";
  } catch (error) {
    if (error.name === "AbortError" || generation !== state.generation) return;
    $("list-status").textContent =
      error.status === 401
        ? "Session expired. Sign in in another tab, then reload."
        : error.message ||
          "Could not load notes. Check your connection and reload.";
    state.notes = [];
    state.total = 0;
  } finally {
    if (generation === state.generation) {
      state.loading = false;
      $("note-list").removeAttribute("aria-busy");
      renderList();
    }
  }
}
function grow() {
  for (const id of ["note-title", "note-body"]) {
    const area = $(id);
    area.style.height = "auto";
    area.style.height = area.scrollHeight + "px";
  }
}
function updateWords() {
  const text = $("note-body").value.trim();
  const words = text ? text.split(/\s+/u).length : 0;
  $("word-count").textContent =
    new Intl.NumberFormat().format(words) + (words === 1 ? " word" : " words");
  grow();
}
function openNote(note = null) {
  state.current = note ? { ...note } : null;
  state.draftID = note?.id || crypto.randomUUID();
  state.dirty = false;
  $("editor-empty").hidden = true;
  $("editor-form").hidden = false;
  $("workspace").classList.add("editing");
  $("note-title").value = note?.title || "";
  $("note-body").value = note?.note || "";
  $("editor-error").textContent = "";
  $("save-status").textContent = note ? "Saved" : "New note";
  $("note-date").textContent = note
    ? "Last edited " + formatDate(note.updated_at)
    : "A fresh page.";
  $("note-date").title = note ? new Date(note.updated_at).toLocaleString() : "";
  updateActions();
  updateWords();
  renderList();
  $("note-title").focus();
  document.title = (note?.title || "New note") + " — Leafmark";
}
function closeEditor() {
  state.current = null;
  state.dirty = false;
  $("editor-form").hidden = true;
  $("editor-empty").hidden = false;
  $("workspace").classList.remove("editing");
  document.title = `Leafmark — ${labels[state.view]}`;
}
function updateActions() {
  const n = state.current;
  for (const id of ["pin", "archive", "trash"])
    $(id).disabled = !n || state.busy;
  $("pin").textContent = n?.pinned ? "Unpin note" : "Pin note";
  $("pin").setAttribute("aria-pressed", String(!!n?.pinned));
  $("archive").textContent = n?.archived ? "Unarchive" : "Archive";
  $("archive").hidden = !!n?.trashed;
  $("trash").textContent = n?.trashed ? "Restore note" : "Move to trash";
  $("trash").classList.toggle("danger", !n?.trashed);
  $("pin").hidden = !!n?.trashed;
}
function draft() {
  return {
    id: state.current?.id || state.draftID,
    title: $("note-title").value,
    note: $("note-body").value,
    version: state.current?.version || 1,
    pinned: state.current?.pinned || false,
    archived: state.current?.archived || false,
    trashed: state.current?.trashed || false,
  };
}
function markDirty() {
  state.dirty = true;
  $("note-title").removeAttribute("aria-invalid");
  $("note-body").removeAttribute("aria-invalid");
  $("save-status").textContent = "Unsaved changes";
  updateWords();
}
$("note-title").oninput = markDirty;
$("note-body").oninput = markDirty;
async function saveNote(patch = {}, leave = false) {
  if (state.busy) return false;
  const data = { ...draft(), ...patch };
  if (!data.title.trim() && !data.note.trim()) {
    $("editor-error").textContent = "Write a title or note before saving.";
    $("note-title").setAttribute("aria-invalid", "true");
    $("note-title").focus();
    return false;
  }
  state.busy = true;
  $("save").disabled = true;
  $("editor-form").setAttribute("aria-busy", "true");
  $("note-title").readOnly = true;
  $("note-body").readOnly = true;
  $("save-status").textContent = "Saving…";
  $("editor-error").textContent = "";
  updateActions();
  try {
    const current = state.current;
    const saved = await api("/notes" + (current ? "/" + current.id : ""), {
      method: current ? "PUT" : "POST",
      body: JSON.stringify(data),
    });
    state.current = saved;
    state.dirty = false;
    $("note-title").value = saved.title;
    $("save-status").textContent = "Saved";
    $("note-date").textContent = "Last edited " + formatDate(saved.updated_at);
    $("note-date").title = new Date(saved.updated_at).toLocaleString();
    document.title = (saved.title || "Untitled note") + " — Leafmark";
    if (leave) closeEditor();
    await loadNotes();
    return true;
  } catch (error) {
    showError("editor-error", error);
    $("save-status").textContent =
      error.status === 409
        ? "Conflicting edits — draft preserved"
        : "Not saved — draft preserved";
    return false;
  } finally {
    state.busy = false;
    $("save").disabled = false;
    $("editor-form").removeAttribute("aria-busy");
    $("note-title").readOnly = false;
    $("note-body").readOnly = false;
    updateActions();
  }
}
$("editor-form").onsubmit = (event) => {
  event.preventDefault();
  if (!event.isComposing) void saveNote();
};
$("new-note").onclick = $("empty-new").onclick = async () => {
  if (await canLeave()) openNote();
};
$("back-list").onclick = async () => {
  if (await canLeave()) {
    closeEditor();
    renderList();
    $("new-note").focus();
  }
};
$("pin").onclick = async () => {
  if (await saveNote({ pinned: !state.current.pinned }))
    announce(state.current.pinned ? "Note pinned" : "Note unpinned");
};
$("archive").onclick = async () => {
  const archived = !state.current.archived;
  if (await saveNote({ archived }, true))
    announce(archived ? "Note archived" : "Note unarchived");
};
$("trash").onclick = async () => {
  const trashed = !state.current.trashed;
  if (
    trashed &&
    !(await ask(
      "Move this note to trash?",
      `“${$("note-title").value.trim() || "Untitled note"}” will stay in Trash until you restore it.`,
      "Move to trash",
      "Keep note",
    ))
  )
    return;
  if (
    await saveNote({ trashed, ...(trashed ? {} : { archived: false }) }, true)
  )
    announce(
      trashed
        ? "Note moved to trash. Restore it from Trash."
        : "Note restored to All notes",
    );
};
for (const button of document.querySelectorAll("[data-view]"))
  button.onclick = async () => {
    if (!(await canLeave())) return;
    state.view = button.dataset.view;
    state.page = 1;
    state.query = "";
    $("search").value = "";
    $("clear-search").hidden = true;
    clearTimeout(searchTimer);
    closeEditor();
    for (const item of document.querySelectorAll("[data-view]")) {
      item.classList.toggle("selected", item === button);
      if (item === button) item.setAttribute("aria-current", "page");
      else item.removeAttribute("aria-current");
    }
    await loadNotes();
  };
async function search() {
  state.page = 1;
  state.query = $("search").value.trim();
  await loadNotes();
}
$("search").oninput = (event) => {
  $("clear-search").hidden = !$("search").value;
  clearTimeout(searchTimer);
  if (!event.isComposing) searchTimer = setTimeout(search, 300);
};
$("search").oncompositionstart = () => clearTimeout(searchTimer);
$("search").oncompositionend = () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(search, 300);
};
$("search-form").onsubmit = (event) => {
  event.preventDefault();
  if (!event.isComposing) {
    clearTimeout(searchTimer);
    void search();
  }
};
$("clear-search").onclick = () => {
  clearTimeout(searchTimer);
  $("search").value = "";
  $("clear-search").hidden = true;
  $("search").focus();
  void search();
};
$("previous").onclick = async () => {
  if (await canLeave()) {
    state.page--;
    closeEditor();
    await loadNotes();
  }
};
$("next").onclick = async () => {
  if (await canLeave()) {
    state.page++;
    closeEditor();
    await loadNotes();
  }
};
$("refresh").onclick = async () => {
  if (await canLeave()) {
    closeEditor();
    await loadNotes();
  }
};
$("logout").onclick = async () => {
  if (!(await canLeave())) return;
  state.busy = true;
  $("logout").disabled = true;
  try {
    await api("/auth/logout", { method: "POST" });
    state.controller?.abort();
    state.generation++;
    clearTimeout(searchTimer);
    closeEditor();
    state.user = null;
    state.notes = [];
    state.total = 0;
    state.view = "all";
    state.page = 1;
    state.query = "";
    $("note-list").replaceChildren();
    $("note-title").value = "";
    $("note-body").value = "";
    $("search").value = "";
    $("clear-search").hidden = true;
    for (const b of document.querySelectorAll("[data-view]"))
      b.classList.toggle("selected", b.dataset.view === "all");
    $("workspace").hidden = true;
    $("auth").hidden = false;
    authMode("login");
    document.title = "Leafmark — Sign in";
    $("username").focus();
    announce("Signed out");
  } catch (error) {
    announce(error.message || "Could not sign out. Try again.", true);
  } finally {
    state.busy = false;
    $("logout").disabled = false;
  }
};
function download(content, type, filename) {
  const url = URL.createObjectURL(new Blob([content], { type }));
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
$("download-draft").onclick = () => {
  const title = $("note-title").value.trim();
  download(
    (title ? "# " + title + "\n\n" : "") + $("note-body").value,
    "text/markdown;charset=utf-8",
    "leafmark-draft.md",
  );
  announce("Draft downloaded");
};
$("export-all").onclick = async () => {
  const button = $("export-all");
  button.disabled = true;
  try {
    const result = await api("/export");
    download(
      JSON.stringify(result, null, 2),
      "application/json",
      "leafmark-notes.json",
    );
    announce("All notes downloaded");
  } catch (error) {
    announce(error.message || "Could not export. Try again.", true);
  } finally {
    button.disabled = false;
  }
};
document.addEventListener("keydown", (event) => {
  if (
    !event.isComposing &&
    (event.ctrlKey || event.metaKey) &&
    event.key.toLowerCase() === "s" &&
    !$("editor-form").hidden &&
    !$("workspace").hidden
  ) {
    event.preventDefault();
    void saveNote();
  }
});
window.addEventListener("beforeunload", (event) => {
  if (state.dirty) {
    event.preventDefault();
    event.returnValue = "";
  }
});
for (const a of document.querySelectorAll(".brand"))
  a.addEventListener("click", async (event) => {
    if (!$("workspace").hidden) {
      event.preventDefault();
      if (await canLeave()) {
        closeEditor();
        renderList();
      }
    }
  });
function connection() {
  $("connection").hidden = navigator.onLine;
}
window.addEventListener("online", connection);
window.addEventListener("offline", connection);
connection();
window.addEventListener("resize", grow);
// Enable controls only after their actions are bound.
for (const button of document.querySelectorAll("button[disabled]"))
  if (!["previous", "next"].includes(button.id)) button.disabled = false;
updateActions();
// Retire the course credential storage without reading or exposing its value.
try {
  localStorage.removeItem("currentUserAPIKey");
} catch {
  /* Storage may be disabled; session auth does not require it. */
}
(async () => {
  try {
    await enterWorkspace();
  } catch (error) {
    if (error.status !== 401) showError("auth-error", error);
  }
})();
