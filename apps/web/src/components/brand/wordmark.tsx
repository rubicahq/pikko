import Link from "next/link";
import { cn } from "@/lib/utils";

export function Wordmark({
  href = "/",
  className,
}: {
  href?: string;
  className?: string;
}) {
  return (
    <Link
      href={href}
      className={cn("text-lg font-medium tracking-tight text-foreground", className)}
    >
      Pikko
    </Link>
  );
}
