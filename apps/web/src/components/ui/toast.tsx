"use client";

import * as React from "react";
import * as ToastPrimitive from "@radix-ui/react-toast";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

export const ToastProvider = ToastPrimitive.Provider;
export const ToastViewport = React.forwardRef<
  React.ElementRef<typeof ToastPrimitive.Viewport>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Viewport>
>(function ToastViewport({ className, ...props }, ref) {
  return (
    <ToastPrimitive.Viewport
      ref={ref}
      className={cn(
        "fixed bottom-4 right-4 z-[100] flex max-h-screen w-[min(100%,24rem)] flex-col gap-2 outline-none",
        className,
      )}
      {...props}
    />
  );
});

const toastVariants = {
  default: "border-border bg-background text-foreground",
  success: "border-success-border bg-success-bg text-success",
  error: "border-error-border bg-error-bg text-error",
} as const;

export const Toast = React.forwardRef<
  React.ElementRef<typeof ToastPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Root> & {
    tone?: keyof typeof toastVariants;
  }
>(function Toast({ className, tone = "default", ...props }, ref) {
  return (
    <ToastPrimitive.Root
      ref={ref}
      className={cn(
        "pointer-events-auto relative flex w-full items-start gap-3 rounded-lg border p-4 shadow-sm",
        toastVariants[tone],
        className,
      )}
      {...props}
    />
  );
});

export const ToastTitle = React.forwardRef<
  React.ElementRef<typeof ToastPrimitive.Title>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Title>
>(function ToastTitle({ className, ...props }, ref) {
  return (
    <ToastPrimitive.Title ref={ref} className={cn("text-sm font-medium", className)} {...props} />
  );
});

export const ToastDescription = React.forwardRef<
  React.ElementRef<typeof ToastPrimitive.Description>,
  React.ComponentPropsWithoutRef<typeof ToastPrimitive.Description>
>(function ToastDescription({ className, ...props }, ref) {
  return (
    <ToastPrimitive.Description
      ref={ref}
      className={cn("text-sm text-foreground-secondary", className)}
      {...props}
    />
  );
});

export function ToastClose({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof ToastPrimitive.Close>) {
  return (
    <ToastPrimitive.Close
      className={cn(
        "absolute right-2 top-2 rounded-md p-1 text-foreground-muted hover:bg-surface hover:text-foreground",
        className,
      )}
      toast-close=""
      {...props}
    >
      <X className="h-4 w-4" strokeWidth={1.75} />
    </ToastPrimitive.Close>
  );
}

type ToastMessage = {
  id: string;
  title: string;
  description?: string;
  tone?: keyof typeof toastVariants;
};

const ToastContext = React.createContext<{
  publish: (msg: Omit<ToastMessage, "id">) => void;
} | null>(null);

export function Toaster({ children }: { children?: React.ReactNode }) {
  const [items, setItems] = React.useState<ToastMessage[]>([]);

  const publish = React.useCallback((msg: Omit<ToastMessage, "id">) => {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    setItems((prev) => [...prev, { ...msg, id }]);
  }, []);

  return (
    <ToastContext.Provider value={{ publish }}>
      <ToastProvider swipeDirection="right">
        {children}
        {items.map((item) => (
          <Toast
            key={item.id}
            tone={item.tone}
            duration={4000}
            onOpenChange={(open) => {
              if (!open) setItems((prev) => prev.filter((t) => t.id !== item.id));
            }}
          >
            <div className="grid gap-1 pr-6">
              <ToastTitle>{item.title}</ToastTitle>
              {item.description ? <ToastDescription>{item.description}</ToastDescription> : null}
            </div>
            <ToastClose />
          </Toast>
        ))}
        <ToastViewport />
      </ToastProvider>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const ctx = React.useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within Toaster");
  }
  return ctx;
}
