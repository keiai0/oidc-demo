"use client";

type BadgeProps = {
  variant: "active" | "inactive";
  children: React.ReactNode;
};

const variantStyles = {
  active: "bg-green-100 text-green-700",
  inactive: "bg-gray-100 text-gray-600",
};

export function Badge({ variant, children }: BadgeProps) {
  return (
    <span
      className={`inline-block px-2 py-0.5 text-xs rounded ${variantStyles[variant]}`}
    >
      {children}
    </span>
  );
}
