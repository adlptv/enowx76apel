import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";

// scrollParent walks up to the nearest ancestor that clips overflow (auto/scroll/
// hidden), which bounds where a popover can show. Falls back to the viewport.
function scrollParent(el: HTMLElement | null): HTMLElement | null {
  let p = el?.parentElement ?? null;
  while (p) {
    const oy = getComputedStyle(p).overflowY;
    if (oy === "auto" || oy === "scroll" || oy === "hidden") return p;
    p = p.parentElement;
  }
  return null;
}
export function clipBottom(el: HTMLElement): number {
  const p = scrollParent(el);
  return p ? Math.min(p.getBoundingClientRect().bottom, window.innerHeight) : window.innerHeight;
}
export function clipTop(el: HTMLElement): number {
  const p = scrollParent(el);
  return p ? Math.max(p.getBoundingClientRect().top, 0) : 0;
}

// shouldFlipUp reports whether a down-anchored panel should open upward. It
// measures against the ANCHOR (the offset parent), not the panel's own rect, so
// the decision is stable regardless of whether the panel is currently flipped —
// otherwise flipping would move the panel and re-trigger the opposite decision.
export function shouldFlipUp(el: HTMLElement): boolean {
  const anchor = (el.offsetParent as HTMLElement) ?? el;
  const a = anchor.getBoundingClientRect();
  const h = el.offsetHeight;
  const spaceBelow = clipBottom(el) - a.bottom; // room under the anchor
  const spaceAbove = a.top - clipTop(el); // room above the anchor
  return spaceBelow < h + 8 && spaceAbove > spaceBelow;
}

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

  // Auto-flip: measure the panel against its clipping container and open upward
  // when it wouldn't fit below. Re-measures on size changes (async content like a
  // profile card grows after its fetch resolves) via a ResizeObserver.
  useLayoutEffect(() => {
    if (valign !== "auto") {
      setUp(valign === "up");
      return;
    }
    const el = ref.current;
    if (!el) return;
    const measure = () => setUp(shouldFlipUp(el));
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
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
