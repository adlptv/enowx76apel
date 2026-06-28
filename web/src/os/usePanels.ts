import { useCallback, useState } from "react";
import type { AppId, Side } from "./types";

// One active app per side. Opening an app toggles it (same app closes the side;
// a different app on the same side replaces it).
export function usePanels() {
  const [active, setActive] = useState<Record<Side, AppId | null>>({ left: null, right: null });

  const toggle = useCallback((side: Side, appId: AppId) => {
    setActive((prev) => ({ ...prev, [side]: prev[side] === appId ? null : appId }));
  }, []);

  const close = useCallback((side: Side) => {
    setActive((prev) => ({ ...prev, [side]: null }));
  }, []);

  return { active, toggle, close };
}
