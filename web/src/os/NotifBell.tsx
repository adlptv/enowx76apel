import { useState } from "react";
import { Bell, Volume2, VolumeX } from "lucide-react";
import { Popover } from "../components/Popover";
import { useProfile } from "./useProfile";
import { useNotifications, markNotificationsRead } from "./notifBus";
import { NOTIF_ICON, NOTIF_VERB, routeNotif } from "./notifMeta";
import { isNotifSoundMuted, setNotifSoundMuted, playNotifSound } from "./notifSound";
import type { Notification } from "../lib/api";

// NotifBell is the top-bar notifications bell + dropdown. Login-gated.
export function NotifBell() {
  const profile = useProfile();
  const { items, unread } = useNotifications();
  const [open, setOpen] = useState(false);

  if (!profile.loggedIn) return null;

  function toggle() {
    setOpen((v) => {
      const next = !v;
      if (next && unread > 0) markNotificationsRead();
      return next;
    });
  }

  return (
    <div className="relative">
      <button onClick={toggle} className="relative flex items-center rounded p-0.5 text-white/70 hover:text-white" title="Notifications">
        <Bell className="h-3.5 w-3.5" />
        {unread > 0 && (
          <span className="absolute -right-1 -top-1 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-rose-500 px-1 text-[9px] font-bold text-white">
            {unread > 9 ? "9+" : unread}
          </span>
        )}
      </button>
      {open && (
        <Popover onClose={() => setOpen(false)} anchor="right" valign="down" className="w-72">
          <div className="max-h-80 overflow-auto rounded-xl border border-white/10 bg-[#0e1016] shadow-2xl">
            <div className="flex items-center justify-between border-b border-white/5 px-3 py-2">
              <span className="text-[11px] font-semibold uppercase tracking-wide text-white/40">Notifications</span>
              <SoundToggle />
            </div>
            {items.length === 0 ? (
              <div className="px-3 py-6 text-center text-xs text-white/40">Nothing yet.</div>
            ) : (
              items.map((n) => <NotifRow key={n.id} n={n} onClose={() => setOpen(false)} />)
            )}
          </div>
        </Popover>
      )}
    </div>
  );
}

// SoundToggle mutes/unmutes the notification chime (plays a preview on unmute).
function SoundToggle() {
  const [muted, setMuted] = useState(isNotifSoundMuted());
  return (
    <button
      onClick={() => {
        const next = !muted;
        setNotifSoundMuted(next);
        setMuted(next);
        if (!next) playNotifSound(); // preview when turning on
      }}
      title={muted ? "Notification sound off" : "Notification sound on"}
      className="rounded p-1 text-white/40 hover:bg-white/10 hover:text-white"
    >
      {muted ? <VolumeX className="h-3.5 w-3.5" /> : <Volume2 className="h-3.5 w-3.5" />}
    </button>
  );
}

function NotifRow({ n, onClose }: { n: Notification; onClose: () => void }) {
  const Icon = NOTIF_ICON[n.type] ?? Bell;
  return (
    <button
      onClick={() => {
        routeNotif(n);
        onClose();
      }}
      className={`flex w-full items-start gap-2 border-b border-white/5 px-3 py-2 text-left hover:bg-white/5 ${n.read ? "" : "bg-indigo-500/[0.06]"}`}
    >
      <Icon className="mt-0.5 h-3.5 w-3.5 shrink-0 text-white/50" />
      <div className="min-w-0 flex-1">
        <div className="text-xs text-white/80">
          <span className="font-semibold text-white">{n.actor_name || "Someone"}</span> {NOTIF_VERB[n.type] ?? "did something"}
        </div>
        {n.preview && <div className="truncate text-[11px] text-white/45">{n.preview}</div>}
        <div className="text-[10px] text-white/30">{new Date(n.created_at).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}</div>
      </div>
    </button>
  );
}
