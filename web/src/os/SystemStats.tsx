import { useEffect, useRef, useState } from "react";
import { Cpu, MemoryStick, X } from "lucide-react";
import { debugApi, type DebugInfo } from "../lib/api";
import { Sparkline } from "../components/Sparkline";

const mb = (b: number) => `${(b / 1024 / 1024).toFixed(0)} MB`;

function uptime(sec: number) {
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

const CAP = 40;

// SystemStats is the TopBar CPU/MEM readout for the enx process itself; click to
// open a Debug popover (process resources + Go runtime + build info).
export function SystemStats() {
  const [info, setInfo] = useState<DebugInfo | null>(null);
  const [open, setOpen] = useState(false);
  const cpuHist = useRef<number[]>([]);
  const memHist = useRef<number[]>([]);
  const [, force] = useState(0);

  useEffect(() => {
    let alive = true;
    const tick = () => {
      debugApi
        .get()
        .then((d) => {
          if (!alive) return;
          setInfo(d);
          cpuHist.current = [...cpuHist.current, d.process.cpu_percent].slice(-CAP);
          memHist.current = [...memHist.current, d.process.rss / 1024 / 1024].slice(-CAP);
          force((n) => n + 1);
        })
        .catch(() => {});
    };
    tick();
    const id = setInterval(tick, 2000);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, []);

  const cpu = info ? Math.round(info.process.cpu_percent) : 0;
  const memMB = info ? Math.round(info.process.rss / 1024 / 1024) : 0;

  return (
    <div className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-2 rounded px-1.5 py-0.5 font-mono text-[10px] tabular-nums text-white/70 transition-colors hover:bg-white/10"
        title="Process debug"
      >
        <span className="flex items-center gap-1">
          <Cpu className="h-3 w-3 text-emerald-400/80" /> {cpu}%
        </span>
        <span className="flex items-center gap-1">
          <MemoryStick className="h-3 w-3 text-emerald-400/80" /> {memMB}MB
        </span>
      </button>

      {open && info && <DebugPopover info={info} cpuHist={cpuHist.current} memHist={memHist.current} onClose={() => setOpen(false)} />}
    </div>
  );
}

function DebugPopover({
  info,
  cpuHist,
  memHist,
  onClose,
}: {
  info: DebugInfo;
  cpuHist: number[];
  memHist: number[];
  onClose: () => void;
}) {
  return (
    <div className="absolute right-0 top-8 z-[10000] w-80 overflow-hidden rounded-2xl border border-white/10 bg-[#11131a]/98 shadow-2xl glass">
      <div className="flex items-center justify-between border-b border-white/5 px-3 py-2">
        <span className="text-xs font-semibold text-white/80">Debug · enx process</span>
        <button onClick={onClose} className="rounded p-1 text-white/40 hover:bg-white/10 hover:text-white">
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="max-h-[70vh] space-y-3 overflow-auto p-3">
        <div className="grid grid-cols-2 gap-2">
          <Card title="CPU" value={`${info.process.cpu_percent}%`}>
            <Sparkline values={cpuHist} />
          </Card>
          <Card title="MEM (RSS)" value={mb(info.process.rss)}>
            <Sparkline values={memHist} />
          </Card>
        </div>

        <Section title="RUNTIME">
          <Row k="Goroutines" v={String(info.goroutines)} />
          <Row k="Heap alloc" v={mb(info.memory.heap_alloc)} />
          <Row k="Heap sys" v={mb(info.memory.heap_sys)} />
          <Row k="Live objects" v={info.memory.live_objects.toLocaleString()} />
          <Row k="GC runs" v={String(info.gc.num_gc)} />
          <Row k="GC cpu" v={`${(info.gc.gc_cpu_fraction * 100).toFixed(2)}%`} />
        </Section>

        <Section title="BUILD">
          <Row k="Version" v={`enx ${info.build.version}`} />
          <Row k="Go" v={info.build.go_version} />
          <Row k="Platform" v={`${info.build.os}/${info.build.arch}`} />
          <Row k="CPUs" v={`${info.build.num_cpu} (procs ${info.build.max_procs})`} />
          <Row k="PID" v={String(info.process.pid)} />
          <Row k="Uptime" v={uptime(info.uptime_seconds)} />
        </Section>
      </div>
    </div>
  );
}

function Card({ title, value, children }: { title: string; value: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-emerald-500/15 bg-black/40 p-2.5">
      <div className="flex items-baseline justify-between">
        <span className="font-mono text-[10px] tracking-widest text-emerald-400/80">{title}</span>
      </div>
      <div className="my-1 text-lg font-bold tabular-nums text-white">{value}</div>
      <div className="text-emerald-400">{children}</div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.02] p-2.5">
      <p className="mb-1.5 font-mono text-[10px] tracking-widest text-white/40">{title}</p>
      <div className="space-y-1">{children}</div>
    </div>
  );
}

function Row({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-center justify-between font-mono text-[11px]">
      <span className="text-white/40">{k}</span>
      <span className="tabular-nums text-white/75">{v}</span>
    </div>
  );
}
