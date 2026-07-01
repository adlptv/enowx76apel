import { useEffect, useState } from "react";
import type { Post } from "../lib/api";

// postViewer is a tiny global store for "which post's detail page is open".
// Any component can call openPost(post) to show the full-page post overlay
// (mounted once in Desktop, with its comment thread). Set to null to close.
let current: Post | null = null;
const listeners = new Set<() => void>();

export function openPost(post: Post) {
  current = post;
  listeners.forEach((l) => l());
}

export function closePost() {
  current = null;
  listeners.forEach((l) => l());
}

export function usePostViewer(): Post | null {
  const [, force] = useState(0);
  useEffect(() => {
    const l = () => force((n) => n + 1);
    listeners.add(l);
    return () => {
      listeners.delete(l);
    };
  }, []);
  return current;
}
