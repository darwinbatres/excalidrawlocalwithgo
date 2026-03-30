import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import type { SystemStatsResult } from "@/types";

export function BuildInfoSection({ stats }: { stats: SystemStatsResult }) {
  const { build } = stats;
  return (
    <section>
      <SectionHeader title="Build Info" description="Server build metadata and version" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Version"
          value={build.version}
          color="indigo"
          icon={Icons.info}
        />
        <StatCard
          title="Commit SHA"
          value={build.commitSHA.length > 8 ? build.commitSHA.substring(0, 8) : build.commitSHA}
          subtitle={build.commitSHA}
          color="blue"
          icon={Icons.info}
        />
        <StatCard
          title="Build Time"
          value={build.buildTime !== "unknown" ? new Date(build.buildTime).toLocaleDateString() : build.buildTime}
          subtitle={build.buildTime !== "unknown" ? new Date(build.buildTime).toLocaleTimeString() : undefined}
          color="green"
          icon={Icons.clock}
        />
        <StatCard
          title="Go Version"
          value={build.goVersion}
          color="cyan"
          icon={Icons.server}
        />
      </div>
    </section>
  );
}
