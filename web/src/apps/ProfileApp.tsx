import { useEffect, useRef, useState } from "react";
import { Loader2, LogOut, LogIn, ShieldCheck, Sparkles, Settings as SettingsIcon } from "lucide-react";
import { AppShell } from "./shell";
import { Tooltip } from "../components/Tooltip";
import { useProfile } from "../os/useProfile";
import { ProfileEditor } from "./ProfileEditor";
import { ProfileCard } from "../components/ProfileCard";

// ProfileApp is the account surface: sign in with Discord to unlock features
// (sync runs automatically in the background once signed in). No server URL to
// configure — the cloud endpoint is built into enowx. Sync controls live in
// Settings → Cloud Sync.
export function ProfileApp() {
  const profile = useProfile();
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const poll = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    return () => {
      if (poll.current) clearInterval(poll.current);
    };
  }, []);

  async function connect() {
    setError("");
    setBusy("Opening Discord…");
    try {
      const { authorize_url, state } = await profile.startLogin();
      window.open(authorize_url, "_blank", "noopener");
      setBusy("Waiting for Discord authorization…");
      poll.current = setInterval(async () => {
        try {
          const done = await profile.pollLogin(state);
          if (done) {
            if (poll.current) clearInterval(poll.current);
            setBusy("");
          }
        } catch {
          /* keep polling */
        }
      }, 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "couldn't reach the server");
      setBusy("");
    }
  }

  async function logout() {
    setBusy("Signing out…");
    try {
      await profile.logout();
    } finally {
      setBusy("");
    }
  }

  if (profile.loading) {
    return (
      <AppShell title="Profile" subtitle="Your enowx account">
        <div className="flex h-32 items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-white/40" />
        </div>
      </AppShell>
    );
  }

  return (
    <AppShell title="Profile" subtitle="Your enowx account">
      {error && <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">{error}</div>}

      {profile.loggedIn && profile.user ? (
        <div className="space-y-4">
          {/* Discord-style profile card (reused for public profiles too). */}
          <ProfileCard p={profile.user} />

          <div className="flex items-center justify-between gap-2">
            <ProfileEditor />
            <span className="text-[11px] text-white/35">via Discord</span>
          </div>

          {/* Account notes */}
          <div className="rounded-xl border border-white/10 bg-white/[0.02] p-3.5">
            <div className="mb-2 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-white/40">
              <Sparkles className="h-3 w-3" /> Account
            </div>
            <p className="text-xs leading-relaxed text-white/55">
              Your playlists sync automatically across signed-in devices.
            </p>
            {!profile.user.wears_tag && (
              <p className="mt-2 text-[11px] leading-relaxed text-white/40">
                Wear the <span className="font-semibold text-white/60">[enow]</span> server tag on Discord to unlock
                extra profile features.
              </p>
            )}
            <div className="mt-3 flex items-center gap-1.5 text-[11px] text-white/40">
              <SettingsIcon className="h-3 w-3" />
              Manage sync in Settings → Cloud Sync.
            </div>
          </div>

          <Tooltip label="Sign out of this device" place="bottom">
            <button
              onClick={logout}
              disabled={!!busy}
              className="flex items-center gap-1.5 rounded-lg border border-white/10 px-3 py-1.5 text-xs text-white/60 hover:bg-white/5 hover:text-white disabled:opacity-50"
            >
              <LogOut className="h-3.5 w-3.5" /> Sign out
            </button>
          </Tooltip>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-6 text-center">
          <div className="mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-500/30 to-violet-600/30">
            <ShieldCheck className="h-7 w-7 text-indigo-200" />
          </div>
          <h2 className="text-sm font-semibold text-white">Sign in to enowx</h2>
          <p className="mt-1 max-w-xs text-[11px] leading-relaxed text-white/50">
            Connect your Discord account to sync across devices and unlock account features. enowx works fine without
            signing in — login just adds more.
          </p>
          <Tooltip label="Sign in with Discord" place="bottom">
            <button
              onClick={connect}
              disabled={!!busy}
              className="mt-4 flex items-center gap-1.5 rounded-lg bg-[#5865F2] px-4 py-2 text-xs font-semibold text-white hover:opacity-90 disabled:opacity-50"
            >
              {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <LogIn className="h-3.5 w-3.5" />}
              Connect Discord
            </button>
          </Tooltip>
          {busy && <p className="mt-2 text-[11px] text-white/45">{busy}</p>}
        </div>
      )}
    </AppShell>
  );
}
