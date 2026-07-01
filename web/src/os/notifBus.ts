import { useEffect, useState } from "react";
import { notifApi, type Notification } from "../lib/api";

// notifBus is the shared notifications store: loads recent notifications + the
// unread count, and applies live `notification` events from the SSE stream.
let items: Notification[] = [];
let unread = 0;
let loaded = false;
let es: EventSource | null = null;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

export async function loadNotifications() {
  try {
    const r = await notifApi.list();
    items = r.notifications ?? [];
    unread = r.unread ?? 0;
    loaded = true;
  } catch {
    /* ignore */
  }
  emit();
}

export async function markNotificationsRead() {
  unread = 0;
  items = items.map((n) => ({ ...n, read: true }));
  emit();
  await notifApi.markRead().catch(() => {});
}

function ensureStream() {
  if (es) return;
  es = new EventSource("/api/chat/stream");
  es.addEventListener("message", (e) => {
    try {
      const ev = JSON.parse((e as MessageEvent).data) as { event: string; data: Notification };
      if (ev.event === "notification" && ev.data) {
        items = [ev.data, ...items].slice(0, 50);
        unread += 1;
        emit();
      }
    } catch {
      /* ignore */
    }
  });
}

export interface NotifState {
  items: Notification[];
  unread: number;
}

export function useNotifications(): NotifState {
  const [, force] = useState(0);
  useEffect(() => {
    const l = () => force((n) => n + 1);
    listeners.add(l);
    if (!loaded) loadNotifications();
    ensureStream();
    return () => {
      listeners.delete(l);
    };
  }, []);
  return { items, unread };
}
