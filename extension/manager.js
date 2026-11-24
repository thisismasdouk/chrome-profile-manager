const HOST = "com.profile_manager.helper";
const $ = (id) => document.getElementById(id);

let state = { profiles: [], current: "", pending: null, sizes: {}, selected: new Set() };

// ---------- talking to the helper ----------

function callHelper(msg) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendNativeMessage(HOST, msg, (resp) => {
      if (chrome.runtime.lastError) return reject(new Error(chrome.runtime.lastError.message));
      if (!resp) return reject(new Error("The helper did not answer."));
      if (!resp.ok) return reject(new Error(resp.error || "Unknown error"));
      resolve(resp);
    });
  });
}

// A random token saved in this profile's extension storage. The helper looks
// for it on disk to know which profile you are using, so that one is protected.
async function getToken() {
  let { pmToken } = await chrome.storage.local.get("pmToken");
  if (!pmToken) {
    const bytes = crypto.getRandomValues(new Uint8Array(16));
    pmToken = "pm" + Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  }
  // Re-save on every visit so the token is freshly written to disk.
  await chrome.storage.local.set({ pmToken, pmSeen: Date.now() });
  return pmToken;
}

// ---------- subscription (7-day free trial, then paid) ----------

let license = { canDelete: true, kind: "free" };
const extpay = (typeof EXTPAY_ID !== "undefined" && EXTPAY_ID) ? ExtPay(EXTPAY_ID) : null;

async function checkLicense() {
  if (!extpay) {
    license = { canDelete: true, kind: "free" };
  } else {
    try {
      const user = await extpay.getUser();
      if (user.paid) {
        license = { canDelete: true, kind: "paid" };
      } else {
        const left = TRIAL_DAYS - (Date.now() - new Date(user.installedAt).getTime()) / 86400000;
        license = left > 0 ? { canDelete: true, kind: "trial", daysLeft: Math.ceil(left) }
                           : { canDelete: false, kind: "expired" };
      }
      await chrome.storage.local.set({ pmLicense: license });
    } catch (e) {
      // Offline or payment service unreachable: use the last known status.
      const { pmLicense } = await chrome.storage.local.get("pmLicense");
      license = pmLicense || { canDelete: true, kind: "unknown" };
    }
  }
  renderPlan();
  render();
}

function subscribe() { if (extpay) extpay.openPaymentPage(); }

function renderPlan() {
  const p = $("plan");
  p.replaceChildren();
  if (license.kind === "trial") {
    p.append(el("span", { text: `Free trial: ${license.daysLeft} day${license.daysLeft === 1 ? "" : "s"} left` }));
    const b = el("button", { class: "primary", text: `Subscribe · ${PRICE_TEXT}` });
    b.onclick = subscribe;
    p.append(b);
  } else if (license.kind === "expired") {
    p.append(el("span", { text: "Free trial ended" }));
    const b = el("button", { class: "primary", text: `Subscribe · ${PRICE_TEXT}` });
    b.onclick = subscribe;
    p.append(b);
  } else if (license.kind === "paid") {
    const b = el("button", { class: "ghost", text: "Manage subscription" });
    b.onclick = subscribe;
    p.append(el("span", { text: "Subscribed ✓" }), b);
  }
  p.classList.toggle("hidden", p.childElementCount === 0);
}

// After paying in the other tab, coming back here re-checks the status.
window.addEventListener("focus", () => { if (extpay) checkLicense(); });

// ---------- helpers ----------

function fmtSize(bytes) {
  if (bytes == null) return "";
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  while (bytes >= 1024 && i < u.length - 1) { bytes /= 1024; i++; }
  return `${bytes.toFixed(bytes < 10 && i > 0 ? 1 : 0)} ${u[i]}`;
}

function fmtAgo(sec) {
  if (!sec) return "never used";
  const d = (Date.now() / 1000 - sec) / 86400;
  if (d < 1) return "used today";
  if (d < 2) return "used yesterday";
  if (d < 60) return `used ${Math.floor(d)} days ago`;
  if (d < 730) return `used ${Math.floor(d / 30)} months ago`;
  return `used ${Math.floor(d / 365)} years ago`;
}

function colorFor(p) {
  if (p.color && p.color !== "#000000") return p.color;
  let h = 0;
  for (const c of p.dir) h = (h * 31 + c.charCodeAt(0)) % 360;
  return `hsl(${h} 55% 45%)`;
}

function isLight(c) {
  const m = /^#([0-9a-f]{6})$/i.exec(c);
  if (!m) return false;
  const n = parseInt(m[1], 16);
  return (0.299 * (n >> 16) + 0.587 * ((n >> 8) & 255) + 0.114 * (n & 255)) > 170;
}

function el(tag, attrs = {}, ...kids) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") e.className = v;
    else if (k === "text") e.textContent = v;
    else e.setAttribute(k, v);
  }
  for (const k of kids) if (k) e.append(k);
  return e;
}

function showBanner(kind, text, buttonLabel, onClick) {
  const b = $("banner");
  b.className = `banner ${kind}`;
  b.replaceChildren(el("span", { text }));
  if (buttonLabel) {
    const btn = el("button", { class: "ghost", text: buttonLabel });
    btn.onclick = onClick;
    b.append(btn);
  }
}
function hideBanner() { $("banner").className = "banner hidden"; }

const pendingDirs = () => new Set(state.pending ? state.pending.dirs : []);
const selectable = (p) => !p.isCurrent && !pendingDirs().has(p.dir);

// ---------- rendering ----------

function visibleProfiles() {
  const q = $("search").value.trim().toLowerCase();
  let list = state.profiles.filter((p) =>
    !q || [p.name, p.email, p.fullName, p.dir].some((s) => (s || "").toLowerCase().includes(q)));
  const sort = $("sort").value;
  list = [...list].sort((a, b) => {
    if (a.isCurrent !== b.isCurrent) return a.isCurrent ? -1 : 1;
    if (sort === "lastUsed") return (a.lastUsed || 0) - (b.lastUsed || 0);
    if (sort === "size") return (state.sizes[b.dir] || 0) - (state.sizes[a.dir] || 0);
    return a.name.localeCompare(b.name, undefined, { sensitivity: "base" });
  });
  return list;
}

function render() {
  const list = $("list");
  list.replaceChildren();
  const vis = visibleProfiles();
  const pend = pendingDirs();
  for (const p of vis) {
    const locked = !selectable(p);
    const li = el("li", { class: [locked ? "locked" : "", state.selected.has(p.dir) ? "selected" : "", pend.has(p.dir) ? "pending" : ""].join(" ") });
    const cb = el("input", { type: "checkbox", "aria-label": `Select ${p.name}` });
    cb.checked = state.selected.has(p.dir);
    cb.disabled = locked;

    const av = el("div", { class: "avatar" });
    av.style.background = colorFor(p);
    if (isLight(colorFor(p))) av.style.color = "#1d2330";
    if (p.avatar) av.append(el("img", { src: p.avatar, alt: "" }));
    else av.textContent = (p.name || "?").trim().charAt(0).toUpperCase();

    const name = el("div", { class: "name" }, el("span", { text: p.name }));
    if (p.isCurrent) name.append(el("span", { class: "badge", text: "You're using this one" }));
    if (pend.has(p.dir)) name.append(el("span", { class: "badge pending", text: "Will be deleted" }));
    const metaBits = [p.email || p.fullName || "Not signed in", fmtAgo(p.lastUsed), `folder: ${p.dir}`];
    const info = el("div", { class: "info" }, name, el("div", { class: "meta", text: metaBits.join(" · ") }));
    const size = el("div", { class: "size", text: state.sizes[p.dir] != null ? fmtSize(state.sizes[p.dir]) : "…" });

    li.append(cb, av, info, size);
    if (!locked) {
      const toggle = () => {
        state.selected.has(p.dir) ? state.selected.delete(p.dir) : state.selected.add(p.dir);
        render();
      };
      li.addEventListener("click", (e) => { if (e.target !== cb) toggle(); });
      cb.addEventListener("change", toggle);
    }
    list.append(li);
  }
  $("empty").classList.toggle("hidden", vis.length > 0);

  const selectableVis = vis.filter(selectable);
  const all = $("selectAll");
  const nSelVis = selectableVis.filter((p) => state.selected.has(p.dir)).length;
  all.checked = selectableVis.length > 0 && nSelVis === selectableVis.length;
  all.indeterminate = nSelVis > 0 && nSelVis < selectableVis.length;
  all.disabled = selectableVis.length === 0;

  const n = state.selected.size;
  const total = [...state.selected].reduce((s, d) => s + (state.sizes[d] || 0), 0);
  $("selCount").textContent = n === 0 ? "Nothing selected" :
    `${n} profile${n > 1 ? "s" : ""} selected${total ? ` · frees about ${fmtSize(total)}` : ""}`;
  $("deleteBtn").disabled = n === 0;
  const label = $("deleteLabel");
  if (!license.canDelete) {
    label.textContent = `Subscribe to delete · ${PRICE_TEXT}`;
    $("deleteBtn").disabled = false;
  } else {
    label.textContent = "Delete selected";
  }
}

function renderPendingBanner() {
  const r = state.lastResult;
  if (state.pending) {
    const n = state.pending.dirs.length;
    showBanner("warn",
      `${n} profile${n > 1 ? "s are" : " is"} waiting to be deleted. Quit Chrome completely (menu ⋮ → Exit) to finish.`,
      "Cancel deletion", async () => {
        await callHelper({ cmd: "cancel" });
        load();
      });
  } else if (r) {
    const failed = Object.entries(r.failed || {});
    const del = r.deleted || [];
    const names = (dirs) => dirs.map((d) => (r.names && r.names[d]) || d).join(", ");
    let text = del.length ? `Deleted ${del.length} profile${del.length > 1 ? "s" : ""}: ${names(del)}.` : "";
    if (failed.length) text += ` Problems: ${failed.map(([d, why]) => `${d === "*" ? "" : ((r.names && r.names[d]) || d) + ": "}${why}`).join("; ")}`;
    showBanner(failed.length ? "err" : "ok", text.trim(), "Dismiss", async () => {
      await callHelper({ cmd: "clearResult" });
      hideBanner();
    });
    if (del.length && !failed.length) maybeAskForReview(del.length);
  } else {
    hideBanner();
  }
}

// One gentle review request, after the first cleanup that worked.
async function maybeAskForReview(n) {
  if (typeof STORE_URL === "undefined" || !STORE_URL) return;
  const { pmReviewAsked } = await chrome.storage.local.get("pmReviewAsked");
  if (pmReviewAsked) return;
  await chrome.storage.local.set({ pmReviewAsked: Date.now() });
  const a = el("a", { href: STORE_URL.replace(/\/$/, "") + "/reviews", target: "_blank", rel: "noopener",
    class: "review-link", text: "Saved you some clicks? Leave a ★★★★★ review" });
  $("banner").querySelector("span").append(" ", a);
}

// ---------- loading ----------

async function load() {
  $("refresh").disabled = true;
  try {
    const token = await getToken();
    const resp = await callHelper({ cmd: "list", token });
    $("setup").classList.add("hidden");
    $("listCard").classList.remove("hidden");
    $("bar").classList.remove("hidden");
    state.profiles = resp.profiles;
    state.current = resp.current;
    state.pending = resp.pending || null;
    state.lastResult = resp.lastResult || null;
    // drop selections that no longer exist or are now locked
    for (const d of [...state.selected]) {
      const p = state.profiles.find((x) => x.dir === d);
      if (!p || !selectable(p)) state.selected.delete(d);
    }
    render();
    renderPendingBanner();
    if (!state.current) {
      showBanner("err", "Couldn't tell which profile you're using, so deleting is turned off for safety. Click Refresh to try again.");
      $("deleteBtn").disabled = true;
      $("deleteBtn").dataset.blocked = "1";
    } else {
      delete $("deleteBtn").dataset.blocked;
    }
    loadSizes();
  } catch (e) {
    const missing = /not found|forbidden|Native host has exited|Error when communicating/i.test(e.message);
    if (missing) {
      $("setup").classList.remove("hidden");
      $("listCard").classList.add("hidden");
      $("bar").classList.add("hidden");
      $("setupError").textContent = e.message;
      hideBanner();
      showSetup();
    } else {
      showBanner("err", e.message);
    }
  } finally {
    $("refresh").disabled = false;
  }
}

// ---------- helper not installed yet ----------

let setupTimer = null;
async function showSetup() {
  let os = "win";
  try { os = (await chrome.runtime.getPlatformInfo()).os; } catch (e) {}
  const key = os === "mac" ? "mac" : os === "linux" ? "linux" : "win";
  $("downloadHelper").href = HELPER_DOWNLOADS[key];
  $("downloadHelper").textContent = { win: "Download the helper for Windows", mac: "Download the helper for Mac", linux: "Download the helper for Linux" }[key];
  $("runStep").textContent = {
    win: "If Windows shows \"Windows protected your PC\", click More info, then Run anyway.",
    mac: "Open the downloaded zip, then right-click \"Install Profile Manager helper\" and choose Open.",
    linux: "Unpack the download and run ./install-linux.sh.",
  }[key];
  $("runStepShort").textContent = { win: "and click OK", mac: "right-click, Open", linux: "run the installer" }[key];
  // Keep checking quietly; switch to the profile list as soon as the helper answers.
  if (!setupTimer) {
    setupTimer = setInterval(async () => {
      try {
        await callHelper({ cmd: "ping" });
        clearInterval(setupTimer);
        setupTimer = null;
        load();
      } catch (e) { /* not installed yet */ }
    }, 3000);
  }
}

async function loadSizes() {
  const dirs = state.profiles.map((p) => p.dir).filter((d) => state.sizes[d] == null);
  if (!dirs.length) return render();
  try {
    const resp = await callHelper({ cmd: "sizes", dirs });
    Object.assign(state.sizes, resp.sizes);
  } catch (e) { /* sizes are optional */ }
  render();
}

// ---------- events ----------

$("refresh").onclick = () => { state.sizes = {}; load(); };
$("search").oninput = render;
$("sort").onchange = render;
$("selectAll").onchange = (e) => {
  for (const p of visibleProfiles().filter(selectable)) {
    e.target.checked ? state.selected.add(p.dir) : state.selected.delete(p.dir);
  }
  render();
};

$("deleteBtn").onclick = () => {
  if (!license.canDelete) return subscribe();
  if ($("deleteBtn").dataset.blocked || state.selected.size === 0) return;
  const ul = $("confirmList");
  ul.replaceChildren(...[...state.selected].map((d) => {
    const p = state.profiles.find((x) => x.dir === d);
    return el("li", { text: `${p ? p.name : d}${p && p.email ? ` (${p.email})` : ""}` });
  }));
  const n = state.selected.size;
  $("modalTitle").textContent = `Delete ${n} profile${n > 1 ? "s" : ""}?`;
  $("understand").checked = false;
  $("confirmDelete").disabled = true;
  $("confirmDelete").textContent = `Delete ${n} profile${n > 1 ? "s" : ""}`;
  $("modal").classList.remove("hidden");
  $("cancelModal").focus();
};
$("understand").onchange = (e) => { $("confirmDelete").disabled = !e.target.checked; };
$("cancelModal").onclick = () => $("modal").classList.add("hidden");
document.addEventListener("keydown", (e) => { if (e.key === "Escape") $("modal").classList.add("hidden"); });

$("confirmDelete").onclick = async () => {
  if (!license.canDelete) return subscribe();
  const now = document.querySelector('input[name="when"]:checked').value === "now";
  $("confirmDelete").disabled = true;
  try {
    const token = await getToken();
    const resp = await callHelper({ cmd: "delete", token, dirs: [...state.selected], quitNow: now, relaunch: now });
    $("modal").classList.add("hidden");
    state.pending = resp.pending;
    state.selected.clear();
    render();
    if (now) {
      showBanner("warn", "Closing Chrome now. The selected profiles will be deleted, then Chrome reopens with your tabs in a few seconds…");
    } else {
      renderPendingBanner();
    }
  } catch (e) {
    $("modal").classList.add("hidden");
    showBanner("err", e.message);
  }
};

load();
checkLicense();
