import { useState } from "react";
import { api } from "../lib/api";
import { useStore } from "../store";

export function BenchPanel() {
  const metrics = useStore((s) => s.metrics);
  const push = useStore((s) => s.pushToast);
  const bench = (metrics.bench as Record<string, unknown>) || {};
  const [workers, setWorkers] = useState(8);
  const [qps, setQps] = useState(0);
  const [val, setVal] = useState(100);

  return (
    <section className="rounded-2xl border border-white/10 bg-trench/80 p-4">
      <h3 className="font-display text-lg">压测泵</h3>
      <label className="mt-3 block text-xs text-mist">
        并发 {workers}
        <input
          type="range"
          min={1}
          max={32}
          value={workers}
          onChange={(e) => setWorkers(Number(e.target.value))}
          className="w-full"
        />
      </label>
      <label className="block text-xs text-mist">
        目标 QPS {qps || "不限"}
        <input type="range" min={0} max={20000} step={500} value={qps} onChange={(e) => setQps(Number(e.target.value))} className="w-full" />
      </label>
      <label className="block text-xs text-mist">
        Value {val}B
        <input type="range" min={16} max={512} step={16} value={val} onChange={(e) => setVal(Number(e.target.value))} className="w-full" />
      </label>
      <div className="mt-3 flex gap-2">
        <button
          type="button"
          aria-label="启动压测"
          className="flex-1 rounded-lg bg-amber px-3 py-1.5 text-sm text-abyss"
          onClick={async () => {
            try {
              await api.benchStart(workers, qps, val);
              push("ok", "压测已启动");
            } catch (e) {
              push("err", e instanceof Error ? e.message : "启动失败");
            }
          }}
        >
          启动
        </button>
        <button
          type="button"
          aria-label="停止压测"
          className="flex-1 rounded-lg border border-white/10 px-3 py-1.5 text-sm"
          onClick={async () => {
            try {
              await api.benchStop();
              push("ok", "压测已停");
            } catch (e) {
              push("err", e instanceof Error ? e.message : "停止失败");
            }
          }}
        >
          停止
        </button>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-2 font-mono text-xs">
        <div className="rounded-lg bg-abyss p-2">
          <p className="text-mist">实测 QPS</p>
          <p className="text-xl text-sonar">{Number(bench.qps || 0).toFixed(0)}</p>
        </div>
        <div className="rounded-lg bg-abyss p-2">
          <p className="text-mist">累计 ops</p>
          <p className="text-xl">{Number(bench.ops || 0)}</p>
        </div>
      </div>
    </section>
  );
}
