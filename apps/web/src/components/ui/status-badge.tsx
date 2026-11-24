import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type Tone = "neutral" | "success" | "warning" | "error" | "info" | "brand";

const STATUS_TONES: Record<string, Tone> = {
  paid: "success",
  published: "success",
  completed: "success",
  delivered: "success",
  collected: "success",
  confirmed: "info",
  open: "info",
  purchasing: "info",
  ready_to_ship: "info",
  ready_for_pickup: "info",
  shipped: "info",
  awaiting: "warning",
  awaiting_payment: "warning",
  awaiting_verification: "warning",
  awaiting_arrival: "warning",
  pending_confirmation: "warning",
  unpaid: "warning",
  draft: "neutral",
  closed: "neutral",
  unfulfilled: "neutral",
  cancelled: "error",
  canceled: "error",
  failed: "error",
  refunded: "neutral",
  partially_refunded: "warning",
};

function normalize(status: string) {
  return status.trim().toLowerCase().replace(/\s+/g, "_");
}

function labelize(status: string) {
  return status
    .trim()
    .replaceAll("_", " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function statusTone(status: string): Tone {
  return STATUS_TONES[normalize(status)] ?? "neutral";
}

export function StatusBadge({
  status,
  label,
  className,
}: {
  status: string;
  label?: string;
  className?: string;
}) {
  return (
    <Badge tone={statusTone(status)} className={cn(className)}>
      {label ?? labelize(status)}
    </Badge>
  );
}
