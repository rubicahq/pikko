import { notFound } from "next/navigation";
import { ComponentPreviewClient } from "./preview-client";

export default function ComponentPreviewPage() {
  // Prefer APP_ENV / NEXT_PUBLIC_APP_ENV so local `next start` e2e can keep /dev open
  // (NODE_ENV is always "production" under `next start` / standalone).
  const appEnv =
    process.env.NEXT_PUBLIC_APP_ENV || process.env.APP_ENV || process.env.NODE_ENV;
  if (appEnv === "production") {
    notFound();
  }

  return <ComponentPreviewClient />;
}
