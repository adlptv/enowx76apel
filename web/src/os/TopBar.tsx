import { useEffect, useState } from "react";
import { SystemStats } from "./SystemStats";

export function TopBar() {
  const [clock, setClock] = useState("--:--");
  useEffect(() => {
    const tick = () => setClock(new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }));
    tick();
    const id = setInterval(tick, 15000);
    return () => clearInterval(id);
  }, []);
  return (
    <div className="glass absolute inset-x-0 top-0 z-[9999] flex h-7 items-center justify-between border-b border-white/5 bg-[var(--topbar-bg)] px-3 text-[11px] text-white/85">
      <span className="font-semibold tracking-wide">enowx</span>
      <div className="flex items-center gap-3">
        <SystemStats />
        <span className="tabular-nums text-white/70">{clock}</span>
      </div>
    </div>
  );
}
