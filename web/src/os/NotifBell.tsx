import { useState } from "react";
import { Bell, MessageSquare, ChevronUp, Smile, AtSign } from "lucide-react";
import { Popover } from "../components/Popover";
import { useProfile } from "./useProfile";
import { useNotifications, markNotificationsRead } from "./notifBus";
import { openProfile } from "./profileViewer";
import type { Notification } from "../lib/api";

const ICON: Record<string, typeof Bell> = { reply: MessageSquare, upvote: ChevronUp, reaction: Smile, mention: AtSign };
const VERB: Record<string, string> = { reply: "replied", upvote: "upvoted", reaction: "reacted", mention: "mentioned you" };

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
        <Popover onClose={() => setOpen(false)} anchor="right" className="top-6 w-72">
          <div className="max-h-80 overflow-auto rounded-xl border border-white/10 bg-[#0e1016] shadow-2xl">
            <div className="border-b border-white/5 px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-white/40">Notifications</div>
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

function NotifRow({ n, onClose }: { n: Notification; onClose: () => void }) {
  const Icon = ICON[n.type] ?? Bell;
  return (
    <button
      onClick={() => {
        if (n.actor_id) openProfile(n.actor_id);
        onClose();
      }}
      className={`flex w-full items-start gap-2 border-b border-white/5 px-3 py-2 text-left hover:bg-white/5 ${n.read ? "" : "bg-indigo-500/[0.06]"}`}
    >
      <Icon className="mt-0.5 h-3.5 w-3.5 shrink-0 text-white/50" />
      <div className="min-w-0 flex-1">
        <div className="text-xs text-white/80">
          <span className="font-semibold text-white">{n.actor_name || "Someone"}</span> {VERB[n.type] ?? "did something"}
        </div>
        {n.preview && <div className="truncate text-[11px] text-white/45">{n.preview}</div>}
        <div className="text-[10px] text-white/30">{new Date(n.created_at).toLocaleString([], { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}</div>
      </div>
    </button>
  );
}
