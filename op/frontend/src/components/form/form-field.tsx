"use client";

import type { FieldError } from "react-hook-form";

type FormFieldProps = {
  label: string;
  error?: FieldError;
  children: React.ReactNode;
};

export function FormField({ label, error, children }: FormFieldProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}
      </label>
      {children}
      {error && (
        <p className="text-xs text-red-600 mt-1">{error.message}</p>
      )}
    </div>
  );
}
