import { ShieldCheck, Coins, Link as LinkIcon } from "lucide-react";
import type { TopRole, ProfileLink } from "../lib/api";

// CardProfile is the shape the card renders. Both SyncUser (self) and
// PublicProfile (others) satisfy it, so the card is reused in both places.
export interface CardProfile {
  username: string;
  avatar_url?: string;
  display_name?: string;
  bio?: string;
  accent_color?: string;
  links?: ProfileLink[];
  plan?: string;
  top_role?: TopRole | null;
  wears_tag?: boolean;
  guild_tag?: string;
  kleos?: number;
  created_at?: string;
}

// ProfileCard is the Discord-style profile card: accent banner, large avatar
// overlapping it, name + handle, badges row, and an About section (bio, links,
// member-since). Used as the Profile-app hero and (later) as a popover.
export function ProfileCard({ p, footer }: { p: CardProfile; footer?: React.ReactNode }) {
  const accent = p.accent_color || "#6366f1";
  const initial = (p.display_name || p.username || "?").charAt(0).toUpperCase();

  return (
    <div className="overflow-hidden rounded-2xl border border-white/10 bg-white/[0.03]">
      {/* Banner — accent gradient (image upload is a future tag-gated perk). */}
      <div
        className="h-20 w-full"
        style={{ background: `linear-gradient(135deg, ${accent}, ${accent}55)` }}
      />

      <div className="px-4 pb-4">
        {/* Avatar overlapping the banner. */}
        <div className="-mt-9 mb-2 flex items-end justify-between">
          <div
            className="flex h-[72px] w-[72px] items-center justify-center rounded-full ring-4"
            style={{ ["--tw-ring-color" as string]: "#0b0c10" }}
          >
            {p.avatar_url ? (
              <img src={p.avatar_url} alt="" className="h-full w-full rounded-full" />
            ) : (
              <div
                className="flex h-full w-full items-center justify-center rounded-full text-xl font-bold text-white"
                style={{ background: `linear-gradient(135deg, ${accent}, ${accent}99)` }}
              >
                {initial}
              </div>
            )}
          </div>
          {p.kleos !== undefined && (
            <span className="mb-1 flex items-center gap-1 rounded-full bg-amber-400/[0.08] px-2 py-0.5 text-[11px] font-semibold text-amber-200 ring-1 ring-inset ring-amber-400/15">
              <span className="flex h-3.5 w-3.5 items-center justify-center rounded-full bg-gradient-to-br from-amber-300 to-amber-500">
                <Coins className="h-2 w-2 text-amber-950" />
              </span>
              {p.kleos.toLocaleString()}
            </span>
          )}
        </div>

        {/* Name + handle. */}
        <div className="flex items-baseline gap-1.5">
          <span className="truncate text-base font-bold text-white">{p.display_name || p.username}</span>
          {p.display_name && <span className="truncate text-xs text-white/35">@{p.username}</span>}
        </div>

        {/* Badges. */}
        <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
          {p.top_role?.name ? <RoleBadge role={p.top_role} /> : p.plan && <PlanBadge plan={p.plan} />}
          {p.wears_tag && <TagBadge tag={p.guild_tag} />}
        </div>

        {/* About: bio + links + member since. */}
        {(p.bio || (p.links && p.links.length > 0) || p.created_at) && (
          <div className="mt-3 space-y-2 border-t border-white/10 pt-3">
            {p.bio && <p className="text-xs leading-relaxed text-white/70">{p.bio}</p>}
            {p.links && p.links.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {p.links.map((l, i) => (
                  <a
                    key={i}
                    href={l.url}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="flex items-center gap-1 rounded-full border border-white/10 bg-white/5 px-2 py-0.5 text-[11px] text-white/70 hover:bg-white/10 hover:text-white"
                  >
                    <LinkIcon className="h-2.5 w-2.5" /> {l.label || l.url}
                  </a>
                ))}
              </div>
            )}
            {p.created_at && (
              <p className="text-[10px] uppercase tracking-wide text-white/30">
                Member since {new Date(p.created_at).toLocaleDateString(undefined, { month: "short", year: "numeric" })}
              </p>
            )}
          </div>
        )}

        {footer && <div className="mt-3">{footer}</div>}
      </div>
    </div>
  );
}

// hex turns a Discord decimal color into #rrggbb.
function hex(n: number): string {
  return "#" + (n & 0xffffff).toString(16).padStart(6, "0");
}

function RoleBadge({ role }: { role: TopRole }) {
  const c1 = hex(role.primary || role.color);
  const c2 = role.secondary ? hex(role.secondary) : c1;
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold ring-1 ring-inset ring-white/10"
      style={{ background: `${c1}22` }}
    >
      {role.icon_url && <img src={role.icon_url} alt="" className="h-3.5 w-3.5" />}
      <span
        className="bg-clip-text text-transparent"
        style={{ backgroundImage: `linear-gradient(90deg, ${c1}, ${c2})`, WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent" }}
      >
        {role.name}
      </span>
    </span>
  );
}

function PlanBadge({ plan }: { plan: string }) {
  return (
    <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-emerald-300">
      {plan}
    </span>
  );
}

function TagBadge({ tag }: { tag?: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-indigo-500/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-indigo-200 ring-1 ring-inset ring-indigo-400/20">
      <ShieldCheck className="h-3 w-3" /> {tag ? `[${tag}]` : "Tag"}
    </span>
  );
}
