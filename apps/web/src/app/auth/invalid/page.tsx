import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function AuthInvalidPage() {
  return (
    <div>
      <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
        This link has expired.
      </h1>
      <p className="mt-3 text-base text-foreground-secondary">
        Request a new sign-in link to continue.
      </p>
      <Button asChild className="mt-8 w-full" size="large">
        <Link href="/auth">Send a new link.</Link>
      </Button>
    </div>
  );
}
