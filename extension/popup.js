const $ = (selector) => document.querySelector(selector);
const state = { tabId: null, enabled: false };

function shellQuote(value) {
  return `'${String(value).replace(/'/g, `'"'"'`)}'`;
}

function commandFor(url) {
  return `open-stream-saver download --url ${shellQuote(url)} --acknowledge-rights`;
}

function setStatus(enabled) {
  state.enabled = enabled;
  $("#status-title").textContent = enabled ? "Discovery is on" : "Discovery is off";
  $("#status-copy").textContent = enabled
    ? "This session is observing public .m3u8, .mpd, and .mp4 requests. It does not collect credentials or modify traffic."
    : "Turn it on to allow this extension to observe public HTTP(S) media requests in the current browser session.";
  $("#toggle-discovery").textContent = enabled ? "Disable discovery" : "Enable discovery";
}

function setSaveStatus(text, type = "") {
  const element = $("#save-status");
  element.textContent = text;
  element.dataset.state = type;
}

function formatTime(timestamp) {
  if (!timestamp) return "Unknown time";
  return new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" }).format(new Date(timestamp));
}

async function copyCommand(url, button) {
  const original = button.textContent;
  try {
    await navigator.clipboard.writeText(commandFor(url));
    button.textContent = "Copied";
  } catch {
    button.textContent = "Copy failed";
  }
  window.setTimeout(() => {
    button.textContent = original;
  }, 1200);
}

async function saveLocally(stream, button, variantInput) {
  if (!$("#acknowledge-rights").checked) {
    setSaveStatus("Confirm your right to save this public media before sending any local task.", "error");
    return;
  }
  const original = button.textContent;
  button.disabled = true;
  button.textContent = "Saving locally…";
  setSaveStatus("The local host is validating the public URL and starting a bounded download.");
  const variant = Number(variantInput?.value ?? -1);
  try {
    const response = await chrome.runtime.sendMessage({
      type: "saveLocally",
      url: stream.url,
      acknowledgeRights: true,
      workers: 4,
      variant: Number.isInteger(variant) ? variant : -1,
    });
    if (!response?.ok) {
      throw new Error(response?.error || "The local host could not start the download.");
    }
    setSaveStatus(`Saved locally: ${response.output}`, "success");
    button.textContent = "Saved";
  } catch (error) {
    const message = error?.message || "The local host could not start the download.";
    setSaveStatus(`${message} Install and register the optional native host, then try again.`, "error");
    button.textContent = "Save locally";
  } finally {
    button.disabled = false;
    window.setTimeout(() => {
      if (button.textContent === "Saved") button.textContent = original;
    }, 1800);
  }
}

function renderStreams(streams) {
  const list = $("#stream-list");
  const empty = $("#empty-state");
  list.replaceChildren();
  empty.hidden = streams.length > 0;

  for (const stream of streams) {
    const item = document.createElement("li");
    item.className = "stream-item";

    const metadata = document.createElement("div");
    metadata.className = "stream-metadata";

    const top = document.createElement("div");
    top.className = "stream-topline";
    const type = document.createElement("span");
    type.className = `badge ${stream.kind}`;
    type.textContent = stream.kind.toUpperCase();
    const time = document.createElement("time");
    time.textContent = formatTime(stream.discoveredAt);
    top.append(type, time);

    const url = document.createElement("code");
    url.className = "url";
    url.textContent = stream.url;
    url.title = stream.url;
    metadata.append(top, url);

    const actions = document.createElement("div");
    actions.className = "stream-actions";
    let variantInput;
    if (stream.kind === "m3u8") {
      const variant = document.createElement("label");
      variant.className = "variant-control";
      variant.textContent = "HLS variant";
      variantInput = document.createElement("input");
      variantInput.type = "number";
      variantInput.min = "-1";
      variantInput.value = "-1";
      variantInput.title = "-1 selects the highest advertised muxed variant. Use the CLI inspect-hls command to list indices.";
      variant.append(variantInput);
      actions.append(variant);
    }

    const save = document.createElement("button");
    save.className = "primary compact";
    save.type = "button";
    save.textContent = "Save locally";
    save.addEventListener("click", () => saveLocally(stream, save, variantInput));

    const copy = document.createElement("button");
    copy.className = "secondary compact";
    copy.type = "button";
    copy.textContent = "Copy CLI command";
    copy.addEventListener("click", () => copyCommand(stream.url, copy));

    actions.append(save, copy);
    item.append(metadata, actions);
    list.append(item);
  }
}

async function getActiveTab() {
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  return tab;
}

async function refresh() {
  const status = await chrome.runtime.sendMessage({ type: "status" });
  setStatus(Boolean(status?.enabled));

  const tab = await getActiveTab();
  state.tabId = tab?.id ?? null;
  if (!Number.isInteger(state.tabId)) {
    renderStreams([]);
    return;
  }
  const response = await chrome.runtime.sendMessage({ type: "getStreams", tabId: state.tabId });
  renderStreams(Array.isArray(response?.streams) ? response.streams : []);
}

$("#toggle-discovery").addEventListener("click", async () => {
  const button = $("#toggle-discovery");
  button.disabled = true;
  try {
    if (state.enabled) {
      await chrome.runtime.sendMessage({ type: "disableDiscovery" });
    } else {
      const response = await chrome.runtime.sendMessage({ type: "enableDiscovery" });
      if (!response?.granted) {
        $("#status-copy").textContent = "Permission was not granted. Discovery remains off.";
      }
    }
    await refresh();
  } finally {
    button.disabled = false;
  }
});

$("#clear").addEventListener("click", async () => {
  if (!Number.isInteger(state.tabId)) return;
  await chrome.runtime.sendMessage({ type: "clearStreams", tabId: state.tabId });
  await refresh();
});

void refresh();
