// pluginBus is a tiny signal for "the installed-plugin set changed" (created,
// deleted, started, stopped). The desktop's usePluginApps subscribes so plugin
// apps appear/disappear in the drawer/dock immediately, without waiting for a
// window-focus refresh.
const listeners = new Set<() => void>();

export function notifyPluginsChanged() {
  listeners.forEach((l) => l());
}

export function onPluginsChanged(fn: () => void): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}
