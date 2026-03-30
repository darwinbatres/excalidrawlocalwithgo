import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { formatBytes } from "@/services/logger";
import type { SystemStatsResult } from "@/types";

export function RuntimeSection({ stats }: { stats: SystemStatsResult }) {
  const { runtime: rt } = stats;
  return (
    <section>
      <SectionHeader title="Go Runtime" description="Process memory, goroutines, and garbage collection" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Goroutines"
          value={rt.goroutines}
          subtitle={`${rt.numCPU} CPU cores`}
          color="blue"
          icon={Icons.connection}
        />
        <StatCard
          title="Heap Allocated"
          value={formatBytes(rt.heapAlloc)}
          subtitle={`${formatBytes(rt.heapSys)} system`}
          color="purple"
          icon={Icons.database}
        />
        <StatCard
          title="Heap In-Use"
          value={formatBytes(rt.heapInuse)}
          subtitle={`Stack: ${formatBytes(rt.stackInuse)}`}
          color="orange"
          icon={Icons.storage}
        />
        <StatCard
          title="GC Cycles"
          value={rt.numGC}
          subtitle={`Last pause: ${(rt.gcPauseNs / 1_000_000).toFixed(2)}ms`}
          color="green"
          icon={Icons.clock}
        />
      </div>
    </section>
  );
}
