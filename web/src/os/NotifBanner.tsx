import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import type { Notification } from "../lib/api";
import { onBanner } from "./notifBus";
import { NOTIF_ICON, NOTIF_VERB, routeNotif } from "./notifMeta";
import { playNotifSound } from "./notifSound";
import { useProfile } from "./useProfile";

const DISMISS_MS = 5000;
const MAX_VISIBLE = 3;

type Banner = { key: number; n: Notification };

// NotifBanner renders macOS-style notification cards that slide in at the top
// right, auto-dismiss after ~5s (paused while hovered), and stack. Mounted once
// at the desktop root. Login-gated.
export function NotifBanner() {
  const profile = useProfile();
  const [banners, setBanners] = useState<Banner[]>([]);
  const seq = useRef(0);

  useEffect(() => {
    return onBanner((n) => {
      const key = seq.current++;
      setBanners((prev) => [...prev, { key, n }].slice(-MAX_VISIBLE));
      playNotifSound();
    });
  }, []);

  if (!profile.loggedIn) return null;

  const dismiss = (key: number) => setBanners((prev) => prev.filter((b) => b.key !== key));

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-[9999] flex w-80 flex-col gap-2">
      {banners.map((b) => (
        <BannerCard key={b.key} n={b.n} onDismiss={() => dismiss(b.key)} />
      ))}
    </div>
  );
}

function BannerCard({ n, onDismiss }: { n: Notification; onDismiss: () => void }) {
  const [shown, setShown] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const hovering = useRef(false);
  const Icon = NOTIF_ICON[n.type] ?? NOTIF_ICON.reply;

  const close = () => {
    setLeaving(true);
    setTimeout(onDismiss, 200); // let the exit transition play
  };

  useEffect(() => {
    // Slide in on mount.
    const t = requestAnimationFrame(() => setShown(true));
    // Auto-dismiss unless hovered; re-check near the deadline.
    const timer = setTimeout(() => {
      if (!hovering.current) close();
    }, DISMISS_MS);
    return () => {
      cancelAnimationFrame(t);
      clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <button
      onMouseEnter={() => (hovering.current = true)}
      onMouseLeave={() => {
        hovering.current = false;
        setTimeout(() => !hovering.current && close(), 1500);
      }}
      onClick={() => {
        routeNotif(n);
        close();
      }}
      className={`pointer-events-auto group flex w-full items-start gap-2.5 rounded-2xl border border-white/10 bg-[#16181f]/85 p-3 text-left shadow-2xl backdrop-blur-xl transition-all duration-200 ${
        shown && !leaving ? "translate-x-0 opacity-100" : "translate-x-6 opacity-0"
      }`}
    >
      <div className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-white/10 text-white/80">
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="text-xs text-white/85">
          <span className="font-semibold text-white">{n.actor_name || "Someone"}</span> {NOTIF_VERB[n.type] ?? "notified you"}
        </div>
        {n.preview && <div className="mt-0.5 truncate text-[11px] text-white/50">{n.preview}</div>}
      </div>
      <span
        role="button"
        onClick={(e) => {
          e.stopPropagation();
          close();
        }}
        className="rounded p-0.5 text-white/30 opacity-0 transition-opacity hover:bg-white/10 hover:text-white group-hover:opacity-100"
      >
        <X className="h-3.5 w-3.5" />
      </span>
    </button>
  );
}
