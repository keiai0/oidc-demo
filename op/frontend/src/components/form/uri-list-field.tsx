"use client";

type URIListFieldProps = {
  label: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
};

export function URIListField({
  label,
  values,
  onChange,
  placeholder = "https://example.com/callback",
}: URIListFieldProps) {
  const updateAt = (index: number, value: string) => {
    const updated = [...values];
    updated[index] = value;
    onChange(updated);
  };

  const removeAt = (index: number) => {
    onChange(values.filter((_, i) => i !== index));
  };

  const add = () => {
    onChange([...values, ""]);
  };

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}
      </label>
      {values.map((uri, i) => (
        <div key={i} className="flex gap-2 mb-2">
          <input
            value={uri}
            onChange={(e) => updateAt(i, e.target.value)}
            placeholder={placeholder}
            className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {values.length > 1 && (
            <button
              type="button"
              onClick={() => removeAt(i)}
              className="px-2 text-red-500 hover:text-red-700"
            >
              Remove
            </button>
          )}
        </div>
      ))}
      <button
        type="button"
        onClick={add}
        className="text-sm text-blue-600 hover:underline"
      >
        + Add URI
      </button>
    </div>
  );
}
