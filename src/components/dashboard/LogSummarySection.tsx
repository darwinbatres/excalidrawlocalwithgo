import { SectionHeader } from "@/components/ui/SectionHeader";
import type { LogLevelSummary } from "@/types";

const levelConfig = [
  { key: "fatal" as const, label: "Fatal", bg: "bg-red-600", text: "text-red-600 dark:text-red-400", ring: "ring-red-200 dark:ring-red-800" },
  { key: "error" as const, label: "Error", bg: "bg-red-400", text: "text-red-500 dark:text-red-400", ring: "ring-red-200 dark:ring-red-800" },
  { key: "warn" as const, label: "Warn", bg: "bg-amber-400", text: "text-amber-600 dark:text-amber-400", ring: "ring-amber-200 dark:ring-amber-800" },
  { key: "info" as const, label: "Info", bg: "bg-blue-400", text: "text-blue-600 dark:text-blue-400", ring: "ring-blue-200 dark:ring-blue-800" },
  { key: "debug" as const, label: "Debug", bg: "bg-gray-400", text: "text-gray-600 dark:text-gray-400", ring: "ring-gray-200 dark:ring-gray-700" },
];

export function LogSummarySection({ logs }: { logs: LogLevelSummary }) {
  return (
    <section>
      <SectionHeader
        title="Log Level Summary"
        description={`${logs.total.toLocaleString()} log entries in the ring buffer`}
      />
      <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
        {/* Bar visualization */}
        {logs.total > 0 && (
          <div className="flex h-3 rounded-full overflow-hidden mb-4">
            {levelConfig.map(({ key, bg }) => {
              const pct = (logs[key] / logs.total) * 100;
              if (pct === 0) return null;
              return (
                <div
                  key={key}
                  className={`${bg} transition-all duration-300`}
                  style={{ width: `${pct}%` }}
                  title={`${key}: ${logs[key]}`}
                />
              );
            })}
          </div>
        )}
        {/* Level pills */}
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
          {levelConfig.map(({ key, label, text, ring }) => (
            <div
              key={key}
              className={`flex flex-col items-center p-3 rounded-lg ring-1 ${ring} ${logs[key] > 0 && (key === "fatal" || key === "error") ? "bg-red-50/50 dark:bg-red-900/10" : ""}`}
            >
              <span className={`text-2xl font-bold tabular-nums ${text}`}>
                {logs[key].toLocaleString()}
              </span>
              <span className="text-xs font-medium text-gray-500 dark:text-gray-400 mt-1">
                {label}
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
