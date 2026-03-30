import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import type { BruteForceStats } from "@/types";

export function BruteForceSection({ stats }: { stats: BruteForceStats }) {
  return (
    <section>
      <SectionHeader title="Brute-Force Protection" description="Login attempt monitoring and IP lockouts" />
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <StatCard
          title="Tracked IPs"
          value={stats.trackedIPs}
          subtitle="IPs with recent failed attempts"
          color={stats.trackedIPs > 10 ? "orange" : "green"}
          icon={Icons.users}
        />
        <StatCard
          title="Locked IPs"
          value={stats.lockedIPs}
          subtitle="Currently blocked from login"
          color={stats.lockedIPs > 0 ? "red" : "green"}
          icon={Icons.audit}
        />
      </div>
    </section>
  );
}
