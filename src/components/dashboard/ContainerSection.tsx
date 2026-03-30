import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Icons } from "@/components/ui/Icons";
import { formatBytes, formatPercent } from "@/services/logger";
import type { ContainerStats } from "@/types";

function formatDuration(usec: number): string {
  if (usec < 1000) return `${usec}μs`;
  if (usec < 1_000_000) return `${(usec / 1000).toFixed(1)}ms`;
  return `${(usec / 1_000_000).toFixed(2)}s`;
}

export function ContainerSection({ container }: { container: ContainerStats }) {
  if (!container.isContainer && !container.memoryLimitBytes) return null;

  const hasNetwork = (container.networkRxBytes ?? 0) > 0 || (container.networkTxBytes ?? 0) > 0;
  const hasCpuUsage = (container.cpuUsageUsec ?? 0) > 0;
  const hasPids = (container.pidsCurrent ?? 0) > 0;

  return (
    <section>
      <SectionHeader
        title="Container Runtime"
        description={`Docker/cgroup metrics${container.containerId ? ` — ${container.containerId}` : ""}`}
      />

      {/* Row 1: Memory + CPU core metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {container.memoryLimitBytes && container.memoryLimitBytes > 0 ? (
          <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
            <div className="flex items-center gap-3 mb-4">
              <div className={`p-2 rounded-lg ${(container.memoryUsagePercent ?? 0) > 80 ? "bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400" : "bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"}`}>
                {Icons.server}
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Memory Usage</p>
                <p className="text-xl font-bold text-gray-900 dark:text-white">
                  {formatBytes(container.memoryUsageBytes ?? 0)} / {formatBytes(container.memoryLimitBytes)}
                </p>
              </div>
            </div>
            <ProgressBar
              value={container.memoryUsageBytes ?? 0}
              max={container.memoryLimitBytes}
              color="blue"
            />
            <p className="text-xs text-gray-400 mt-2">
              {formatPercent((container.memoryUsagePercent ?? 0) / 100)} utilized
              {container.memoryCacheBytes ? ` · ${formatBytes(container.memoryCacheBytes)} cached` : ""}
            </p>
          </div>
        ) : (
          <StatCard
            title="Memory Limit"
            value="Unlimited"
            subtitle="No cgroup memory limit set"
            color="green"
            icon={Icons.server}
          />
        )}
        <StatCard
          title="CPU Allocation"
          value={container.effectiveCpus ? `${container.effectiveCpus.toFixed(2)} cores` : "Unlimited"}
          subtitle={container.cpuQuota && container.cpuPeriod ? `${container.cpuQuota}/${container.cpuPeriod} μs` : "No CPU quota"}
          color="purple"
          icon={Icons.connection}
        />
        <StatCard
          title="Container"
          value={container.isContainer ? "Yes" : "No"}
          subtitle={container.containerId || "Bare metal / VM"}
          color={container.isContainer ? "blue" : "green"}
          icon={Icons.server}
        />
        <StatCard
          title="OOM Kills"
          value={container.oomKills ?? 0}
          subtitle={container.swapUsageBytes ? `Swap: ${formatBytes(container.swapUsageBytes)}` : "No swap usage"}
          color={(container.oomKills ?? 0) > 0 ? "red" : "green"}
          icon={Icons.shield}
        />
      </div>

      {/* Row 2: CPU usage, throttling, PIDs, Network — only if data available */}
      {(hasCpuUsage || hasPids || hasNetwork) && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mt-4">
          {hasCpuUsage && (
            <StatCard
              title="CPU Time"
              value={formatDuration(container.cpuUsageUsec ?? 0)}
              subtitle={`Throttled: ${formatDuration(container.throttledUsec ?? 0)} (${(container.throttledPeriods ?? 0).toLocaleString()} periods)`}
              color={(container.throttledPeriods ?? 0) > 100 ? "red" : "blue"}
              icon={Icons.clock}
            />
          )}
          {hasPids && (
            <StatCard
              title="PIDs"
              value={`${container.pidsCurrent ?? 0}${container.pidsLimit ? ` / ${container.pidsLimit}` : ""}`}
              subtitle={container.pidsLimit ? `${((container.pidsCurrent ?? 0) / container.pidsLimit * 100).toFixed(0)}% of limit` : "No PID limit"}
              color={(container.pidsCurrent ?? 0) > (container.pidsLimit ?? Infinity) * 0.8 ? "red" : "green"}
              icon={Icons.runtime}
            />
          )}
          {hasNetwork && (
            <>
              <StatCard
                title="Network RX"
                value={formatBytes(container.networkRxBytes ?? 0)}
                subtitle={`${(container.networkRxPackets ?? 0).toLocaleString()} packets`}
                color="blue"
                icon={Icons.connection}
              />
              <StatCard
                title="Network TX"
                value={formatBytes(container.networkTxBytes ?? 0)}
                subtitle={`${(container.networkTxPackets ?? 0).toLocaleString()} packets`}
                color="purple"
                icon={Icons.connection}
              />
            </>
          )}
        </div>
      )}
    </section>
  );
}
