import { AnimatePresence, motion } from "framer-motion";
import type { CompactionShow } from "../store";

export function CompactionStage({ show }: { show: CompactionShow }) {
  return (
    <section className="relative min-h-[220px] overflow-hidden rounded-2xl border border-sonar/25 bg-abyss/80 p-5">
      <p className="font-display text-xs uppercase tracking-[0.25em] text-sonar">Current · Compaction</p>
      <h2 className="mt-1 font-display text-2xl text-foam">归并洋流舞台</h2>
      <AnimatePresence>
        {show.active ? (
          <motion.div
            key="live"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="mt-4"
          >
            <p className="font-mono text-xs text-mist">
              L{show.level} → L{show.outputLevel} · 源文件 {show.inputs.join(", ") || "—"}
            </p>
            <div className="mt-4 flex flex-wrap items-center justify-center gap-3">
              {show.inputs.map((n, i) => (
                <motion.div
                  key={n}
                  initial={{ x: -40 * (i + 1), opacity: 0 }}
                  animate={{ x: 0, opacity: 1 }}
                  transition={{ type: "spring", stiffness: 90, delay: i * 0.05 }}
                  className="rounded-lg border border-sonar/40 bg-sonar/10 px-3 py-2 font-mono text-xs"
                >
                  SST {n}
                </motion.div>
              ))}
              <motion.div
                className="h-10 w-10 rounded-full border border-amber/50"
                animate={{ rotate: 360, boxShadow: ["0 0 0px #F2B84B", "0 0 18px #3EE0C8", "0 0 0px #F2B84B"] }}
                transition={{ rotate: { repeat: Infinity, duration: 2, ease: "linear" }, boxShadow: { repeat: Infinity, duration: 1.6 } }}
              />
            </div>
            <div className="mt-4 flex justify-center gap-6 font-mono text-xs">
              <span className="text-foam">读 {show.keysRead}</span>
              <span className="text-amber">旧版本消散 {show.droppedVersions}</span>
              <span className="text-coral">墓碑融化 {show.droppedTombs}</span>
            </div>
            <div className="mt-3 flex justify-center gap-2">
              {Array.from({ length: 6 }).map((_, i) => (
                <motion.span
                  key={i}
                  className="h-2 w-8 rounded-full bg-sonar/50"
                  animate={{ y: [0, -8, 0], opacity: [0.3, 1, 0.3] }}
                  transition={{ repeat: Infinity, duration: 0.9, delay: i * 0.1 }}
                />
              ))}
            </div>
          </motion.div>
        ) : show.result ? (
          <motion.div key="done" initial={{ y: 16, opacity: 0 }} animate={{ y: 0, opacity: 1 }} className="mt-4">
            <p className="text-sm text-sonar">本轮归并完成</p>
            <div className="mt-3 grid grid-cols-2 gap-2 font-mono text-xs md:grid-cols-5">
              <Stat label="读取键" v={show.keysRead} />
              <Stat label="消除旧版本" v={show.droppedVersions} />
              <Stat label="清除墓碑" v={show.droppedTombs} />
              <Stat label="新文件" v={show.outputs.length} />
              <Stat label="耗时 ms" v={show.durationMs} />
            </div>
          </motion.div>
        ) : (
          <p className="mt-8 text-center text-sm text-mist">后台安静 · 等待 Level 过载触发洋流</p>
        )}
      </AnimatePresence>
    </section>
  );
}

function Stat({ label, v }: { label: string; v: number }) {
  return (
    <div className="rounded-lg border border-white/10 bg-trench px-2 py-2">
      <p className="text-mist">{label}</p>
      <p className="text-lg text-foam">{v}</p>
    </div>
  );
}
