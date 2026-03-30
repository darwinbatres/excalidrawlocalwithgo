import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Icons } from "@/components/ui/Icons";
import { formatPercent } from "@/services/logger";
import type { SystemStatsResult } from "@/types";

export function PoolSection({ stats }: { stats: SystemStatsResult }) {
  const { pool } = stats;
  const usedPct = pool.maxConns > 0 ? pool.acquiredConns / pool.maxConns : 0;
  return (
    <section>
      <SectionHeader title="Connection Pool" description="pgxpool connection-pool metrics" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400">
              {Icons.connection}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Pool Connections</p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">
                {pool.acquiredConns} / {pool.maxConns}
              </p>
            </div>
          </div>
          <ProgressBar value={pool.acquiredConns} max={pool.maxConns} color="blue" />
          <p className="text-xs text-gray-400 mt-2">
            {formatPercent(usedPct)} in use — {pool.idleConns} idle, {pool.constructingConns} constructing
          </p>
        </div>

        <StatCard
          title="Total Acquired"
          value={pool.acquireCount.toLocaleString()}
          subtitle={`${pool.emptyAcquireCount.toLocaleString()} waited (empty pool)`}
          color="purple"
          icon={Icons.database}
        />
        <StatCard
          title="Total Connections"
          value={pool.totalConns}
          subtitle={`${pool.canceledAcquireCount.toLocaleString()} canceled acquires`}
          color="cyan"
          icon={Icons.connection}
        />
      </div>
    </section>
  );
}
