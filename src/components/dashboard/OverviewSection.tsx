import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { formatBytes, formatPercent } from "@/services/logger";
import type { SystemStatsResult, OrgStatsResult } from "@/types";

interface OverviewSectionProps {
  stats: SystemStatsResult;
  orgStats?: OrgStatsResult | null;
  orgName?: string;
}

export function OverviewSection({ stats, orgStats, orgName }: OverviewSectionProps) {
  const { counts, database } = stats;
  const org = orgStats?.counts;

  return (
    <section className="space-y-6">
      {/* Workspace-scoped stats (what the current user sees) */}
      {org && (
        <>
          <SectionHeader
            title={`Workspace — ${orgName || "Current"}`}
            description="Counts scoped to your current organization"
          />
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="Boards"
              value={org.boardsActive}
              subtitle={`${org.boards} total · ${org.boardsArchived} archived`}
              color="blue"
              icon={Icons.boards}
            />
            <StatCard
              title="Versions"
              value={org.boardVersions.toLocaleString()}
              subtitle={`${org.boardAssets.toLocaleString()} asset${org.boardAssets !== 1 ? "s" : ""}`}
              color="purple"
              icon={Icons.database}
            />
            <StatCard
              title="Members"
              value={org.members}
              subtitle="Active members in workspace"
              color="green"
              icon={Icons.users}
            />
            <StatCard
              title="Share Links"
              value={org.shareLinksActive}
              subtitle={`${org.shareLinks} total${org.shareLinksExpired > 0 ? ` · ${org.shareLinksExpired} expired` : ""}`}
              color="orange"
              icon={Icons.audit}
            />
          </div>
        </>
      )}

      {/* System-wide stats */}
      <SectionHeader
        title="System-wide"
        description="Aggregate counts across all organizations"
      />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Boards"
          value={counts.boardsActive}
          subtitle={`${counts.boards} total · ${counts.boardsArchived} archived · ${counts.boardVersions.toLocaleString()} versions · ${counts.boardAssets.toLocaleString()} assets`}
          color="blue"
          icon={Icons.boards}
        />
        <StatCard
          title="Users"
          value={counts.users}
          subtitle={`${counts.accounts} OAuth account${counts.accounts !== 1 ? "s" : ""} · ${counts.refreshTokensActive} active session${counts.refreshTokensActive !== 1 ? "s" : ""}`}
          color="green"
          icon={Icons.users}
        />
        <StatCard
          title="Organizations"
          value={counts.organizations}
          subtitle={`${counts.memberships} membership${counts.memberships !== 1 ? "s" : ""}`}
          color="cyan"
          icon={Icons.server}
        />
        <StatCard
          title="Database Size"
          value={formatBytes(database.databaseSizeBytes)}
          subtitle={`Cache hit ${formatPercent(database.cacheHitRatio)}`}
          color="purple"
          icon={Icons.database}
        />
        <StatCard
          title="Audit Events"
          value={counts.auditEvents.toLocaleString()}
          subtitle={`${counts.shareLinksActive} active share link${counts.shareLinksActive !== 1 ? "s" : ""}${counts.shareLinksExpired > 0 ? ` · ${counts.shareLinksExpired} expired` : ""}`}
          color="orange"
          icon={Icons.audit}
        />
        <StatCard
          title="Backups"
          value={counts.backupsCompleted}
          subtitle={[
            counts.backupsFailed > 0 ? `${counts.backupsFailed} failed` : null,
            counts.backupsInProgress > 0 ? `${counts.backupsInProgress} in progress` : null,
            stats.backupInfo?.scheduleEnabled ? `Scheduled: ${stats.backupInfo.scheduleCron || "enabled"}` : "No schedule",
          ].filter(Boolean).join(" · ")}
          color="indigo"
          icon={Icons.bucket}
        />
      </div>
    </section>
  );
}
