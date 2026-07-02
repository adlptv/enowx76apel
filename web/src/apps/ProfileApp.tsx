import { useEffect, useRef, useState } from "react";
import { Loader2, LogOut, LogIn, ShieldCheck, Sparkles, Crown, Check, Gift, X, Search } from "lucide-react";
import { createPortal } from "react-dom";
import { subscriptionApi, type SubscriptionStatus, type CouponPreview, type UserHit } from "../lib/api";
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
          {/* Discord-style profile card with the Edit action in the banner corner. */}
          <ProfileCard p={profile.user} action={<ProfileEditor />} />

          {/* Subscription / upgrade */}
          <SubscriptionCard />

          {/* Sign out — full-width at the bottom, matching the card frame. */}
          <button
            onClick={logout}
            disabled={!!busy}
            title="Sign out of this device"
            className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-white/10 px-3 py-2 text-xs text-white/60 hover:bg-white/5 hover:text-white disabled:opacity-50"
          >
            <LogOut className="h-3.5 w-3.5" /> Sign out
          </button>
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

// SubscriptionCard shows Premium status + upgrade/renew, replacing the old
// cloud-sync note. Free users see an upgrade CTA; premium users see their expiry.
function SubscriptionCard() {
  const [sub, setSub] = useState<SubscriptionStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [coupon, setCoupon] = useState("");
  const [preview, setPreview] = useState<CouponPreview | null>(null);
  const [checking, setChecking] = useState(false);

  const load = () => subscriptionApi.status().then(setSub).catch(() => setSub(null));
  useEffect(() => { load(); }, []);

  const applyCoupon = async () => {
    if (!coupon.trim()) { setPreview(null); return; }
    setChecking(true); setErr("");
    try {
      setPreview(await subscriptionApi.validateCoupon(coupon.trim()));
    } catch {
      setPreview(null);
    } finally { setChecking(false); }
  };

  const subscribe = async () => {
    setBusy(true); setErr("");
    try {
      const r = await subscriptionApi.subscribe(coupon.trim() || undefined);
      if (r.free) { await load(); return; } // fully-discounted → premium granted now
      if (r.pay_url) window.open(r.pay_url, "_blank", "noopener");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "could not start payment");
    } finally { setBusy(false); }
  };

  const idr = (n: number) => "Rp" + n.toLocaleString("id-ID");
  const fmtDate = (iso?: string) => iso ? new Date(iso).toLocaleDateString([], { day: "numeric", month: "short", year: "numeric" }) : "";

  if (!sub) {
    return (
      <div className="flex justify-center rounded-xl border border-white/10 bg-white/[0.02] p-4"><Loader2 className="h-4 w-4 animate-spin text-white/30" /></div>
    );
  }

  if (sub.active) {
    return (
      <div className="space-y-3">
        <div className="rounded-xl border border-amber-400/25 bg-gradient-to-b from-amber-500/10 to-transparent p-3.5">
          <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-amber-300"><Crown className="h-3.5 w-3.5" /> Premium</div>
          <p className="text-xs text-white/60">You're Premium — full access to cloud features.</p>
          {sub.premium_until && <p className="mt-1 text-[11px] text-white/40">Expires {fmtDate(sub.premium_until)}</p>}
        </div>
        <GiftPremium />
      </div>
    );
  }

  const finalPrice = preview?.valid ? preview.final_price : sub.price;
  const isFree = preview?.valid && finalPrice <= 0;

  return (
    <div className="rounded-xl border border-white/10 bg-gradient-to-b from-indigo-500/10 to-transparent p-3.5">
      <div className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-indigo-300"><Sparkles className="h-3.5 w-3.5" /> Upgrade to Premium</div>
      <ul className="mb-3 space-y-1 text-xs text-white/60">
        <li className="flex items-center gap-1.5"><Check className="h-3 w-3 shrink-0 text-emerald-400" /> Full cloud sync (providers, accounts, keys, settings)</li>
        <li className="flex items-center gap-1.5"><Check className="h-3 w-3 shrink-0 text-emerald-400" /> More paid cloud features as they land</li>
      </ul>

      {/* Optional coupon. */}
      <div className="mb-2 flex items-center gap-1.5">
        <input
          value={coupon}
          onChange={(e) => { setCoupon(e.target.value.toUpperCase()); setPreview(null); }}
          onKeyDown={(e) => e.key === "Enter" && applyCoupon()}
          placeholder="Coupon code (optional)"
          className="min-w-0 flex-1 rounded-md border border-white/10 bg-black/30 px-2.5 py-1.5 text-xs text-white outline-none focus:border-white/25"
        />
        <button onClick={applyCoupon} disabled={checking || !coupon.trim()} className="shrink-0 rounded-md border border-white/10 px-2.5 py-1.5 text-[11px] text-white/70 hover:bg-white/5 disabled:opacity-40">
          {checking ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Apply"}
        </button>
      </div>
      {preview && (
        <p className={`mb-2 text-[11px] ${preview.valid ? "text-emerald-300" : "text-red-300"}`}>
          {preview.valid
            ? (isFree ? "Coupon applied — Premium is free! 🎉" : `Coupon applied — ${idr(preview.discount)} off.`)
            : preview.message}
        </p>
      )}

      <button onClick={subscribe} disabled={busy || (!isFree && !sub.pay_enabled)} className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-white px-3 py-2 text-xs font-semibold text-black hover:opacity-90 disabled:opacity-50">
        {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Crown className="h-3.5 w-3.5" />}
        {isFree ? "Claim Premium · Free" : (
          <>Subscribe · {preview?.valid && finalPrice !== sub.price ? (<><span className="line-through opacity-50">{idr(sub.price)}</span> {idr(finalPrice)}</>) : idr(sub.price)}/mo</>
        )}
      </button>
      {!sub.pay_enabled && !isFree && <p className="mt-1.5 text-[11px] text-white/35">Payment is not configured yet — use a coupon.</p>}
      {err && <p className="mt-1.5 text-[11px] text-red-300">{err}</p>}

      <div className="mt-3 border-t border-white/5 pt-3"><GiftPremium /></div>
    </div>
  );
}

// GiftPremium gifts Premium (full price) to another user, in a modal: search a
// recipient by username/display name, then pay. The recipient must not already be
// Premium.
function GiftPremium() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button onClick={() => setOpen(true)} className="flex items-center gap-1.5 text-[11px] text-white/50 hover:text-white/80">
        <Gift className="h-3.5 w-3.5" /> Gift Premium to a friend
      </button>
      {open && <GiftModal onClose={() => setOpen(false)} />}
    </>
  );
}

function GiftModal({ onClose }: { onClose: () => void }) {
  const [q, setQ] = useState("");
  const [hits, setHits] = useState<UserHit[]>([]);
  const [searching, setSearching] = useState(false);
  const [pick, setPick] = useState<UserHit | null>(null);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  // Realtime search: any partial of username or display name.
  useEffect(() => {
    if (pick || q.trim().length < 2) { setHits([]); return; }
    setSearching(true);
    const t = setTimeout(() => {
      subscriptionApi.searchUsers(q.trim())
        .then((r) => setHits(r.users ?? []))
        .catch(() => setHits([]))
        .finally(() => setSearching(false));
    }, 250);
    return () => clearTimeout(t);
  }, [q, pick]);

  const gift = async () => {
    if (!pick) return;
    setBusy(true); setErr(""); setMsg("");
    try {
      const r = await subscriptionApi.gift(pick.username);
      if (r.pay_url) { window.open(r.pay_url, "_blank", "noopener"); setMsg("Opening payment…"); }
      else setMsg("Gift started.");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "gift failed");
    } finally { setBusy(false); }
  };

  return createPortal(
    <div className="fixed inset-0 z-[10000] flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm" onClick={() => !busy && onClose()}>
      <div className="w-full max-w-sm overflow-hidden rounded-2xl border border-white/10 bg-[#0e1016] shadow-2xl" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between border-b border-white/5 px-4 py-3">
          <div className="flex items-center gap-1.5 text-sm font-semibold text-white"><Gift className="h-4 w-4 text-indigo-300" /> Gift Premium</div>
          <button onClick={onClose} className="rounded p-0.5 text-white/40 hover:bg-white/10 hover:text-white"><X className="h-4 w-4" /></button>
        </div>
        <div className="space-y-3 p-4">
          <p className="text-[11px] text-white/45">Gift one month of Premium to another member — full price, no coupon.</p>

          {pick ? (
            <div className="flex items-center gap-2.5 rounded-lg border border-white/10 bg-white/[0.03] px-3 py-2">
              {pick.avatar_url ? <img src={pick.avatar_url} alt="" className="h-9 w-9 rounded-full" /> : <div className="flex h-9 w-9 items-center justify-center rounded-full bg-white/10 text-sm text-white/60">{(pick.display_name || pick.username).charAt(0).toUpperCase()}</div>}
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-white">{pick.display_name || pick.username}</div>
                <div className="truncate text-[11px] text-white/40">@{pick.username}</div>
              </div>
              <button onClick={() => { setPick(null); setQ(""); }} className="text-white/40 hover:text-white"><X className="h-4 w-4" /></button>
            </div>
          ) : (
            <div>
              <div className="relative">
                <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-white/25" />
                <input autoFocus value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search username or name…" className="w-full rounded-lg border border-white/10 bg-black/30 py-2 pl-8 pr-2 text-sm text-white outline-none focus:border-white/25" />
                {searching && <Loader2 className="absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 animate-spin text-white/30" />}
              </div>
              {q.trim().length >= 2 && (
                <div className="mt-1.5 max-h-56 overflow-auto rounded-lg border border-white/10">
                  {hits.length === 0 && !searching ? (
                    <div className="px-3 py-3 text-center text-[11px] text-white/35">No users found.</div>
                  ) : hits.map((h) => (
                    <button key={h.id} onClick={() => { setPick(h); setHits([]); }} className="flex w-full items-center gap-2.5 border-b border-white/5 px-3 py-2 text-left last:border-0 hover:bg-white/5">
                      {h.avatar_url ? <img src={h.avatar_url} alt="" className="h-8 w-8 rounded-full" /> : <div className="flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-xs text-white/60">{(h.display_name || h.username).charAt(0).toUpperCase()}</div>}
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-xs font-medium text-white">{h.display_name || h.username}</div>
                        <div className="truncate text-[10px] text-white/40">@{h.username}</div>
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          <button onClick={gift} disabled={!pick || busy} className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-white px-3 py-2 text-xs font-semibold text-black hover:opacity-90 disabled:opacity-50">
            {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Gift className="h-3.5 w-3.5" />} Gift Premium
          </button>
          {msg && <p className="text-[11px] text-emerald-300">{msg}</p>}
          {err && <p className="text-[11px] text-red-300">{err}</p>}
        </div>
      </div>
    </div>,
    document.body,
  );
}
