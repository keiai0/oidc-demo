"use client";

type AlertProps = {
  variant: "error" | "success" | "warning";
  children: React.ReactNode;
  className?: string;
};

const variantStyles = {
  error: "text-red-600 bg-red-50 border-red-200",
  success: "text-green-700 bg-green-50 border-green-200",
  warning: "text-yellow-800 bg-yellow-50 border-yellow-200",
};

export function Alert({ variant, children, className = "" }: AlertProps) {
  return (
    <div
      className={`text-sm border rounded p-3 mb-4 ${variantStyles[variant]} ${className}`}
    >
      {children}
    </div>
  );
}
