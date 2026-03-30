import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { formatBytes } from "@/services/logger";
import type { BackupStats } from "@/types";

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return "just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

const statusColors: Record<string, string> = {
  completed: "text-green-600 dark:text-green-400",
  failed: "text-red-600 dark:text-red-400",
  running: "text-blue-600 dark:text-blue-400",
};

export function BackupInfoSection({ backup }: { backup: BackupStats }) {
  return (
    <section>
      <SectionHeader
        title="Backup Status"
        description={backup.scheduleEnabled ? `Automated schedule: ${backup.scheduleCron || "enabled"}` : "No automated schedule configured"}
      />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Backups"
          value={backup.totalBackups}
          subtitle={backup.scheduleEnabled ? "Schedule active" : "Manual only"}
          color="indigo"
          icon={Icons.bucket}
        />
        {backup.lastBackupAt && (
          <StatCard
            title="Last Backup"
            value={timeAgo(backup.lastBackupAt)}
            subtitle={new Date(backup.lastBackupAt).toLocaleString()}
            color="blue"
            icon={Icons.clock}
          />
        )}
        {backup.lastBackupSizeBytes != null && (
          <StatCard
            title="Last Backup Size"
            value={formatBytes(backup.lastBackupSizeBytes)}
            color="purple"
            icon={Icons.storage}
          />
        )}
        {backup.lastBackupStatus && (
          <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
            <div className="flex items-center gap-4">
              <div className={`p-3 rounded-lg ${
                backup.lastBackupStatus === "completed"
                  ? "bg-green-50 text-green-600 dark:bg-green-900/20 dark:text-green-400"
                  : backup.lastBackupStatus === "failed"
                    ? "bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400"
                    : "bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400"
              }`}>
                {Icons.shield}
              </div>
              <div>
                <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Last Status</p>
                <p className={`text-2xl font-bold capitalize ${statusColors[backup.lastBackupStatus] ?? "text-gray-900 dark:text-white"}`}>
                  {backup.lastBackupStatus}
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </section>
  );
}
