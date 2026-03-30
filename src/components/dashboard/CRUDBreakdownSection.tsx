import { SectionHeader } from "@/components/ui/SectionHeader";
import type { CRUDBreakdown } from "@/types";

/** Friendly labels for audit actions */
const actionLabels: Record<string, string> = {
  "board.create": "Board Created",
  "board.update": "Board Updated",
  "board.delete": "Board Deleted",
  "board.archive": "Board Archived",
  "board.restore": "Board Restored",
  "version.create": "Version Saved",
  "version.restore": "Version Restored",
  "asset.upload": "File Uploaded",
  "asset.delete": "File Deleted",
  "member.invite": "Member Invited",
  "member.remove": "Member Removed",
  "member.role_change": "Role Changed",
  "org.create": "Org Created",
  "org.update": "Org Updated",
  "org.delete": "Org Deleted",
  "share.create": "Share Created",
  "share.delete": "Share Revoked",
  "share.access": "Share Accessed",
  "user.login": "User Login",
  "user.logout": "User Logout",
  "user.register": "User Registered",
  "user.update": "User Updated",
  "backup.create": "Backup Created",
  "backup.restore": "Backup Restored",
  "backup.delete": "Backup Deleted",
  "backup.schedule_update": "Schedule Updated",
  "backup.download": "Backup Downloaded",
};

function formatAction(action: string): string {
  return actionLabels[action] ?? action;
}

/** Group actions by their domain prefix (e.g. "board", "user") */
function groupByDomain(byAction: { action: string; count: number }[]) {
  const groups: Record<string, { action: string; count: number }[]> = {};
  for (const item of byAction) {
    const domain = item.action.split(".")[0] ?? "other";
    if (!groups[domain]) groups[domain] = [];
    groups[domain].push(item);
  }
  return groups;
}

const domainColors: Record<string, string> = {
  board: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  version: "bg-indigo-100 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-300",
  asset: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  member: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  org: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300",
  share: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300",
  user: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300",
  backup: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
};

interface BreakdownGridProps {
  breakdown: CRUDBreakdown;
}

function BreakdownGrid({ breakdown }: BreakdownGridProps) {
  const groups = groupByDomain(breakdown.byAction ?? []);
  const sortedDomains = Object.keys(groups).sort();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {sortedDomains.map((domain) => {
        const items = groups[domain];
        const domainTotal = items.reduce((s, i) => s + i.count, 0);
        const badgeClass = domainColors[domain] ?? "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300";
        return (
          <div
            key={domain}
            className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 shadow-sm"
          >
            <div className="flex items-center justify-between mb-3">
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium capitalize ${badgeClass}`}>
                {domain}
              </span>
              <span className="text-sm font-semibold text-gray-900 dark:text-white">
                {domainTotal.toLocaleString()}
              </span>
            </div>
            <div className="space-y-2">
              {items
                .sort((a, b) => b.count - a.count)
                .map((item) => {
                  const pct = breakdown.totalEvents > 0 ? (item.count / breakdown.totalEvents) * 100 : 0;
                  return (
                    <div key={item.action} className="flex items-center justify-between text-sm">
                      <span className="text-gray-600 dark:text-gray-400 truncate mr-2">
                        {formatAction(item.action)}
                      </span>
                      <div className="flex items-center gap-2 shrink-0">
                        <div className="w-16 bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                          <div
                            className="bg-indigo-500 h-1.5 rounded-full"
                            style={{ width: `${Math.max(pct, 1)}%` }}
                          />
                        </div>
                        <span className="text-gray-900 dark:text-white font-medium tabular-nums w-12 text-right">
                          {item.count.toLocaleString()}
                        </span>
                      </div>
                    </div>
                  );
                })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

interface CRUDBreakdownSectionProps {
  breakdown: CRUDBreakdown;
  orgBreakdown?: CRUDBreakdown | null;
  orgName?: string;
}

export function CRUDBreakdownSection({ breakdown, orgBreakdown, orgName }: CRUDBreakdownSectionProps) {
  return (
    <section className="space-y-6">
      {orgBreakdown && (orgBreakdown.byAction?.length ?? 0) > 0 && (
        <>
          <SectionHeader
            title={`Activity — ${orgName || "Current Workspace"}`}
            description={`${orgBreakdown.totalEvents.toLocaleString()} events in this workspace`}
          />
          <BreakdownGrid breakdown={orgBreakdown} />
        </>
      )}
      <SectionHeader
        title="Activity — System-wide"
        description={`${breakdown.totalEvents.toLocaleString()} total events across all organizations`}
      />
      <BreakdownGrid breakdown={breakdown} />
    </section>
  );
}
