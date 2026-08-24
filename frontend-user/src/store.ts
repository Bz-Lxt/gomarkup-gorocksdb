import { create } from "zustand";
import type { EngineEvent, LSMState, Toast } from "./types";

export type CompactionShow = {
  active: boolean;
  level: number;
  outputLevel: number;
  inputs: number[];
  outputs: number[];
  keysRead: number;
  droppedVersions: number;
  droppedTombs: number;
  durationMs: number;
  result?: Record<string, unknown>;
};

type Store = {
  connected: boolean;
  lsm: LSMState | null;
  metrics: Record<string, unknown>;
  events: EngineEvent[];
  compaction: CompactionShow;
  toasts: Toast[];
  setConnected: (v: boolean) => void;
  applyEvent: (ev: EngineEvent) => void;
  pushToast: (kind: Toast["kind"], text: string) => void;
  dismissToast: (id: number) => void;
};

let toastSeq = 1;

const emptyComp: CompactionShow = {
  active: false,
  level: 0,
  outputLevel: 1,
  inputs: [],
  outputs: [],
  keysRead: 0,
  droppedVersions: 0,
  droppedTombs: 0,
  durationMs: 0,
};

export const useStore = create<Store>((set) => ({
  connected: false,
  lsm: null,
  metrics: {},
  events: [],
  compaction: emptyComp,
  toasts: [],
  setConnected: (v) => set({ connected: v }),
  applyEvent: (ev) =>
    set((s) => {
      const next: Partial<Store> = {
        events: [{ ...ev }, ...s.events].slice(0, 40),
      };
      const p = ev.payload || {};
      if (ev.type === "lsm.snapshot" || (ev.type === "metrics.tick" && p.lsm)) {
        next.lsm = (ev.type === "lsm.snapshot" ? p : (p.lsm as object)) as LSMState;
      }
      if (ev.type === "metrics.tick") {
        next.metrics = p;
        if (p.lsm) next.lsm = p.lsm as LSMState;
      }
      if (ev.type === "compaction.start") {
        next.compaction = {
          ...emptyComp,
          active: true,
          level: Number(p.level || 0),
          outputLevel: Number(p.output_level || 1),
          inputs: (p.input_files as number[]) || [],
        };
      }
      if (ev.type === "compaction.progress" && s.compaction.active) {
        next.compaction = {
          ...s.compaction,
          keysRead: Number(p.keys_read || 0),
          droppedVersions: Number(p.dropped_versions || 0),
          droppedTombs: Number(p.dropped_tombs || 0),
        };
      }
      if (ev.type === "compaction.done") {
        next.compaction = {
          ...s.compaction,
          active: false,
          outputs: (p.output_files as number[]) || [],
          keysRead: Number(p.keys_read || 0),
          droppedVersions: Number(p.dropped_versions || 0),
          droppedTombs: Number(p.dropped_tombs || 0),
          durationMs: Number(p.duration_ms || 0),
          result: p,
        };
      }
      return next;
    }),
  pushToast: (kind, text) => {
    const id = toastSeq++;
    set((s) => ({ toasts: [...s.toasts, { id, kind, text }] }));
    setTimeout(() => {
      set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
    }, 5000);
  },
  dismissToast: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));
