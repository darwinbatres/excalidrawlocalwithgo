import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { formatBytes } from "@/services/logger";
import type { SystemStatsResult } from "@/types";

function formatUptime(seconds: number) {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h ${m}m`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function ProcessSection({ stats }: { stats: SystemStatsResult }) {
  const { process: proc } = stats;
  return (
    <section>
      <SectionHeader title="Process" description="OS-level process metrics" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Process Uptime"
          value={formatUptime(proc.uptimeSeconds)}
          subtitle={`PID ${proc.pid}`}
          color="green"
          icon={Icons.clock}
        />
        <StatCard
          title="RSS Memory"
          value={formatBytes(proc.rss)}
          subtitle="Resident set size"
          color="orange"
          icon={Icons.server}
        />
        <StatCard
          title="Open File Descriptors"
          value={proc.openFDs}
          subtitle="Active FDs"
          color="blue"
          icon={Icons.connection}
        />
        <StatCard
          title="Started At"
          value={new Date(proc.startTime).toLocaleString()}
          subtitle="Process start time"
          color="purple"
          icon={Icons.clock}
        />
      </div>
    </section>
  );
}
