import { useStore } from "../store";

export function ToastStack() {
  const toasts = useStore((s) => s.toasts);
  const dismiss = useStore((s) => s.dismissToast);
  return (
    <div className="fixed right-4 top-4 z-50 flex w-80 max-w-[90vw] flex-col gap-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`flex items-start justify-between gap-3 rounded-lg border px-3 py-2 text-sm shadow-lg ${
            t.kind === "err"
              ? "border-coral/40 bg-coral/15 text-foam"
              : "border-sonar/40 bg-sonar/15 text-foam"
          }`}
        >
          <span>{t.text}</span>
          <button
            type="button"
            aria-label="关闭提示"
            className="text-mist hover:text-foam"
            onClick={() => dismiss(t.id)}
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
