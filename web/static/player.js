// Persistent player + dock controller.
//
// The dock element lives in base.html so it is never re-rendered when the
// user navigates with hx-boost. That keeps the <audio> element alive and
// playback uninterrupted across page swaps.

(function () {
  const audio = document.getElementById("radio");
  const playBtn = document.getElementById("playBtn");
  const vol = document.getElementById("vol");
  const status = document.getElementById("status");
  const dock = document.getElementById("dock");
  const dockToggle = document.getElementById("dockToggle");
  const dockClose = document.getElementById("dockClose");

  if (!audio || !playBtn) return;

  // --- Volume ---------------------------------------------------------
  const savedVol = parseFloat(localStorage.getItem("radioapp.volume"));
  if (!Number.isNaN(savedVol)) {
    audio.volume = savedVol;
    if (vol) vol.value = savedVol;
  } else {
    audio.volume = 0.8;
  }
  if (vol) {
    vol.addEventListener("input", () => {
      audio.volume = parseFloat(vol.value);
      localStorage.setItem("radioapp.volume", vol.value);
    });
  }

  function setStatus(text, live) {
    if (status) status.textContent = text;
    dock.classList.toggle("is-live", !!live);
  }

  // --- Play / pause ---------------------------------------------------
  //
  // Design note: we DO NOT call audio.load() before play(). For live
  // Icecast streams served with Transfer-Encoding: chunked, forcing a
  // reload resets the pipeline and can trap Chrome in a loop where it
  // fires `waiting`, then `pause`, then never reconnects — exactly the
  // "tries to buffer then pauses" symptom. Just calling play() lets the
  // browser's media stack reuse any existing connection or open a fresh
  // one as needed. If the user is coming back after a long pause, the
  // browser handles the stale buffer internally.

  let bufferTimer = null;
  const clearBufferTimer = () => {
    if (bufferTimer) { clearTimeout(bufferTimer); bufferTimer = null; }
  };

  playBtn.addEventListener("click", () => {
    if (audio.paused) {
      setStatus("Connecting…", false);
      const p = audio.play();
      if (p && typeof p.then === "function") {
        p.catch((err) => {
          console.error("playback failed:", err);
          setStatus("Tap play to retry.", false);
        });
      }
    } else {
      audio.pause();
    }
  });

  audio.addEventListener("playing", () => {
    clearBufferTimer();
    playBtn.classList.add("is-playing");
    setStatus("Live", true);
  });
  audio.addEventListener("pause", () => {
    clearBufferTimer();
    playBtn.classList.remove("is-playing");
    setStatus("Paused", false);
  });
  audio.addEventListener("waiting", () => {
    setStatus("Buffering…", false);
    // If buffering drags on past 8 seconds, surface a hint instead of
    // leaving the user staring at a frozen "Buffering…" label.
    clearBufferTimer();
    bufferTimer = setTimeout(() => {
      if (audio.readyState < 3) {
        setStatus("Stream slow — tap play to retry.", false);
      }
    }, 8000);
  });
  audio.addEventListener("error", () => {
    clearBufferTimer();
    const err = audio.error;
    console.error("audio error", err && err.code, err && err.message);
    setStatus("Stream error. Tap play to retry.", false);
  });
  audio.addEventListener("stalled", () => console.warn("audio stalled"));

  // --- Contact dock toggle -------------------------------------------
  function setDockOpen(open) {
    dock.dataset.expanded = open ? "true" : "false";
    if (dockToggle) dockToggle.setAttribute("aria-expanded", open ? "true" : "false");
    const panel = document.getElementById("dockPanel");
    if (panel) panel.setAttribute("aria-hidden", open ? "false" : "true");
  }

  if (dockToggle) {
    dockToggle.addEventListener("click", () => {
      setDockOpen(dock.dataset.expanded !== "true");
    });
  }
  if (dockClose) {
    dockClose.addEventListener("click", () => setDockOpen(false));
  }

  // Any in-page button can request the dock to open by carrying
  // data-trigger="dock-toggle" (e.g. the hero CTA).
  document.addEventListener("click", (e) => {
    const t = e.target.closest('[data-trigger="dock-toggle"]');
    if (t) {
      e.preventDefault();
      setDockOpen(true);
    }
  });

  // Auto-close after a successful contact submission, with a small delay
  // so the user sees the success message first.
  document.body.addEventListener("htmx:afterRequest", (evt) => {
    const form = evt.target.closest && evt.target.closest("#contactForm");
    if (form && evt.detail.successful) {
      setTimeout(() => setDockOpen(false), 1800);
    }
  });

  // --- Media Session (OS lock screen / car HUD) ----------------------
  if ("mediaSession" in navigator) {
    navigator.mediaSession.metadata = new MediaMetadata({
      title: "Live broadcast",
      artist: document.querySelector(".dock-title")?.textContent || "RadioApp",
      album: "RadioApp",
    });
    navigator.mediaSession.setActionHandler("play",  () => playBtn.click());
    navigator.mediaSession.setActionHandler("pause", () => playBtn.click());
  }
})();
