import { useEffect, useState } from "react";
import { Puzzle } from "lucide-react";
import { pluginsApi, type PluginManifest } from "../lib/api";
import type { DesktopApp } from "./types";

// PLUGIN_PREFIX namespaces plugin app ids so they never collide with built-ins.
export const PLUGIN_PREFIX = "plugin:";

// PluginFrame renders a plugin's UI (served at /plugins/<id>/) in an iframe. For
// a sidecar plugin it must be started first — offer a start button if not.
function PluginFrame({ plugin }: { plugin: PluginManifest }) {
  const [running, setRunning] = useState(!!plugin.running || plugin.runtime === "static");
  const [err, setErr] = useState("");

  const start = async () => {
    setErr("");
    try {
      await pluginsApi.start(plugin.id);
      setRunning(true);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "failed to start");
    }
  };

  if (!running) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
        <Puzzle className="h-8 w-8 text-white/40" />
        <p className="text-sm text-white/60">{plugin.name} isn't running.</p>
        {err && <p className="text-xs text-red-300">{err}</p>}
        <button onClick={start} className="rounded-lg bg-white px-4 py-2 text-sm font-medium text-black hover:opacity-90">Start plugin</button>
      </div>
    );
  }
  return (
    <iframe
      title={plugin.name}
      src={`/plugins/${plugin.id}/`}
      className="h-full w-full rounded-lg border border-white/10 bg-white"
      sandbox="allow-scripts allow-forms allow-same-origin allow-popups"
    />
  );
}

// usePluginApps fetches installed plugins and exposes them as DesktopApps so they
// appear in the WebOS drawer/dock with a "plugin" badge. Refreshes on focus so a
// newly-created plugin shows up.
export function usePluginApps(): DesktopApp[] {
  const [plugins, setPlugins] = useState<PluginManifest[]>([]);
  useEffect(() => {
    const load = () => pluginsApi.list().then((r) => setPlugins(r.plugins ?? [])).catch(() => {});
    load();
    window.addEventListener("focus", load);
    return () => window.removeEventListener("focus", load);
  }, []);

  return plugins.map((p) => ({
    id: PLUGIN_PREFIX + p.id,
    label: p.name,
    icon: <Puzzle />,
    accent: "from-violet-500 to-purple-600",
    home: "drawer" as const,
    badge: "plugin",
    render: () => <PluginFrame plugin={p} />,
  }));
}
