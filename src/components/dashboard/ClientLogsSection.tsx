import { useState, useMemo } from "react";
import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { LogTable } from "@/components/ui/LogTable";
import type { LogTableEntry } from "@/components/ui/LogTable";
import { Pagination } from "@/components/ui/Pagination";
import { Icons } from "@/components/ui/Icons";
import { logger, formatPercent } from "@/services/logger";

const PAGE_SIZE = 50;

export function ClientLogsSection() {
  const [logLevel, setLogLevel] = useState<"debug" | "info" | "warn" | "error">("info");
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [page, setPage] = useState(0);

  const metrics = logger.getApiMetrics();

  // Get all entries at this level, then apply search filter
  const allEntries = logger.getEntries(logLevel, 500);
  const filtered = useMemo(() => {
    const term = search.toLowerCase();
    const list = term
      ? allEntries.filter(
          (e) =>
            e.message.toLowerCase().includes(term) ||
            JSON.stringify(e.context ?? {}).toLowerCase().includes(term),
        )
      : allEntries;
    // Newest first
    return [...list].reverse();
  }, [allEntries, search]);

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const pageEntries: LogTableEntry[] = filtered
    .slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE)
    .map((e) => ({
      timestamp: e.timestamp,
      level: e.level,
      message: e.message,
      source: e.context?.caller as string | undefined,
      detail: e.context,
    }));

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(0);
    setSearch(searchInput);
  };

  const clearFilters = () => {
    setLogLevel("info");
    setSearch("");
    setSearchInput("");
    setPage(0);
  };

  return (
    <section>
      <SectionHeader title="Client Observability" description="Frontend logs and API call metrics" />

      {/* Summary cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
        <StatCard
          title="API Calls"
          value={metrics.totalCalls}
          subtitle={`Avg ${metrics.avgDurationMs}ms`}
          color="blue"
          icon={Icons.connection}
        />
        <StatCard
          title="Error Rate"
          value={formatPercent(metrics.errorRate)}
          subtitle={metrics.errorRate > 0.05 ? "Above threshold" : "Healthy"}
          color={metrics.errorRate > 0.05 ? "red" : "green"}
          icon={Icons.audit}
        />
        <StatCard
          title="Log Entries"
          value={filtered.length}
          subtitle={`${logLevel}+ level${search ? " (filtered)" : ""}`}
          color="purple"
          icon={Icons.log}
        />
        <StatCard
          title="Slow Requests"
          value={metrics.recent.filter((m) => m.durationMs > 2000).length}
          subtitle="> 2 000 ms"
          color={metrics.recent.some((m) => m.durationMs > 2000) ? "orange" : "green"}
          icon={Icons.clock}
        />
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm mb-4">
        <div className="px-4 py-3 flex flex-wrap items-center gap-3">
          <select
            value={logLevel}
            onChange={(e) => {
              setLogLevel(e.target.value as typeof logLevel);
              setPage(0);
            }}
            className="text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-1.5 bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200"
          >
            <option value="debug">Debug+</option>
            <option value="info">Info+</option>
            <option value="warn">Warn+</option>
            <option value="error">Errors only</option>
          </select>

          <form onSubmit={handleSearch} className="flex gap-2 flex-1 min-w-[200px]">
            <input
              type="text"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search logs..."
              className="flex-1 text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-1.5 bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200 placeholder-gray-400"
            />
            <button
              type="submit"
              className="text-sm px-3 py-1.5 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
            >
              Search
            </button>
          </form>

          {(logLevel !== "info" || search) && (
            <button
              onClick={clearFilters}
              className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 underline"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      {/* Log table (shared component) */}
      <LogTable entries={pageEntries} sourceLabel="Context">
        <Pagination
          page={page}
          totalPages={totalPages}
          total={filtered.length}
          pageSize={PAGE_SIZE}
          onPageChange={setPage}
        />
      </LogTable>

      {/* API metrics breakdown */}
      {metrics.recent.length > 0 && (
        <div className="mt-4 bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
              Recent API Calls
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
              Last {metrics.recent.length} tracked requests
            </p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-700/50">
                <tr>
                  <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-44">
                    Timestamp
                  </th>
                  <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-20">
                    Method
                  </th>
                  <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Path
                  </th>
                  <th className="text-left py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-20">
                    Status
                  </th>
                  <th className="text-right py-2.5 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-24">
                    Duration
                  </th>
                </tr>
              </thead>
              <tbody className="font-mono text-xs">
                {[...metrics.recent].reverse().map((m, i) => (
                  <tr
                    key={i}
                    className="border-b border-gray-100 dark:border-gray-700/50 last:border-0 hover:bg-gray-50 dark:hover:bg-gray-700/30"
                  >
                    <td className="py-2 px-4 text-gray-500 dark:text-gray-400 whitespace-nowrap">
                      {new Date(m.timestamp).toLocaleString()}
                    </td>
                    <td className="py-2 px-4">
                      <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-bold ${
                        m.method === "GET" ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                          : m.method === "POST" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300"
                          : m.method === "PUT" || m.method === "PATCH" ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300"
                          : m.method === "DELETE" ? "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300"
                          : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300"
                      }`}>{m.method}</span>
                    </td>
                    <td className="py-2 px-4 text-gray-700 dark:text-gray-300 truncate max-w-xs">
                      {m.path}
                    </td>
                    <td className="py-2 px-4">
                      <span
                        className={
                          m.status < 300
                            ? "text-green-600"
                            : m.status < 400
                              ? "text-blue-500"
                              : m.status < 500
                                ? "text-orange-500"
                                : "text-red-600 font-bold"
                        }
                      >
                        {m.status}
                      </span>
                    </td>
                    <td className="py-2 px-4 text-right">
                      <span className={m.durationMs > 2000 ? "text-red-600 font-bold" : "text-gray-600 dark:text-gray-400"}>
                        {m.durationMs}ms
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  );
}
