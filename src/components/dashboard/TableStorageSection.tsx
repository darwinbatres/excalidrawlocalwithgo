import { useMemo } from "react";
import { formatBytes } from "@/services/logger";
import type { TableSizeInfo } from "@/types";

export function TableStorageSection({ tables }: { tables: TableSizeInfo[] }) {
  const sorted = useMemo(
    () => [...tables].sort((a, b) => b.totalBytes - a.totalBytes),
    [tables],
  );
  const maxSize = sorted.length > 0 ? sorted[0].totalBytes : 1;

  return (
    <section className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
      <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Table Storage Breakdown</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Per-table disk usage including data and indexes
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50 dark:bg-gray-700/50">
            <tr>
              <th className="text-left py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Table</th>
              <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Rows</th>
              <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Data</th>
              <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Indexes</th>
              <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Total</th>
              <th className="py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-32"></th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((t) => (
              <tr key={t.table} className="border-b border-gray-100 dark:border-gray-700/50 last:border-0">
                <td className="py-3 px-4">
                  <span className="font-mono text-sm text-gray-900 dark:text-white">{t.table}</span>
                </td>
                <td className="py-3 px-4 text-right font-mono text-sm text-gray-700 dark:text-gray-300">
                  {t.rowCount.toLocaleString()}
                </td>
                <td className="py-3 px-4 text-right font-mono text-sm text-gray-700 dark:text-gray-300">
                  {formatBytes(t.dataBytes)}
                </td>
                <td className="py-3 px-4 text-right font-mono text-sm text-gray-700 dark:text-gray-300">
                  {formatBytes(t.indexBytes)}
                </td>
                <td className="py-3 px-4 text-right font-mono text-sm font-medium text-gray-900 dark:text-white">
                  {formatBytes(t.totalBytes)}
                </td>
                <td className="py-3 px-4">
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div
                      className="bg-indigo-500 h-2 rounded-full transition-all"
                      style={{ width: `${maxSize > 0 ? (t.totalBytes / maxSize) * 100 : 0}%` }}
                    />
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
