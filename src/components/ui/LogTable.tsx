import React, { useState } from "react";

export interface LogTableEntry {
  timestamp: string;
  level: string;
  message: string;
  source?: string;
  detail?: Record<string, unknown>;
}

interface LogTableProps {
  entries: LogTableEntry[];
  loading?: boolean;
  sourceLabel?: string;
  children?: React.ReactNode;
}

const LEVEL_COLORS: Record<string, string> = {
  trace: "text-gray-400 bg-gray-100 dark:bg-gray-700",
  debug: "text-gray-400 bg-gray-100 dark:bg-gray-700",
  info: "text-blue-600 bg-blue-50 dark:bg-blue-900/30",
  warn: "text-orange-600 bg-orange-50 dark:bg-orange-900/30",
  warning: "text-orange-600 bg-orange-50 dark:bg-orange-900/30",
  error: "text-red-600 bg-red-50 dark:bg-red-900/30",
  fatal: "text-red-800 bg-red-100 dark:bg-red-900/50",
  panic: "text-red-800 bg-red-100 dark:bg-red-900/50",
};

export function LevelBadge({ level }: { level: string }) {
  return (
    <span
      className={`inline-block px-2 py-0.5 rounded-md text-xs font-semibold uppercase ${LEVEL_COLORS[level] || "text-gray-500"}`}
    >
      {level}
    </span>
  );
}

export function LogTable({ entries, loading, sourceLabel = "Source", children }: LogTableProps) {
  const [expandedRow, setExpandedRow] = useState<number | null>(null);

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-44">
                Timestamp
              </th>
              <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-20">
                Level
              </th>
              <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Message
              </th>
              <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-48">
                {sourceLabel}
              </th>
            </tr>
          </thead>
          <tbody className="font-mono text-xs">
            {entries.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-gray-400">
                  {loading ? "Loading..." : "No log entries found"}
                </td>
              </tr>
            ) : (
              entries.map((entry, i) => (
                <React.Fragment key={i}>
                  <tr
                    className="border-b border-gray-100 dark:border-gray-700/50 last:border-0 hover:bg-gray-50 dark:hover:bg-gray-700/30 cursor-pointer"
                    onClick={() => setExpandedRow(expandedRow === i ? null : i)}
                  >
                    <td className="py-2 px-4 text-gray-500 dark:text-gray-400 whitespace-nowrap">
                      {new Date(entry.timestamp).toLocaleString()}
                    </td>
                    <td className="py-2 px-4">
                      <LevelBadge level={entry.level} />
                    </td>
                    <td className="py-2 px-4 text-gray-700 dark:text-gray-300 max-w-md truncate">
                      {entry.message}
                    </td>
                    <td className="py-2 px-4 text-gray-400 dark:text-gray-500 truncate">
                      {entry.source || "—"}
                    </td>
                  </tr>
                  {expandedRow === i && entry.detail && Object.keys(entry.detail).length > 0 && (
                    <tr className="bg-gray-50 dark:bg-gray-700/20">
                      <td colSpan={4} className="px-6 py-3">
                        <pre className="text-xs text-gray-600 dark:text-gray-400 whitespace-pre-wrap break-all">
                          {JSON.stringify(entry.detail, null, 2)}
                        </pre>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>
      {children}
    </div>
  );
}
