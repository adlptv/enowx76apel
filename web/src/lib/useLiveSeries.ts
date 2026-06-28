import { useEffect, useRef, useState } from "react";

export interface LivePoint {
  t: number; // epoch ms
  v: number; // value for this 1s tick
}

// useLiveSeries polls a counter every second and emits the per-second delta as a
// rolling buffer. When the counter does not move it pushes 0, so the chart keeps
// scrolling flat instead of freezing.
export function useLiveSeries(
  read: () => Promise<number>,
  { intervalMs = 1000, capacity = 120 }: { intervalMs?: number; capacity?: number } = {},
) {
  const [points, setPoints] = useState<LivePoint[]>([]);
  const prev = useRef<number | null>(null);

  useEffect(() => {
    let alive = true;
    const tick = async () => {
      let current: number;
      try {
        current = await read();
      } catch {
        current = prev.current ?? 0;
      }
      if (!alive) return;
      const last = prev.current;
      prev.current = current;
      // First sample establishes the baseline; emit 0 so the line starts flat.
      const delta = last === null ? 0 : Math.max(0, current - last);
      setPoints((buf) => {
        const next = [...buf, { t: Date.now(), v: delta }];
        return next.length > capacity ? next.slice(next.length - capacity) : next;
      });
    };
    tick();
    const id = setInterval(tick, intervalMs);
    return () => {
      alive = false;
      clearInterval(id);
    };
    // read is expected to be stable (defined once by caller)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, capacity]);

  return points;
}
