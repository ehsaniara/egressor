interface Props {
  data: string;
}

export function JsonViewer({ data }: Props) {
  let formatted = data;
  try {
    const parsed = JSON.parse(data);
    formatted = JSON.stringify(parsed, null, 2);
  } catch {
    // not JSON, show raw
  }

  return (
    <pre className="text-xs text-gray-300 bg-[#1a1a2e] p-3 rounded overflow-auto max-h-96 whitespace-pre-wrap break-all font-mono">
      {formatted}
    </pre>
  );
}
