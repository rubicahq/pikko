import { notFound } from "next/navigation";
import { ComponentPreviewClient } from "./preview-client";

export default function ComponentPreviewPage() {
  if (process.env.NODE_ENV === "production" || process.env.NEXT_PUBLIC_APP_ENV === "production") {
    notFound();
  }

  return <ComponentPreviewClient />;
}
