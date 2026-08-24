import { useState } from "react";
import { api } from "../lib/api";
import { useStore } from "../store";
import { ConfirmModal } from "./ConfirmModal";

export function ConsolePanel() {
  const push = useStore((s) => s.pushToast);
  const lsm = useStore((s) => s.lsm);
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [scanOut, setScanOut] = useState<string>("");
  const [err, setErr] = useState<Record<string, string>>({});
  const [confirm, setConfirm] = useState(false);
  const [busy, setBusy] = useState(false);

  const validateKV = (needVal: boolean) => {
    const e: Record<string, string> = {};
    if (!key.trim()) e.key = "键必填";
    if (needVal && !value.trim()) e.value = "值必填";
    setErr(e);
    return Object.keys(e).length === 0;
  };

  const run = async (fn: () => Promise<unknown>, ok: string) => {
    setBusy(true);
    try {
      await fn();
      push("ok", ok);
    } catch (e) {
      push("err", e instanceof Error ? e.message : "失败");
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="rounded-2xl border border-white/10 bg-trench/80 p-4">
      <h3 className="font-display text-lg">手工探测</h3>
      <p className="text-xs text-mist">所有读写打到真实引擎，没有假数据。</p>
      <div className="mt-3 grid gap-2">
        <label className="text-xs text-mist">
          Key *
          <input
            className="mt-1 w-full rounded-lg border border-white/10 bg-abyss px-3 py-2 font-mono text-sm"
            value={key}
            onChange={(e) => setKey(e.target.value)}
          />
          {err.key && <span className="text-coral">{err.key}</span>}
        </label>
        <label className="text-xs text-mist">
          Value
          <input
            className="mt-1 w-full rounded-lg border border-white/10 bg-abyss px-3 py-2 font-mono text-sm"
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
          {err.value && <span className="text-coral">{err.value}</span>}
        </label>
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          disabled={busy}
          aria-label="写入键值"
          className="rounded-lg bg-sonar px-3 py-1.5 text-sm text-abyss disabled:opacity-40"
          onClick={() => validateKV(true) && run(() => api.put(key, value), "已写入")}
        >
          Put
        </button>
        <button
          type="button"
          disabled={busy}
          aria-label="读取键值"
          className="rounded-lg border border-sonar/40 px-3 py-1.5 text-sm"
          onClick={() =>
            validateKV(false) &&
            run(async () => {
              const r = await api.get(key);
              setValue(r.value);
            }, "已读取")
          }
        >
          Get
        </button>
        <button
          type="button"
          disabled={busy}
          aria-label="删除键"
          className="rounded-lg border border-coral/50 px-3 py-1.5 text-sm text-coral"
          onClick={() => validateKV(false) && setConfirm(true)}
        >
          Delete
        </button>
      </div>
      <div className="mt-4 grid grid-cols-2 gap-2">
        <input
          placeholder="scan start"
          className="rounded-lg border border-white/10 bg-abyss px-2 py-1.5 font-mono text-xs"
          value={start}
          onChange={(e) => setStart(e.target.value)}
        />
        <input
          placeholder="scan end"
          className="rounded-lg border border-white/10 bg-abyss px-2 py-1.5 font-mono text-xs"
          value={end}
          onChange={(e) => setEnd(e.target.value)}
        />
      </div>
      <button
        type="button"
        aria-label="范围扫描"
        className="mt-2 w-full rounded-lg border border-white/10 py-1.5 text-sm"
        onClick={() =>
          run(async () => {
            const r = await api.scan(start, end, 20);
            setScanOut(r.items.map((i) => `${i.key}=${i.value}`).join("\n") || "(empty)");
          }, "扫描完成")
        }
      >
        Scan
      </button>
      {scanOut && <pre className="mt-2 max-h-28 overflow-auto font-mono text-[11px] text-mist">{scanOut}</pre>}
      <div className="mt-4 flex flex-wrap gap-2">
        {(["demo", "production", "test"] as const).map((p) => (
          <button
            key={p}
            type="button"
            aria-label={`切换到${p}档`}
            className={`rounded-full px-3 py-1 text-xs ${
              lsm?.profile === p ? "bg-amber text-abyss" : "border border-white/10 text-mist"
            }`}
            onClick={() => run(() => api.profile(p), `档位 ${p}`)}
          >
            {p}
          </button>
        ))}
        <button type="button" aria-label="手动刷盘" className="rounded-full border border-white/10 px-3 py-1 text-xs" onClick={() => run(() => api.flush(), "Flush")}>
          Flush
        </button>
        <button type="button" aria-label="手动压缩" className="rounded-full border border-white/10 px-3 py-1 text-xs" onClick={() => run(() => api.compact(), "Compact")}>
          Compact
        </button>
      </div>
      <ConfirmModal
        open={confirm}
        title="删除确认"
        body={`将写入墓碑标记，键「${key}」不会立刻从磁盘消失，而是在 Compaction 时被洋流融化。`}
        onCancel={() => setConfirm(false)}
        onConfirm={() => {
          setConfirm(false);
          run(() => api.del(key), "已删除");
        }}
      />
    </section>
  );
}
