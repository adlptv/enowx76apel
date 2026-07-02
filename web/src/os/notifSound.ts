// A short, pleasant two-note notification chime synthesized with the Web Audio
// API — no asset file needed. Respects a persisted mute toggle.

const MUTE_KEY = "notif-sound-muted";

export function isNotifSoundMuted(): boolean {
  return localStorage.getItem(MUTE_KEY) === "1";
}

export function setNotifSoundMuted(muted: boolean) {
  localStorage.setItem(MUTE_KEY, muted ? "1" : "0");
}

let ctx: AudioContext | null = null;

// playNotifSound plays a soft "ti-dan" chime (two quick descending notes).
export function playNotifSound() {
  if (isNotifSoundMuted()) return;
  try {
    const AC = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
    if (!ctx) ctx = new AC();
    // Browsers suspend the context until a user gesture; resume best-effort.
    if (ctx.state === "suspended") void ctx.resume();

    const now = ctx.currentTime;
    const master = ctx.createGain();
    master.gain.value = 0.14;
    master.connect(ctx.destination);

    // Two notes: G5 then C6, each a short bell-like blip.
    const notes = [
      { freq: 784, at: 0, dur: 0.12 },
      { freq: 1047, at: 0.1, dur: 0.16 },
    ];
    for (const n of notes) {
      const osc = ctx.createOscillator();
      const g = ctx.createGain();
      osc.type = "sine";
      osc.frequency.value = n.freq;
      g.gain.setValueAtTime(0.0001, now + n.at);
      g.gain.exponentialRampToValueAtTime(1, now + n.at + 0.012);
      g.gain.exponentialRampToValueAtTime(0.0001, now + n.at + n.dur);
      osc.connect(g);
      g.connect(master);
      osc.start(now + n.at);
      osc.stop(now + n.at + n.dur + 0.02);
    }
  } catch {
    /* audio unavailable — silently skip */
  }
}
