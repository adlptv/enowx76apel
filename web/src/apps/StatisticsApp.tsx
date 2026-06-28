import { useEffect, useState } from "react";
import { AppShell } from "./shell";
import { TermGauge, TermBarRow } from "../components/term/TermChart";
import { AreaChart } from "../components/term/AreaChart";
import { useLiveSeries } from "../lib/useLiveSeries";
import {
  requestsApi,
  type RequestSummary,
  type ModelStat,
} from "../lib/api";

// Realtime counters: poll the running totals each second and chart the delta.
async function readReqTotal(): Promise<number> {
  const s = await requestsApi.summary();
  return s.total;
}
async function readTokTotal(): Promise<number> {
  const s = await requestsApi.summary();
  return s.in_tokens + s.out_tokens;
}

const compact = (n: number) =>
  n >= 1_000_000 ? `${(n / 1_000_000).toFixed(1)}M` : n >= 1_000 ? `${(n / 1_000).toFixed(1)}K` : String(n);

export function StatisticsApp() {
  const [summary, setSummary] = useState<RequestSummary | null>(null);
  const [models, setModels] = useState<ModelStat[]>([]);

  useEffect(() => {
    let alive = true;
    const load = () => {
      requestsApi.summary().then((s) => alive && setSummary(s)).catch(() => {});
      requestsApi.topModels(8).then((m) => alive && setModels(m ?? [])).catch(() => {});
    };
    load();
    const id = setInterval(load, 5000);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, []);

  const reqLive = useLiveSeries(readReqTotal, { intervalMs: 1000, capacity: 120 });
  const tokLive = useLiveSeries(readTokTotal, { intervalMs: 1000, capacity: 120 });
  const rps = reqLive.length ? reqLive[reqLive.length - 1].v : 0;
  const tps = tokLive.length ? tokLive[tokLive.length - 1].v : 0;
  const okRate = summary && summary.total > 0 ? Math.round((summary.ok / summary.total) * 100) : 0;
  const errRate = summary && summary.total > 0 ? Math.round((summary.errors / summary.total) * 100) : 0;
  const maxModel = Math.max(...models.map((m) => m.requests), 1);
  // Latency gauge: map 0..2000ms onto 0..100% (lower is better, so invert tone by value).
  const latPct = summary ? Math.min(100, Math.round((summary.avg_ms / 2000) * 100)) : 0;

  return (
    <AppShell title="Statistics" subtitle="Live usage — realtime">
      <div className="space-y-3">
        <Panel title="REQUESTS / SEC (LIVE)" hint={`${rps} req/s now`}>
          <div className="h-56">
            <AreaChart points={reqLive} unit="req/s" />
          </div>
        </Panel>

        <Panel title="TOKENS / SEC (LIVE)" hint={`${compact(tps)} tok/s now`}>
          <div className="h-56">
            <AreaChart points={tokLive} unit="tok/s" />
          </div>
        </Panel>

        <Panel title="HEALTH">
          <div className="space-y-1.5">
            <TermGauge label="success" percent={okRate} tone="text-emerald-400" />
            <TermGauge label="errors" percent={errRate} tone="text-red-400" />
            <TermGauge label="latency" percent={latPct} tone={latPct > 60 ? "text-amber-400" : "text-emerald-400"} />
            <p className="pt-1 font-mono text-[11px] text-white/40">
              avg {summary?.avg_ms ?? 0}ms · {summary?.total ?? 0} req today
            </p>
          </div>
        </Panel>

        <Panel title="TOP MODELS (TODAY)">
          {models.length === 0 ? (
            <p className="font-mono text-[11px] text-white/40">no requests yet today</p>
          ) : (
            <div className="space-y-1">
              {models.map((m) => (
                <TermBarRow
                  key={m.model}
                  label={m.model}
                  value={m.requests}
                  max={maxModel}
                  suffix={`${m.requests} req`}
                />
              ))}
            </div>
          )}
        </Panel>
      </div>
    </AppShell>
  );
}

function Panel({ title, hint, children }: { title: string; hint?: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-emerald-500/15 bg-black/40 p-3">
      <div className="mb-2 flex items-center justify-between">
        <span className="font-mono text-[10px] font-semibold tracking-widest text-emerald-400/80">{title}</span>
        {hint && <span className="font-mono text-[10px] text-white/35">{hint}</span>}
      </div>
      {children}
    </div>
  );
}
