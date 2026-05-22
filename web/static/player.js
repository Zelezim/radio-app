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
  playBtn.addEventListener("click", () => {
    if (audio.paused) {
      // Reset the buffer so a long-paused user doesn't hear stale audio.
      audio.load();
      const p = audio.play();
      if (p && typeof p.then === "function") {
        setStatus("Connecting…", false);
        p.then(() => setStatus("Live", true)).catch((err) => {
          console.error("playback failed:", err);
          setStatus("Tap play to retry.", false);
        });
      }
    } else {
      audio.pause();
    }
  });

  audio.addEventListener("playing", () => {
    playBtn.classList.add("is-playing");
    setStatus("Live", true);
  });
  audio.addEventListener("pause", () => {
    playBtn.classList.remove("is-playing");
    setStatus("Paused", false);
  });
  audio.addEventListener("waiting", () => setStatus("Buffering…", false));
  audio.addEventListener("error", () => setStatus("Stream error. Tap play to retry.", false));

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
