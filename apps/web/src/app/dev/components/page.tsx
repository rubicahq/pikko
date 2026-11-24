import { notFound } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";

export default function ComponentPreviewPage() {
  if (process.env.NODE_ENV === "production" || process.env.NEXT_PUBLIC_APP_ENV === "production") {
    notFound();
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-10 px-4 py-10 md:px-8">
      <div>
        <h1 className="text-[30px] font-medium">Component preview</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Dev-only catalog of Pikko UI primitives. Not served in production.
        </p>
      </div>

      <section className="space-y-4">
        <h2 className="text-xl font-medium">Buttons</h2>
        <div className="flex flex-wrap gap-3">
          <Button>Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
          <Button variant="link">Text link</Button>
          <Button loading>Loading</Button>
        </div>
      </section>

      <section className="max-w-md space-y-3">
        <h2 className="text-xl font-medium">Inputs</h2>
        <div className="space-y-2">
          <Label htmlFor="demo">Email address</Label>
          <Input id="demo" placeholder="you@example.com" />
        </div>
        <div className="space-y-2">
          <Label>With error</Label>
          <Input error defaultValue="bad" />
          <p className="text-sm text-error">Enter a valid email address.</p>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-xl font-medium">Badges</h2>
        <div className="flex flex-wrap gap-2">
          <Badge>Draft</Badge>
          <Badge tone="success">Paid</Badge>
          <Badge tone="warning">Awaiting payment</Badge>
          <Badge tone="error">Failed</Badge>
          <Badge tone="info">Open</Badge>
          <Badge tone="brand">Current</Badge>
        </div>
      </section>

      <section className="max-w-md">
        <h2 className="mb-4 text-xl font-medium">Card</h2>
        <Card>
          <CardTitle>Card title</CardTitle>
          <CardDescription>
            White surface, 1px neutral border, radius 10, padding 20. Borders over shadows.
          </CardDescription>
        </Card>
      </section>
    </div>
  );
}
