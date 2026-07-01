import { Music2, ListMusic } from "lucide-react";
import type { MusicShare } from "../lib/api";

// MusicCard renders a shared track/playlist in the music channel. Clicking opens
// the item in the Music app (via onOpen).
export function MusicCard({ m, onOpen }: { m: MusicShare; onOpen?: () => void }) {
  const Icon = m.kind === "playlist" ? ListMusic : Music2;
  return (
    <button
      onClick={onOpen}
      className="mt-1 flex w-full max-w-sm items-center gap-2.5 rounded-lg border border-white/10 bg-white/[0.03] p-2 text-left hover:border-white/25 hover:bg-white/[0.06]"
    >
      {m.cover ? (
        <img src={m.cover} alt="" className="h-11 w-11 shrink-0 rounded object-cover" />
      ) : (
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded bg-indigo-500/20">
          <Icon className="h-5 w-5 text-indigo-300" />
        </div>
      )}
      <div className="min-w-0">
        <div className="flex items-center gap-1 text-[10px] uppercase tracking-wide text-white/40">
          <Icon className="h-3 w-3" /> {m.kind}
        </div>
        <div className="truncate text-sm font-medium text-white">{m.title}</div>
        {m.subtitle && <div className="truncate text-[11px] text-white/45">{m.subtitle}</div>}
      </div>
    </button>
  );
}
