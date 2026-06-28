import { useState, type ReactNode } from "react";

type Place = "top" | "bottom" | "left" | "right";

const POS: Record<Place, string> = {
  top: "bottom-full left-1/2 mb-1.5 -translate-x-1/2",
  bottom: "top-full left-1/2 mt-1.5 -translate-x-1/2",
  left: "right-full top-1/2 mr-1.5 -translate-y-1/2",
  right: "left-full top-1/2 ml-1.5 -translate-y-1/2",
};

// Tooltip shows a glass label on hover/focus. Every interactive control should
// explain itself — see AGENTS.md "Buttons must explain themselves".
export function Tooltip({ label, place = "top", children }: { label: string; place?: Place; children: ReactNode }) {
  const [show, setShow] = useState(false);
  return (
    <span
      className="relative inline-flex"
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
      onFocusCapture={() => setShow(true)}
      onBlurCapture={() => setShow(false)}
    >
      {children}
      {show && (
        <span
          role="tooltip"
          className={`pointer-events-none absolute z-[10000] whitespace-nowrap rounded-md bg-black/85 px-2 py-0.5 text-[11px] font-medium text-white ring-1 ring-white/10 ${POS[place]}`}
        >
          {label}
        </span>
      )}
    </span>
  );
}
