"use client";

export function DiffViewer({ patch }: { patch: string }) {
  const lines = patch.split("\n");

  const cls = (line: string) => {
    if (line.startsWith("+") && !line.startsWith("+++")) {
      return "bg-green-900/20 text-green-400";
    }
    if (line.startsWith("-") && !line.startsWith("---")) {
      return "bg-red-900/20 text-red-400";
    }
    if (line.startsWith("@@")) {
      return "bg-indigo-900/20 text-indigo-300";
    }
    if (
      line.startsWith("diff --git") ||
      line.startsWith("index ") ||
      line.startsWith("--- ") ||
      line.startsWith("+++ ")
    ) {
      return "text-muted";
    }
    return "text-base";
  };

  return (
    <div className="overflow-x-auto text-xs font-mono leading-6 bg-panel whitespace-pre">
      {lines.map((line, i) => (
        <div key={i} className={`px-4 py-0.5 ${cls(line)}`}>{line}</div>
      ))}
    </div>
  );
}

