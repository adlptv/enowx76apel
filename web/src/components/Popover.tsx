import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";

// Reusable popover panel with click-away + Escape to dismiss. Render it
// conditionally (when open) as a sibling of its anchor inside a `relative`
// container; pass `anchor` to position it. The transparent backdrop catches any
// outside click so the user never has to click the trigger again to close.
//
// It auto-flips vertically: if there isn't enough room below the anchor, it
// opens upward so the panel is never clipped off-screen. Pass `valign="up"` to
// force upward.
//
// MANDATORY: every dismissable popover/dropdown must close on an outside click
// (and Escape). Use this component instead of a bare absolute panel — see
// AGENTS.md "Popovers dismiss on outside click".
export function Popover({
  onClose,
  children,
  className = "",
  anchor = "right",
  valign = "auto",
}: {
  onClose: () => void;
  children: ReactNode;
  className?: string;
  anchor?: "left" | "right" | "center";
  valign?: "auto" | "down" | "up";
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [up, setUp] = useState(valign === "up");

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // Auto-flip: measure the panel against the viewport and open upward when the
  // bottom would be clipped but there's room above.
  useLayoutEffect(() => {
    if (valign !== "auto") {
      setUp(valign === "up");
      return;
    }
    const el = ref.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const margin = 12;
    const spaceBelow = window.innerHeight - r.top;
    const needed = el.offsetHeight + margin;
    // r.top is where the down-anchored panel starts; if the panel doesn't fit
    // below and there is more room above the anchor, flip up.
    setUp(spaceBelow < needed && r.top > window.innerHeight - r.top);
  }, [valign, children]);

  const pos =
    anchor === "left"
      ? "left-0"
      : anchor === "center"
        ? "left-1/2 -translate-x-1/2"
        : "right-0";
  const vpos = up ? "bottom-full mb-1" : "top-8";

  return (
    <>
      {/* Transparent click-away layer: any click outside the panel closes it. */}
      <div className="pointer-events-auto fixed inset-0 z-[10000]" onClick={onClose} />
      <div ref={ref} className={`pointer-events-auto absolute z-[10001] ${pos} ${vpos} ${className}`} onClick={(e) => e.stopPropagation()}>
        {children}
      </div>
    </>
  );
}
