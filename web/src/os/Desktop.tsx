import { AnimatePresence } from "framer-motion";
import { buildApps } from "../apps";
import { SideDock } from "./SideDock";
import { SidePanel } from "./SidePanel";
import { TopBar } from "./TopBar";
import { Widgets } from "./Widgets";
import { usePanels } from "./usePanels";
import type { AppId, Side } from "./types";

export function Desktop() {
  const apps = buildApps();
  const { active, toggle, close } = usePanels();

  const leftApps = apps.filter((a) => a.side === "left");
  const rightApps = apps.filter((a) => a.side === "right");
  const find = (id: AppId | null) => apps.find((a) => a.id === id);

  const renderPanel = (side: Side) => {
    const app = find(active[side]);
    return app ? <SidePanel side={side} app={app} onClose={() => close(side)} /> : null;
  };

  return (
    <div className="wallpaper fixed inset-0 select-none overflow-hidden">
      <div className="pointer-events-none absolute inset-x-0 top-7 bottom-3">
        <Widgets onOpen={(id) => toggle(panelSide(apps, id), id)} />
      </div>

      <TopBar />

      <SideDock side="left" apps={leftApps} activeId={active.left} onOpen={toggle} />
      <SideDock side="right" apps={rightApps} activeId={active.right} onOpen={toggle} />

      <AnimatePresence>{renderPanel("left")}</AnimatePresence>
      <AnimatePresence>{renderPanel("right")}</AnimatePresence>
    </div>
  );
}

function panelSide(apps: ReturnType<typeof buildApps>, id: AppId): Side {
  return apps.find((a) => a.id === id)?.side ?? "left";
}
