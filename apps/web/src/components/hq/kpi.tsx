export function KPI({
  label,
  value,
  unavailable,
  hint,
}: {
  label: string;
  value: string;
  unavailable?: boolean;
  hint?: string;
}) {
  return (
    <div className="border-b border-border py-3 first:pt-0">
      <div className="text-xs font-medium uppercase tracking-wide text-foreground-muted">
        {label}
      </div>
      <div className={`mt-1 text-2xl font-medium tracking-tight ${unavailable ? "text-foreground-muted" : "text-foreground"}`}>
        {unavailable ? "Unavailable" : value}
      </div>
      {hint ? <div className="mt-0.5 text-xs text-foreground-muted">{hint}</div> : null}
    </div>
  );
}

export function RangePicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  const opts = [
    { k: "1h", label: "1h" },
    { k: "24h", label: "24h" },
    { k: "7d", label: "7d" },
    { k: "30d", label: "30d" },
  ];
  return (
    <div className="flex gap-1">
      {opts.map((o) => (
        <button
          key={o.k}
          type="button"
          onClick={() => onChange(o.k)}
          className={`rounded px-2.5 py-1 text-sm ${
            value === o.k
              ? "bg-ink text-white"
              : "bg-white text-foreground-secondary ring-1 ring-border hover:text-foreground"
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function EmptyOrUnavailable({
  title,
  detail,
}: {
  title: string;
  detail?: string;
}) {
  return (
    <div className="rounded border border-dashed border-border bg-white px-4 py-8 text-center">
      <p className="text-sm font-medium text-foreground">{title}</p>
      {detail ? <p className="mt-1 text-sm text-foreground-muted">{detail}</p> : null}
    </div>
  );
}
