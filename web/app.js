const setupPanel = document.getElementById("setup-panel");
const gamePanel = document.getElementById("game-panel");
const wordbookSelect = document.getElementById("wordbook-select");
const speedRange = document.getElementById("speed-range");
const speedValue = document.getElementById("speed-value");
const accentSelect = document.getElementById("accent-select");
const maxWordsInput = document.getElementById("max-words-input");
const startBtn = document.getElementById("start-btn");
const setupError = document.getElementById("setup-error");
const scoreEl = document.getElementById("score");
const comboEl = document.getElementById("combo");
const statusEl = document.getElementById("status");
const backBtn = document.getElementById("back-btn");
const resultBox = document.getElementById("result-box");
const finalScoreEl = document.getElementById("final-score");
const restartBtn = document.getElementById("restart-btn");

const canvas = document.getElementById("game-canvas");
const ctx = canvas.getContext("2d");

const state = {
  running: false,
  frozen: false,
  gameOver: false,
  score: 0,
  combo: 0,
  inputBuffer: "",
  targetWordId: null,
  speedLevel: 1,
  pendingWords: [],
  activeWords: [],
  missedWords: [],
  spawnTimer: 0,
  effects: [],
  nextWordId: 1,
  lastFrameTime: 0,
  maxActiveWords: 8,
  accent: "en-US",
};

const WORLD = {
  topY: 26,
  groundY: canvas.height - 42,
  sidePadding: 18,
};

function shuffle(list) {
  const arr = [...list];
  for (let i = arr.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    [arr[i], arr[j]] = [arr[j], arr[i]];
  }
  return arr;
}

function speedToFallSeconds(level) {
  const minSeconds = 3;
  const maxSeconds = 10;
  const t = (level - 1) / 9;
  const curved = Math.pow(t, 1.6);
  return maxSeconds - (maxSeconds - minSeconds) * curved;
}

function speedToPixelsPerSecond(level) {
  const seconds = speedToFallSeconds(level);
  const distance = WORLD.groundY - WORLD.topY;
  return distance / seconds;
}

async function fetchWordbooks() {
  const res = await fetch("/api/wordbooks");
  if (!res.ok) {
    throw new Error("Failed to load wordbook list.");
  }
  const data = await res.json();
  return data.wordbooks || [];
}

async function fetchWordbookWords(name) {
  const res = await fetch(`/api/wordbooks/${encodeURIComponent(name)}`);
  if (!res.ok) {
    throw new Error("Failed to load wordbook words.");
  }
  const data = await res.json();
  return data.words || [];
}

async function fetchSettings() {
  const res = await fetch("/api/settings");
  if (!res.ok) {
    throw new Error("Failed to load settings.");
  }
  return res.json();
}

async function saveAccent(accent) {
  const res = await fetch("/api/settings/accent", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ accent }),
  });
  if (!res.ok) {
    throw new Error("Failed to save accent setting.");
  }
  return res.json();
}

async function saveWordbook(wordbook) {
  const res = await fetch("/api/settings/wordbook", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ wordbook }),
  });
  if (!res.ok) {
    throw new Error("Failed to save wordbook setting.");
  }
  return res.json();
}

function resetRuntimeState() {
  state.running = true;
  state.frozen = false;
  state.gameOver = false;
  state.score = 0;
  state.combo = 0;
  state.inputBuffer = "";
  state.targetWordId = null;
  state.spawnTimer = 0;
  state.activeWords = [];
  state.missedWords = [];
  state.effects = [];
  state.nextWordId = 1;
  state.lastFrameTime = 0;

  updateHUD();
  statusEl.textContent = "Running";
  resultBox.classList.add("hidden");
}

function parseMaxWordsLimit(rawValue) {
  const text = String(rawValue || "").trim();
  if (text === "") {
    return 0;
  }
  const n = Number(text);
  if (!Number.isFinite(n)) {
    return 0;
  }
  return Math.max(0, Math.floor(n));
}

function updateHUD() {
  scoreEl.textContent = String(state.score);
  comboEl.textContent = String(state.combo);
}

function chooseSpawnX(text) {
  ctx.font = "bold 30px Trebuchet MS";
  const width = Math.max(40, ctx.measureText(text).width + 16);
  const minX = WORLD.sidePadding;
  const maxX = canvas.width - WORLD.sidePadding - width;
  const x = minX + Math.random() * Math.max(1, maxX - minX);
  return { x, width };
}

function spawnWord() {
  if (state.activeWords.length >= state.maxActiveWords) {
    return false;
  }

  if (state.pendingWords.length === 0) {
    if (state.missedWords.length === 0) {
      return false;
    }
    state.pendingWords = shuffle(state.missedWords);
    state.missedWords = [];
  }

  if (state.pendingWords.length === 0) {
    return false;
  }

  const text = state.pendingWords.shift();
  const { x, width } = chooseSpawnX(text);

  state.activeWords.push({
    id: state.nextWordId,
    text,
    x,
    y: WORLD.topY,
    width,
  });
  state.nextWordId += 1;
  return true;
}

function chooseTarget(prefix) {
  let best = null;
  for (const w of state.activeWords) {
    if (!w.text.startsWith(prefix)) {
      continue;
    }
    if (!best || w.y > best.y) {
      best = w;
    }
  }
  return best;
}

function clearInputTracking() {
  state.inputBuffer = "";
  state.targetWordId = null;
}

function onWordMissed(word) {
  state.missedWords.push(word.text);
  state.combo = 0;
  updateHUD();

  if (state.targetWordId === word.id) {
    clearInputTracking();
  }
}

function removeActiveWordById(id) {
  const index = state.activeWords.findIndex((w) => w.id === id);
  if (index < 0) {
    return null;
  }
  const [removed] = state.activeWords.splice(index, 1);
  return removed;
}

function animateDissolve(word, durationMs) {
  const startedAt = performance.now();
  state.effects.push({
    kind: "dissolve",
    text: word.text,
    x: word.x,
    y: word.y,
    durationMs,
    startedAt,
  });

  return new Promise((resolve) => {
    setTimeout(resolve, durationMs);
  });
}

function animateScoreGain(word, points, durationMs) {
  const startedAt = performance.now();
  state.effects.push({
    kind: "score",
    text: `+${points}`,
    x: word.x + word.width + 12,
    y: word.y - 4,
    durationMs,
    startedAt,
  });
}

function pickVoiceForLang(lang) {
  if (!("speechSynthesis" in window)) {
    return null;
  }
  const voices = window.speechSynthesis.getVoices() || [];
  return voices.find((v) => v.lang === lang) || voices.find((v) => v.lang.startsWith(lang));
}

function speakWord(text) {
  return new Promise((resolve) => {
    if (!("speechSynthesis" in window)) {
      setTimeout(resolve, 500);
      return;
    }

    window.speechSynthesis.cancel();

    const utterance = new SpeechSynthesisUtterance(text);
    utterance.rate = 0.92;
    utterance.pitch = 1.0;
    utterance.lang = state.accent;
    const voice = pickVoiceForLang(state.accent);
    if (voice) {
      utterance.voice = voice;
    }

    let settled = false;
    const done = () => {
      if (settled) {
        return;
      }
      settled = true;
      resolve();
    };

    utterance.onend = done;
    utterance.onerror = done;

    window.speechSynthesis.speak(utterance);

    // Fallback in case browser events are blocked.
    setTimeout(done, 2500);
  });
}

async function onWordSolved(word) {
  state.combo += 1;
  const points = state.combo;
  state.score += points;
  updateHUD();
  animateScoreGain(word, points, 850);

  state.frozen = true;
  statusEl.textContent = "Speaking";

  await Promise.all([animateDissolve(word, 600), speakWord(word.text)]);

  state.frozen = false;
  statusEl.textContent = "Running";
  maybeFinishGame();
}

function maybeFinishGame() {
  if (
    state.pendingWords.length === 0 &&
    state.activeWords.length === 0 &&
    state.missedWords.length === 0
  ) {
    state.running = false;
    state.gameOver = true;
    statusEl.textContent = "Complete";
    finalScoreEl.textContent = String(state.score);
    resultBox.classList.remove("hidden");
  }
}

function handleTypedCharacter(ch) {
  if (!state.running || state.frozen || state.gameOver) {
    return;
  }

  const lower = ch.toLowerCase();
  const hadInput = state.inputBuffer.length > 0;
  let nextBuffer = state.inputBuffer + lower;
  let target = chooseTarget(nextBuffer);
  let brokenChain = false;

  if (!target) {
    if (hadInput) {
      brokenChain = true;
    }
    nextBuffer = lower;
    target = chooseTarget(nextBuffer);
  }

  if (brokenChain) {
    state.combo = 0;
    updateHUD();
  }

  if (!target) {
    clearInputTracking();
    return;
  }

  state.inputBuffer = nextBuffer;
  state.targetWordId = target.id;

  if (state.inputBuffer === target.text) {
    const solved = removeActiveWordById(target.id);
    clearInputTracking();
    if (solved) {
      void onWordSolved(solved);
    }
  }
}

function handleControlKey(key) {
  if (key === "Backspace") {
    if (state.inputBuffer.length === 0) {
      return;
    }
    state.inputBuffer = state.inputBuffer.slice(0, -1);
    if (state.inputBuffer.length === 0) {
      state.targetWordId = null;
      return;
    }
    const target = chooseTarget(state.inputBuffer);
    state.targetWordId = target ? target.id : null;
    return;
  }

  if (key === "Escape") {
    clearInputTracking();
  }
}

function updateEffects(now) {
  state.effects = state.effects.filter((effect) => now - effect.startedAt < effect.durationMs);
}

function updateWorld(dt) {
  state.spawnTimer += dt;
  while (state.spawnTimer >= 1) {
    state.spawnTimer -= 1;
    spawnWord();
  }

  const speed = speedToPixelsPerSecond(state.speedLevel);
  const survivors = [];

  for (const word of state.activeWords) {
    word.y += speed * dt;
    if (word.y >= WORLD.groundY) {
      onWordMissed(word);
      continue;
    }
    survivors.push(word);
  }

  state.activeWords = survivors;
  maybeFinishGame();
}

function drawWord(word) {
  const isTarget = word.id === state.targetWordId;
  const typedPrefix = isTarget ? state.inputBuffer : "";

  ctx.font = "bold 30px Trebuchet MS";
  ctx.textBaseline = "top";

  if (typedPrefix && word.text.startsWith(typedPrefix)) {
    const done = typedPrefix;
    const rest = word.text.slice(done.length);

    ctx.fillStyle = "#1b7f79";
    ctx.fillText(done, word.x, word.y);
    const doneWidth = ctx.measureText(done).width;

    ctx.fillStyle = "#1d2d3d";
    ctx.fillText(rest, word.x + doneWidth, word.y);
  } else {
    ctx.fillStyle = isTarget ? "#d95d39" : "#1d2d3d";
    ctx.fillText(word.text, word.x, word.y);
  }
}

function drawEffects(now) {
  for (const effect of state.effects) {
    const elapsed = now - effect.startedAt;
    const t = Math.min(1, elapsed / effect.durationMs);
    const alpha = 1 - t;

    ctx.save();
    ctx.globalAlpha = alpha;
    if (effect.kind === "score") {
      const drift = t * 22;
      ctx.fillStyle = "#d95d39";
      ctx.font = "bold 24px Trebuchet MS";
      ctx.fillText(effect.text, effect.x, effect.y - drift);
    } else {
      const drift = t * 26;
      ctx.fillStyle = "#2a9d8f";
      ctx.font = "bold 30px Trebuchet MS";
      ctx.fillText(effect.text, effect.x, effect.y - drift);
    }
    ctx.restore();
  }
}

function render(now) {
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  const inputZoneTop = canvas.height - 44;
  ctx.fillStyle = "rgba(255, 255, 255, 0.72)";
  ctx.fillRect(0, inputZoneTop, canvas.width, canvas.height - inputZoneTop);
  ctx.fillStyle = "rgba(7, 38, 70, 0.24)";
  ctx.fillRect(0, inputZoneTop, canvas.width, 2);

  for (const word of state.activeWords) {
    drawWord(word);
  }

  drawEffects(now);

  ctx.fillStyle = "rgba(19, 42, 59, 0.8)";
  ctx.font = "16px Verdana";
  ctx.fillText(`Input: ${state.inputBuffer}`, 14, canvas.height - 26);
}

function loop(now) {
  if (state.lastFrameTime === 0) {
    state.lastFrameTime = now;
  }
  const dt = Math.min(0.1, (now - state.lastFrameTime) / 1000);
  state.lastFrameTime = now;

  updateEffects(now);

  if (state.running && !state.frozen && !state.gameOver) {
    updateWorld(dt);
  }

  render(now);
  requestAnimationFrame(loop);
}

async function startGame() {
  setupError.textContent = "";

  const selected = wordbookSelect.value;
  if (!selected) {
    setupError.textContent = "Please choose a wordbook.";
    return;
  }

  let words;
  try {
    words = await fetchWordbookWords(selected);
  } catch (err) {
    setupError.textContent = err.message;
    return;
  }

  if (!words.length) {
    setupError.textContent = "Selected wordbook is empty.";
    return;
  }

  resetRuntimeState();
  state.speedLevel = Number(speedRange.value);
  state.accent = accentSelect.value;
  const normalizedWords = words.map((w) => String(w).toLowerCase().trim()).filter(Boolean);
  const maxWords = parseMaxWordsLimit(maxWordsInput.value);
  const selectedWords =
    maxWords > 0 ? shuffle(normalizedWords).slice(0, Math.min(maxWords, normalizedWords.length)) : normalizedWords;
  state.pendingWords = shuffle(selectedWords);

  setupPanel.classList.add("hidden");
  gamePanel.classList.remove("hidden");

  // Spawn one word immediately so the screen never starts empty.
  spawnWord();
}

function goToSetup() {
  state.running = false;
  state.frozen = false;
  state.gameOver = false;
  clearInputTracking();

  gamePanel.classList.add("hidden");
  setupPanel.classList.remove("hidden");
}

async function init() {
  speedValue.textContent = speedRange.value;

  let books;
  try {
    books = await fetchWordbooks();
  } catch (err) {
    setupError.textContent = err.message;
    startBtn.disabled = true;
    return;
  }

  if (books.length === 0) {
    setupError.textContent = "No .txt wordbooks found in the configured folder.";
    startBtn.disabled = true;
    return;
  }

  wordbookSelect.innerHTML = "";
  for (const name of books) {
    const option = document.createElement("option");
    option.value = name;
    option.textContent = name;
    wordbookSelect.appendChild(option);
  }

  try {
    const settings = await fetchSettings();
    if (settings && (settings.accent === "en-US" || settings.accent === "en-GB")) {
      accentSelect.value = settings.accent;
    }
    if (settings && typeof settings.wordbook === "string" && settings.wordbook.trim() !== "") {
      const saved = settings.wordbook.trim();
      if (books.includes(saved)) {
        wordbookSelect.value = saved;
      } else {
        wordbookSelect.value = books[0];
        void saveWordbook(books[0]).catch(() => {});
      }
    }
  } catch (_err) {
    // Keep default accent if settings are unavailable.
  }

  startBtn.addEventListener("click", () => {
    void startGame();
  });

  restartBtn.addEventListener("click", goToSetup);
  backBtn.addEventListener("click", goToSetup);

  speedRange.addEventListener("input", () => {
    speedValue.textContent = speedRange.value;
  });

  accentSelect.addEventListener("change", () => {
    void saveAccent(accentSelect.value).catch(() => {
      setupError.textContent = "Failed to save accent preference.";
    });
  });

  wordbookSelect.addEventListener("change", () => {
    void saveWordbook(wordbookSelect.value).catch(() => {
      setupError.textContent = "Failed to save wordbook preference.";
    });
  });

  document.addEventListener("keydown", (event) => {
    if (!state.running) {
      return;
    }

    if (/^[a-zA-Z]$/.test(event.key)) {
      event.preventDefault();
      handleTypedCharacter(event.key);
      return;
    }

    if (event.key === "Backspace" || event.key === "Escape") {
      event.preventDefault();
      handleControlKey(event.key);
    }
  });

  requestAnimationFrame(loop);
}

void init();
