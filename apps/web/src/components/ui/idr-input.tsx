"use client";

import * as React from "react";
import { formatIDRDigits, parseIDRInput } from "@/lib/api";
import { Input, type InputProps } from "@/components/ui/input";

export type IdrInputProps = Omit<InputProps, "value" | "onChange" | "type" | "inputMode"> & {
  value: number | null | undefined;
  onValueChange: (value: number | null) => void;
};

/** Controlled IDR amount input with id-ID thousand separators (dots). */
export const IdrInput = React.forwardRef<HTMLInputElement, IdrInputProps>(
  ({ value, onValueChange, className, ...props }, ref) => {
    const display =
      value == null || Number.isNaN(value) ? "" : formatIDRDigits(value);

    return (
      <Input
        {...props}
        ref={ref}
        inputMode="numeric"
        autoComplete="off"
        className={className}
        value={display}
        onChange={(e) => {
          const raw = e.target.value;
          if (raw.trim() === "") {
            onValueChange(null);
            return;
          }
          onValueChange(parseIDRInput(raw));
        }}
      />
    );
  },
);
IdrInput.displayName = "IdrInput";
