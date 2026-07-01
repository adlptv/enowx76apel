import { useEffect, useRef, useState } from "react";
import { X, ExternalLink, Loader2 } from "lucide-react";
import { ProviderIcon } from "./ProviderIcon";
import { qwenApi, type Provider } from "../lib/api";

// QwenAddModal runs the Qwen device-code login: show the user code + link,
// open the browser, then auto-poll until the account is authorized.
export function QwenAddModal({ provider, onClose, onSaved }: { provider: Provider; onClose: () => void; onSaved: () => void }) {
  const [start, setStart] = useState<{ session: string; user_code: string; verification_uri: string; verification_uri_complete: string; interval: number } | null>(null);
  const [status, setStatus] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const poll = useRef<number | null>(null);

  useEffect(() => () => { if (poll.current) window.clearInterval(poll.current); }, []);

  const begin = async () => {
    setErr("");
    setBusy(true);
    try {
      const s = await qwenApi.deviceStart();
      setStart(s);
      setStatus("Waiting for authorization…");
      window.open(s.verification_uri_complete || s.verification_uri, "_blank", "noreferrer");
      poll.current = window.setInterval(async () => {
        try {
          const r = await qwenApi.devicePoll(s.session);
          if (r.status === "done") {
            if (poll.current) window.clearInterval(poll.current);
            onSaved();
          }
        } catch (e) {
          if (poll.current) window.clearInterval(poll.current);
          setErr(e instanceof Error ? e.message : "poll failed");
        }
      }, Math.max(2, s.interval) * 1000);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm" onClick={onClose}>
      <div className="flex w-full max-w-md flex-col overflow-hidden rounded-2xl border border-white/10 bg-[#11131a] shadow-2xl" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-3 border-b border-white/5 px-4 py-3">
          <ProviderIcon icon={provider.icon} label={provider.label} size={32} />
          <div className="flex-1">
            <p className="text-sm font-semibold text-white">Add Qwen account</p>
            <p className="text-[11px] text-white/40">Sign in with a Qwen account. Stored locally.</p>
          </div>
          <button onClick={onClose} className="rounded-md p-1 text-white/40 hover:bg-white/10 hover:text-white"><X className="h-4 w-4" /></button>
        </div>
        <div className="p-4">
          {!start ? (
            <div className="space-y-3">
              <p className="text-xs text-white/50">A browser tab opens to authorize. Approve there and this finishes automatically.</p>
              {err && <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">{err}</div>}
              <button onClick={begin} disabled={busy} className="w-full rounded-lg bg-white px-4 py-2 text-sm font-medium text-black hover:opacity-90 disabled:opacity-50">
                {busy ? "Starting…" : "Start Qwen login"}
              </button>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-xs text-white/50">Enter this code on the Qwen page and approve.</p>
              <div className="rounded-lg border border-white/10 bg-black/30 p-3 text-center">
                <p className="text-[11px] uppercase tracking-wide text-white/40">User code</p>
                <p className="my-1 font-mono text-2xl font-bold tracking-widest text-emerald-300">{start.user_code}</p>
              </div>
              <a href={start.verification_uri_complete || start.verification_uri} target="_blank" rel="noreferrer" className="flex items-center justify-center gap-1.5 rounded-lg bg-white px-4 py-2 text-sm font-medium text-black hover:opacity-90">
                Open Qwen verification <ExternalLink className="h-3.5 w-3.5" />
              </a>
              <div className="flex items-center justify-center gap-2 font-mono text-[11px] text-white/40">
                <Loader2 className="h-3.5 w-3.5 animate-spin" /> {status}
              </div>
              {err && <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">{err}</div>}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
