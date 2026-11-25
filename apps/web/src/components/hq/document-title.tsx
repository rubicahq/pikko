"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";

const HQ_TITLES: { prefix: string; label: string }[] = [
  { prefix: "/system-health/errors", label: "Error Logs" },
  { prefix: "/system-health/jobs", label: "Background Jobs" },
  { prefix: "/system-health/services", label: "Services" },
  { prefix: "/system-health/usage", label: "Usage & Costs" },
  { prefix: "/system-health/alerts", label: "Alerts" },
  { prefix: "/system-health", label: "System Health" },
  { prefix: "/auth", label: "HQ Sign in" },
];

/** HQ pages use the same `Menu Name | Pikko` pattern. */
export function HQDocumentTitle() {
  const pathname = usePathname();

  useEffect(() => {
    const match = HQ_TITLES.find(
      (t) => pathname === t.prefix || pathname.startsWith(t.prefix + "/"),
    );
    document.title = `${match?.label || "HQ"} | Pikko`;
  }, [pathname]);

  return null;
}
