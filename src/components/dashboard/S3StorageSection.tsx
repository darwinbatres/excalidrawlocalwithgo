import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { formatBytes } from "@/services/logger";
import type { SystemStatsResult, BucketInfo } from "@/types";

export function S3StorageSection({ stats }: { stats: SystemStatsResult }) {
  const { storage: s3 } = stats;
  if (!s3.buckets || s3.buckets.length === 0) {
    return (
      <section>
        <SectionHeader title="S3 Storage" description="MinIO / S3-compatible object storage" />
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm text-center text-gray-400">
          No bucket data available
        </div>
      </section>
    );
  }
  return (
    <section>
      <SectionHeader title="S3 Storage" description="MinIO / S3-compatible object storage" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mb-4">
        <StatCard
          title="Total Objects"
          value={s3.totalObjects.toLocaleString()}
          subtitle={`Across ${s3.buckets.length} bucket${s3.buckets.length !== 1 ? "s" : ""}`}
          color="blue"
          icon={Icons.storage}
        />
        <StatCard
          title="Total Storage"
          value={formatBytes(s3.totalBytes)}
          subtitle="All buckets combined"
          color="purple"
          icon={Icons.bucket}
        />
        <StatCard
          title="Avg Object Size"
          value={s3.totalObjects > 0 ? formatBytes(Math.round(s3.totalBytes / s3.totalObjects)) : "0 B"}
          subtitle="Across all buckets"
          color="cyan"
          icon={Icons.database}
        />
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">Per-Bucket Breakdown</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 dark:bg-gray-700/50">
              <tr>
                <th className="text-left py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Bucket</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Objects</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Total Size</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Largest Object</th>
                <th className="py-3 px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider w-32"></th>
              </tr>
            </thead>
            <tbody>
              {s3.buckets.map((b: BucketInfo) => (
                <tr key={b.bucket} className="border-b border-gray-100 dark:border-gray-700/50 last:border-0">
                  <td className="py-3 px-4">
                    <span className="font-mono text-sm text-gray-900 dark:text-white">{b.bucket}</span>
                  </td>
                  <td className="py-3 px-4 text-right font-mono text-sm text-gray-700 dark:text-gray-300">
                    {b.objectCount.toLocaleString()}
                  </td>
                  <td className="py-3 px-4 text-right font-mono text-sm font-medium text-gray-900 dark:text-white">
                    {formatBytes(b.totalBytes)}
                  </td>
                  <td className="py-3 px-4 text-right font-mono text-sm text-gray-700 dark:text-gray-300">
                    {formatBytes(b.largestBytes)}
                  </td>
                  <td className="py-3 px-4">
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div
                        className="bg-purple-500 h-2 rounded-full transition-all"
                        style={{ width: `${s3.totalBytes > 0 ? (b.totalBytes / s3.totalBytes) * 100 : 0}%` }}
                      />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
