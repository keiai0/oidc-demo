"use client";

type PageHeaderProps = {
  title: string;
  action?: React.ReactNode;
  description?: string;
};

export function PageHeader({ title, action, description }: PageHeaderProps) {
  return (
    <div className="mb-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
        {action}
      </div>
      {description && (
        <p className="text-sm text-gray-500 mt-1">{description}</p>
      )}
    </div>
  );
}
