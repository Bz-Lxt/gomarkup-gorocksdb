import { motion } from "framer-motion";
import { fmtBytes } from "../lib/api";
import type { FileView, LevelView } from "../types";

function Card({ f, highlight }: { f: FileView; highlight: boolean }) {
  const w = Math.max(88, Math.min(220, 70 + Math.sqrt(f.size)));
  return (
    <motion.article
      layout
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0, scale: highlight ? 1.04 : 1 }}
      className={`rounded-xl border px-3 py-2 ${
        highlight
          ? "border-sonar bg-sonar/15 shadow-sonar"
          : "border-white/10 bg-abyss/50"
      }`}
      style={{ width: w }}
    >
      <p className="font-mono text-[10px] text-mist">SST {String(f.number).padStart(6, "0")}</p>
      <p className="truncate font-mono text-xs text-foam">
        [{f.min_key || "∅"} → {f.max_key || "∅"}]
      </p>
      <p className="mt-1 font-mono text-[10px] text-sonar">
        {fmtBytes(f.size)} · {f.entries} keys
      </p>
    </motion.article>
  );
}

export function LevelReef({
  levels,
  hotFiles,
}: {
  levels: LevelView[];
  hotFiles: number[];
}) {
  const focus = levels.filter((l) => l.level <= 2);
  const deep = levels.filter((l) => l.level >= 3);
  const deepFiles = deep.reduce((n, l) => n + l.files.length, 0);
  const deepBytes = deep.reduce((n, l) => n + l.bytes, 0);

  return (
    <section className="space-y-4">
      {focus.map((lv) => (
        <div key={lv.level} className="rounded-2xl border border-white/10 bg-trench/70 p-4">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="font-display text-lg text-foam">
              Level {lv.level}
              <span className="ml-2 font-mono text-xs text-mist">
                {lv.files.length} files · {fmtBytes(lv.bytes)} / {fmtBytes(lv.limit)}
              </span>
            </h3>
            <div className="h-1.5 w-32 overflow-hidden rounded-full bg-abyss">
              <div
                className="h-full bg-sonar/70"
                style={{ width: `${Math.min(100, lv.limit ? (lv.bytes / lv.limit) * 100 : 0)}%` }}
              />
            </div>
          </div>
          {lv.files.length === 0 ? (
            <p className="py-6 text-center text-sm text-mist">此层尚无 SSTable 礁盘</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {lv.files.map((f) => (
                <Card key={f.number} f={f} highlight={hotFiles.includes(f.number)} />
              ))}
            </div>
          )}
        </div>
      ))}
      <div className="rounded-xl border border-dashed border-white/10 px-4 py-3 text-sm text-mist">
        L3–L6 深渊汇总 · {deepFiles} files · {fmtBytes(deepBytes)}
      </div>
    </section>
  );
}
