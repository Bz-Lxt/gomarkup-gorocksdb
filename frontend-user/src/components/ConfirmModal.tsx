type Props = {
  open: boolean;
  title: string;
  body: string;
  onCancel: () => void;
  onConfirm: () => void;
};

export function ConfirmModal({ open, title, body, onCancel, onConfirm }: Props) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/55 p-4">
      <div className="w-full max-w-md rounded-2xl border border-white/10 bg-trench p-5 shadow-sonar">
        <h3 className="font-display text-lg text-foam">{title}</h3>
        <p className="mt-2 text-sm text-mist">{body}</p>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            className="rounded-lg border border-white/10 px-3 py-1.5 text-sm text-mist hover:text-foam"
            onClick={onCancel}
          >
            取消
          </button>
          <button
            type="button"
            className="rounded-lg bg-coral px-3 py-1.5 text-sm text-abyss hover:brightness-110"
            onClick={onConfirm}
          >
            确认删除
          </button>
        </div>
      </div>
    </div>
  );
}
