import type { AppId, DesktopApp, Side } from "./types";

// A floating vertical dock pinned to one edge, centered vertically.
export function SideDock({
  side,
  apps,
  activeId,
  onOpen,
}: {
  side: Side;
  apps: DesktopApp[];
  activeId: AppId | null;
  onOpen: (side: Side, id: AppId) => void;
}) {
  const edge = side === "left" ? "left-3" : "right-3";
  const tip = side === "left" ? "left-14" : "right-14";

  return (
    <div className={`pointer-events-none absolute ${edge} top-1/2 z-[9000] -translate-y-1/2`}>
      <div className="glass pointer-events-auto flex flex-col gap-2 rounded-2xl border border-white/10 bg-[var(--dock-bg)] p-2 shadow-2xl">
        {apps.map((app) => (
          <button
            key={app.id}
            title={app.label}
            onClick={() => onOpen(side, app.id)}
            className="group relative flex items-center justify-center"
          >
            <span
              className={`absolute ${tip} whitespace-nowrap rounded-md bg-black/80 px-2 py-0.5 text-[11px] font-medium text-white opacity-0 ring-1 ring-white/10 transition-opacity group-hover:opacity-100`}
            >
              {app.label}
            </span>
            <span
              className={
                "flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br text-white shadow-md [&>svg]:!h-5 [&>svg]:!w-5 " +
                app.accent
              }
            >
              {app.icon}
            </span>
            <span
              className={
                "absolute h-1 w-1 rounded-full transition-opacity " +
                (side === "left" ? "-right-1.5" : "-left-1.5") +
                (activeId === app.id ? " bg-white/90 opacity-100" : " opacity-0")
              }
            />
          </button>
        ))}
      </div>
    </div>
  );
}
