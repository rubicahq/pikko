import Image from "next/image";
import Link from "next/link";
import { cn } from "@/lib/utils";

export function Wordmark({
  href = "/",
  className,
  size = "md",
}: {
  href?: string;
  className?: string;
  /** Visual height of the logo wordmark. */
  size?: "sm" | "md" | "lg";
}) {
  const heightClass = size === "lg" ? "h-8" : size === "sm" ? "h-5" : "h-7";

  return (
    <Link
      href={href}
      className={cn("inline-flex items-center", className)}
      aria-label="Pikko"
    >
      <Image
        src="/brand/pikko-logo.png"
        alt="Pikko"
        width={120}
        height={49}
        className={cn("w-auto", heightClass)}
        priority
      />
    </Link>
  );
}
