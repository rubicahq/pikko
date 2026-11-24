import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-semibold transition-colors duration-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/40 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        /** Ink Black primary — Friendly Commerce Workspace default CTA */
        primary:
          "bg-ink text-foreground-inverse hover:bg-foreground active:bg-dark",
        /** Brand orange — selective emphasis only */
        brand:
          "bg-brand text-foreground-inverse hover:bg-brand-hover active:bg-brand-pressed",
        secondary:
          "border border-border bg-background text-foreground hover:bg-surface-hover",
        outline:
          "border border-border bg-background text-foreground hover:bg-surface-hover",
        ghost: "text-foreground hover:bg-surface-hover",
        destructive:
          "bg-error text-foreground-inverse hover:bg-error/90",
        link: "text-brand-dark underline-offset-4 hover:underline h-auto p-0 font-medium",
      },
      size: {
        default: "h-10 px-4 py-2",
        large: "h-11 px-5 text-base",
        sm: "h-9 px-3",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, loading, children, disabled, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        disabled={disabled || loading}
        {...props}
      >
        {loading ? (
          <>
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent" />
            {children}
          </>
        ) : (
          children
        )}
      </Comp>
    );
  },
);
Button.displayName = "Button";

export { buttonVariants };
