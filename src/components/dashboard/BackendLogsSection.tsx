import React, { useState, useEffect, useCallback, useMemo } from "react";
import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { LogTable } from "@/components/ui/LogTable";
import type { LogTableEntry } from "@/components/ui/LogTable";
import { Pagination } from "@/components/ui/Pagination";
import { logsApi } from "@/services/api.client";
import type { BackendLogEntry, LogLevelSummary } from "@/types";

export function BackendLogsSection() {
  const [logEntries, setLogEntries] = useState<BackendLogEntry[]>([]);
  const [summary, setSummary] = useState<LogLevelSummary | null>(null);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(false);
  const [level, setLevel] = useState("");
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const pageSize = 50;

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const [result, sum] = await Promise.all([
        logsApi.query({
          level: level || undefined,
          search: search || undefined,
          start: startDate ? new Date(startDate).toISOString() : undefined,
          end: endDate ? new Date(endDate + "T23:59:59").toISOString() : undefined,
          limit: pageSize,
          offset: page * pageSize,
        }),
        logsApi.summary(),
      ]);
      setLogEntries(result.items);
      setTotal(result.total);
      setSummary(sum);
    } catch {
      // silently fail — dashboard should still work
    } finally {
      setLoading(false);
    }
  }, [level, search, startDate, endDate, page]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const totalPages = Math.ceil(total / pageSize);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(0);
    setSearch(searchInput);
  };

  const clearFilters = () => {
    setLevel("");
    setSearch("");
    setSearchInput("");
    setStartDate("");
    setEndDate("");
    setPage(0);
  };

  const tableEntries: LogTableEntry[] = useMemo(
    () =>
      logEntries.map((entry) => ({
        timestamp: entry.timestamp,
        level: entry.level,
        message: entry.message,
        source: entry.caller,
        detail: entry.fields,
      })),
    [logEntries],
  );

  return (
    <section>
      <SectionHeader
        title="Backend Logs"
        description="Server-side log entries with pagination, search, and filtering"
      />

      {/* Summary cards */}
      {summary && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-4">
          <StatCard title="Total" value={summary.total} color="indigo" icon={Icons.log} />
          <StatCard title="Debug" value={summary.debug} color="blue" icon={Icons.log} />
          <StatCard title="Info" value={summary.info} color="green" icon={Icons.log} />
          <StatCard title="Warn" value={summary.warn} color="orange" icon={Icons.log} />
          <StatCard title="Error" value={summary.error} color="red" icon={Icons.log} />
          <StatCard title="Fatal" value={summary.fatal} color="red" icon={Icons.audit} />
        </div>
      )}

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm mb-4">
        <div className="px-4 py-3 flex flex-wrap items-center gap-3">
          <select
            value={level}
            onChange={(e) => { setLevel(e.target.value); setPage(0); }}
            className="text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-1.5 bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200"
          >
            <option value="">All Levels</option>
            <option value="debug">Debug+</option>
            <option value="info">Info+</option>
            <option value="warn">Warn+</option>
            <option value="error">Error+</option>
            <option value="fatal">Fatal only</option>
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

          <input
            type="date"
            value={startDate}
            onChange={(e) => { setStartDate(e.target.value); setPage(0); }}
            className="text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-1.5 bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200"
          />
          <span className="text-gray-400 text-sm">to</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => { setEndDate(e.target.value); setPage(0); }}
            className="text-sm border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-1.5 bg-white dark:bg-gray-700 text-gray-700 dark:text-gray-200"
          />

          {(level || search || startDate || endDate) && (
            <button
              onClick={clearFilters}
              className="text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 underline"
            >
              Clear
            </button>
          )}

          <button
            onClick={fetchLogs}
            disabled={loading}
            className="text-sm px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 transition-colors"
          >
            {loading ? "..." : "Refresh"}
          </button>
        </div>
      </div>

      {/* Log table (shared component) */}
      <LogTable entries={tableEntries} loading={loading} sourceLabel="Caller">
        <Pagination
          page={page}
          totalPages={totalPages}
          total={total}
          pageSize={pageSize}
          onPageChange={setPage}
        />
      </LogTable>
    </section>
  );
}
