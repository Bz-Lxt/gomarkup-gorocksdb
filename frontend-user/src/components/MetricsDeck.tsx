import { useStore } from "../store";

export function MetricsDeck() {
  const m = useStore((s) => s.metrics);
  const lsm = useStore((s) => s.lsm);
  const items = [
    ["写入", m.puts],
    ["点查", m.gets],
    ["命中", m.get_hits],
    ["Flush", m.flushes],
    ["Compaction", m.compactions],
    ["Bloom 拦截", m.bloom_rejects],
    ["Cache 命中率", Number(m.cache_hit_rate || 0).toFixed(2)],
    ["写延迟均 ms", Number(m.avg_write_ms || 0).toFixed(3)],
    ["Seq", lsm?.last_sequence],
    ["Stall", lsm?.write_stall ? "YES" : "no"],
  ] as const;
  return (
    <section className="rounded-2xl border border-white/10 bg-trench/80 p-4">
      <h3 className="font-display text-lg">仪表面板</h3>
      <div className="mt-3 grid grid-cols-2 gap-2">
        {items.map(([k, v]) => (
          <div key={k} className="rounded-lg bg-abyss px-2 py-2">
            <p className="text-[10px] uppercase tracking-wider text-mist">{k}</p>
            <p className="font-mono text-sm text-foam">{String(v ?? "—")}</p>
          </div>
        ))}
      </div>
    </section>
  );
}
