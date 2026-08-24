import { useStore } from "../store";
import type { EngineEvent } from "../types";
import { wsURL } from "./api";

export function connectEvents() {
  let ws: WebSocket | null = null;
  let timer: number | undefined;
  const open = () => {
    ws = new WebSocket(wsURL());
    ws.onopen = () => useStore.getState().setConnected(true);
    ws.onclose = () => {
      useStore.getState().setConnected(false);
      timer = window.setTimeout(open, 1200);
    };
    ws.onerror = () => ws?.close();
    ws.onmessage = (m) => {
      try {
        const ev = JSON.parse(m.data) as EngineEvent;
        useStore.getState().applyEvent(ev);
      } catch {
        /* ignore malformed */
      }
    };
  };
  open();
  return () => {
    if (timer) window.clearTimeout(timer);
    ws?.close();
  };
}
