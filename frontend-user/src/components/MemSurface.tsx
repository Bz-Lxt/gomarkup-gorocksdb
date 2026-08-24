import { motion } from "framer-motion";
import { fmtBytes } from "../lib/api";
import type { LSMState } from "../types";

export function MemSurface({ lsm }: { lsm: LSMState | null }) {
  if (!lsm) {
    return (
      <section className="rounded-2xl border border-white/10 bg-trench/80 p-5">
        <div className="h-24 animate-pulse rounded-xl bg-white/5" />
      </section>
    );
  }
  const pct = Math.min(100, Math.round(lsm.mem_ratio * 100));
  return (
    <section className="rounded-2xl border border-amber/20 bg-trench/80 p-5 shadow-amber">
      <div className="flex items-end justify-between gap-4">
        <div>
          <p className="font-display text-xs uppercase tracking-[0.25em] text-amber">Surface · MemTable</p>
          <h2 className="mt-1 font-display text-2xl text-foam">潮池写入面</h2>
        </div>
        <p className="font-mono text-sm text-mist">
          {lsm.mem_entries} keys · {fmtBytes(lsm.mem_bytes)} / {fmtBytes(lsm.mem_limit)}
        </p>
      </div>
      <div className="mt-4 h-4 overflow-hidden rounded-full bg-abyss">
        <motion.div
          className="h-full rounded-full bg-gradient-to-r from-amber to-sonar"
          animate={{ width: `${pct}%`, opacity: [0.75, 1, 0.75] }}
          transition={{ opacity: { repeat: Infinity, duration: 2.4 }, width: { duration: 0.4 } }}
        />
      </div>
      <p className="mt-2 font-mono text-amber">{pct}% 容量</p>
      <div className="mt-4 flex flex-wrap gap-2">
        {lsm.immutable.length === 0 ? (
          <span className="text-sm text-mist">Immutable 队列空 · 无需刷盘</span>
        ) : (
          lsm.immutable.map((m) => (
            <div
              key={m.id}
              className="rounded-lg border border-sonar/30 bg-abyss/60 px-3 py-2 font-mono text-xs text-sonar"
            >
              IMM #{m.id} · {m.entries} keys · {fmtBytes(m.bytes)}
            </div>
          ))
        )}
      </div>
    </section>
  );
}
