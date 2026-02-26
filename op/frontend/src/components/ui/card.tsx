"use client";

type CardProps = {
  title?: string;
  titleAction?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  variant?: "default" | "danger";
};

export function Card({
  title,
  titleAction,
  children,
  className = "",
  variant = "default",
}: CardProps) {
  const borderColor =
    variant === "danger" ? "border-red-200" : "border-gray-200";
  const titleColor =
    variant === "danger" ? "text-red-600" : "text-gray-900";

  return (
    <div className={`bg-white rounded-lg border ${borderColor} p-6 ${className}`}>
      {title && (
        <div className="flex items-center justify-between mb-4">
          <h2 className={`font-semibold ${titleColor}`}>{title}</h2>
          {titleAction}
        </div>
      )}
      {children}
    </div>
  );
}
