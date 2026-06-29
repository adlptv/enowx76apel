import { useEffect, useRef, useState } from "react";
import { RefreshCw, Loader2, LogOut, Cloud, CloudOff, Check } from "lucide-react";
import { AppShell } from "./shell";
import { Tooltip } from "../components/Tooltip";
import { usePersisted } from "../os/usePersisted";
import { syncApi, type SyncStatus } from "../lib/api";

const DEFAULT_SERVER = "http://localhost:8787";

export function SyncApp() {
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [server, setServer] = usePersisted("sync-server", DEFAULT_SERVER);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [result, setResult] = useState("");
  const poll = useRef<ReturnType<typeof setInterval> | null>(null);

  async function load() {
    try {
      setStatus(await syncApi.status());
    } catch (e) {
      setError(e instanceof Error ? e.message : "failed to load");
    }
  }

  useEffect(() => {
    load();
    return () => {
      if (poll.current) clearInterval(poll.current);
    };
  }, []);

  async function connect() {
    setError("");
    setBusy("Opening Discord…");
    try {
      const { authorize_url, state } = await syncApi.loginStart(server.trim());
      window.open(authorize_url, "_blank", "noopener");
      setBusy("Waiting for Discord authorization…");
      // Poll until the browser flow completes.
      poll.current = setInterval(async () => {
        try {
          const r = await syncApi.loginPoll(state);
          if (r.done) {
            if (poll.current) clearInterval(poll.current);
            setBusy("");
            await load();
          }
        } catch {
          /* keep polling */
        }
      }, 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "login failed");
      setBusy("");
    }
  }

  async function syncNow() {
    setError("");
    setResult("");
    setBusy("Syncing…");
    try {
      const r = await syncApi.now();
      setResult(`Pushed ${r.pushed}, pulled ${r.pulled}.`);
      setTimeout(() => setResult(""), 4000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "sync failed");
    } finally {
      setBusy("");
    }
  }

  async function logout() {
    await syncApi.logout();
    await load();
  }

  const connected = status?.enabled && status.user;

  return (
    <AppShell title="Sync" subtitle="Sign in with Discord to sync across devices">
      {error && <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">{error}</div>}

      {connected ? (
        <div className="space-y-3">
          <div className="flex items-center gap-3 rounded-xl border border-emerald-500/20 bg-emerald-500/[0.05] p-3.5">
            {status!.user!.avatar_url ? (
              <img src={status!.user!.avatar_url} alt="" className="h-10 w-10 rounded-full" />
            ) : (
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-white/10">
                <Cloud className="h-5 w-5 text-emerald-300" />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <div className="truncate text-sm font-semibold text-white">{status!.user!.username || "Connected"}</div>
              <div className="text-[11px] text-white/45">
                Synced · plan <span className="text-emerald-300">{status!.user!.plan}</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Tooltip label="Reconcile now" place="bottom">
              <button
                onClick={syncNow}
                disabled={!!busy}
                className="flex items-center gap-1.5 rounded-lg bg-white px-3 py-1.5 text-xs font-semibold text-black hover:opacity-90 disabled:opacity-50"
              >
                {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
                Sync now
              </button>
            </Tooltip>
            <Tooltip label="Disconnect this device" place="bottom">
              <button
                onClick={logout}
                className="flex items-center gap-1.5 rounded-lg border border-white/10 px-3 py-1.5 text-xs text-white/60 hover:bg-white/5 hover:text-white"
              >
                <LogOut className="h-3.5 w-3.5" /> Disconnect
              </button>
            </Tooltip>
            {result && (
              <span className="flex items-center gap-1 text-[11px] text-emerald-300">
                <Check className="h-3 w-3" /> {result}
              </span>
            )}
            {busy && !result && <span className="text-[11px] text-white/45">{busy}</span>}
          </div>

          <p className="text-[11px] leading-relaxed text-white/40">
            Playlists sync automatically across your signed-in devices. Sensitive data (accounts, keys) will sync
            end-to-end encrypted in a later step.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center gap-2 text-white/55">
            <CloudOff className="h-4 w-4" />
            <span className="text-xs">Not connected.</span>
          </div>
          <label className="block">
            <span className="mb-1 block text-[11px] font-medium text-white/50">Cloud server</span>
            <input
              value={server}
              onChange={(e) => setServer(e.target.value)}
              placeholder={DEFAULT_SERVER}
              className="h-9 w-full rounded-lg border border-white/10 bg-white/5 px-3 font-mono text-xs text-white placeholder:text-white/25 focus:border-white/25 focus:outline-none"
            />
          </label>
          <Tooltip label="Sign in with Discord" place="bottom">
            <button
              onClick={connect}
              disabled={!!busy}
              className="flex items-center gap-1.5 rounded-lg bg-[#5865F2] px-3 py-1.5 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50"
            >
              {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Cloud className="h-3.5 w-3.5" />}
              Connect Discord
            </button>
          </Tooltip>
          {busy && <p className="text-[11px] text-white/45">{busy}</p>}
        </div>
      )}
    </AppShell>
  );
}
