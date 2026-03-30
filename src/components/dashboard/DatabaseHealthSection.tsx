import { SectionHeader } from "@/components/ui/SectionHeader";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Icons } from "@/components/ui/Icons";
import { formatPercent } from "@/services/logger";
import type { SystemStatsResult } from "@/types";

export function DatabaseHealthSection({ stats }: { stats: SystemStatsResult }) {
  const { database } = stats;
  const connPct = database.maxConnections > 0
    ? database.activeConnections / database.maxConnections
    : 0;
  const cacheOk = database.cacheHitRatio >= 0.99;

  return (
    <section>
      <SectionHeader title="Database Health" description="PostgreSQL connection pool and performance" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {/* Connections */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400">
              {Icons.connection}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Connections</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">
                {database.activeConnections} / {database.maxConnections}
              </p>
            </div>
          </div>
          <ProgressBar value={database.activeConnections} max={database.maxConnections} color="blue" />
          <p className="text-xs text-gray-400 mt-2">
            {formatPercent(connPct)} utilized
          </p>
        </div>

        {/* Cache Hit Ratio */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className={`p-2 rounded-lg ${cacheOk ? "bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400" : "bg-orange-50 text-orange-600 dark:bg-orange-900/20 dark:text-orange-400"}`}>
              {Icons.storage}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Cache Hit Ratio</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">
                {formatPercent(database.cacheHitRatio)}
              </p>
            </div>
          </div>
          <ProgressBar value={database.cacheHitRatio * 100} max={100} color={cacheOk ? "green" : "orange"} highIsGood />
          <p className="text-xs text-gray-400 mt-2">
            {cacheOk ? "Healthy — queries served from cache" : "Consider increasing shared_buffers"}
          </p>
        </div>

        {/* Uptime */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-cyan-50 text-cyan-600 dark:bg-cyan-900/20 dark:text-cyan-400">
              {Icons.clock}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Uptime</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">
                {database.uptime.trim()}
              </p>
            </div>
          </div>
          <p className="text-xs text-gray-400 mt-2">
            Since last PostgreSQL restart
          </p>
        </div>
      </div>
    </section>
  );
}
