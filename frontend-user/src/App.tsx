import { useEffect } from "react";
import { BenchPanel } from "./components/BenchPanel";
import { CompactionStage } from "./components/CompactionStage";
import { ConsolePanel } from "./components/ConsolePanel";
import { LevelReef } from "./components/LevelReef";
import { MemSurface } from "./components/MemSurface";
import { MetricsDeck } from "./components/MetricsDeck";
import { ToastStack } from "./components/ToastStack";
import { api } from "./lib/api";
import { connectEvents } from "./lib/ws";
import { useStore } from "./store";

export default function App() {
  const connected = useStore((s) => s.connected);
  const lsm = useStore((s) => s.lsm);
  const compaction = useStore((s) => s.compaction);
  const events = useStore((s) => s.events);

  useEffect(() => connectEvents(), []);
  useEffect(() => {
    let alive = true;
    const tick = async () => {
      if (useStore.getState().connected) return;
      try {
        const [state, metrics] = await Promise.all([api.state(), api.metrics()]);
        if (!alive) return;
        useStore.getState().applyEvent({ type: "lsm.snapshot", payload: state as Record<string, unknown> });
        useStore.getState().applyEvent({ type: "metrics.tick", payload: metrics as Record<string, unknown> });
      } catch {
        /* keep last frame */
      }
    };
    tick();
    const id = window.setInterval(tick, 400);
    return () => {
      alive = false;
      window.clearInterval(id);
    };
  }, []);

  const hot = compaction.active ? compaction.inputs : compaction.outputs;

  return (
    <div className="relative min-h-screen w-full">
      <div className="grain" />
      <div className="scanline" />
      <ToastStack />
      {!connected && (
        <div className="relative z-10 bg-coral/20 px-4 py-2 text-center text-sm text-foam">
          WebSocket 已断开，正在重连测深声呐…
        </div>
      )}
      <header className="relative z-10 flex w-full flex-col gap-2 border-b border-white/10 px-4 py-4 md:flex-row md:items-end md:justify-between md:px-8">
        <div>
          <p className="font-display text-xs uppercase tracking-[0.3em] text-sonar">GoRocksDB</p>
          <h1 className="font-display text-3xl md:text-4xl">LSM 测深控制室</h1>
          <p className="text-sm text-mist">内存潮池 · 磁盘礁盘 · 归并洋流 · 真实引擎，无假卡片</p>
        </div>
        <div className="font-mono text-xs text-mist">
          档位 <span className="text-amber">{lsm?.profile || "—"}</span>
          {" · "}
          seq {lsm?.last_sequence ?? 0}
          {lsm?.compacting ? " · COMPACTING" : ""}
        </div>
      </header>
      <main className="relative z-10 mx-auto grid w-full grid-cols-1 gap-4 px-4 py-4 lg:grid-cols-[minmax(0,1fr)_360px] lg:px-8">
        <div className="space-y-4">
          <MemSurface lsm={lsm} />
          <CompactionStage show={compaction} />
          <LevelReef levels={lsm?.levels || []} hotFiles={hot} />
        </div>
        <aside className="space-y-4">
          <ConsolePanel />
          <BenchPanel />
          <MetricsDeck />
          <section className="rounded-2xl border border-white/10 bg-trench/80 p-4">
            <h3 className="font-display text-lg">事件潮汐</h3>
            <ul className="mt-2 max-h-48 space-y-1 overflow-auto font-mono text-[11px] text-mist">
              {events.length === 0 && <li>等待声呐回波</li>}
              {events.map((e, i) => (
                <li key={`${e.type}-${e.time}-${i}`}>
                  <span className="text-sonar">{e.time || ""}</span> {e.type}
                </li>
              ))}
            </ul>
          </section>
        </aside>
      </main>
    </div>
  );
}
