import { useEffect, useState } from "react";
import { Monitor } from "lucide-react";

// enowx is a desktop experience with a minimum usable size. When the viewport is
// too small (a phone/tablet, or a desktop window shrunk enough that the layout
// would get clipped), this gate covers the app with a black overlay asking the
// user to enlarge the window or switch to a larger screen.
const MIN_WIDTH = 1024;
const MIN_HEIGHT = 640;

function bigEnough(): boolean {
  const coarse = window.matchMedia("(pointer: coarse)").matches;
  if (coarse) return false; // touch device → not a desktop
  return window.innerWidth >= MIN_WIDTH && window.innerHeight >= MIN_HEIGHT;
}

export function RequireDesktop({ children }: { children: React.ReactNode }) {
  const [ok, setOk] = useState(bigEnough);

  useEffect(() => {
    const check = () => setOk(bigEnough());
    window.addEventListener("resize", check);
    return () => window.removeEventListener("resize", check);
  }, []);

  return (
    <>
      {children}
      {!ok && (
        <div className="fixed inset-0 z-[99999] flex flex-col items-center justify-center gap-6 bg-black px-6 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl border border-white/10 bg-white/5">
            <Monitor className="h-7 w-7 text-white/70" />
          </div>
          <div className="space-y-1.5">
            <h1 className="text-lg font-semibold text-white">Window too small</h1>
            <p className="max-w-sm text-sm text-white/50">
              enowx needs a bigger window so the layout isn't clipped. Enlarge this window (or open it on a larger screen) to continue.
            </p>
          </div>
        </div>
      )}
    </>
  );
}
