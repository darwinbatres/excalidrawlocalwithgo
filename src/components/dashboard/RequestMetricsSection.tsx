import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import { formatPercent } from "@/services/logger";
import type { RequestMetricsSnapshot } from "@/types";

export function RequestMetricsSection({ metrics }: { metrics: RequestMetricsSnapshot }) {
  const statusEntries = Object.entries(metrics.statusCodes).sort(([a], [b]) => a.localeCompare(b));
  const methodEntries = Object.entries(metrics.methodCounts).sort(([, a], [, b]) => b - a);
  const detailEntries = Object.entries(metrics.statusDetail || {}).sort(([, a], [, b]) => b - a);
  const topEndpoints = metrics.topEndpoints || [];

  return (
    <section>
      <SectionHeader title="HTTP Request Metrics" description="In-memory request counters, latency percentiles, and error rates" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
        <StatCard
          title="Total Requests"
          value={metrics.totalRequests.toLocaleString()}
          subtitle={`${metrics.requestsPerSec.toFixed(1)} req/s`}
          color="blue"
          icon={Icons.connection}
        />
        <StatCard
          title="Server Errors (5xx)"
          value={metrics.totalErrors.toLocaleString()}
          subtitle={`Error rate: ${formatPercent(metrics.errorRate)}`}
          color={metrics.totalErrors > 0 ? "red" : "green"}
          icon={Icons.audit}
        />
        <StatCard
          title="Client Errors (4xx)"
          value={metrics.total4xx.toLocaleString()}
          subtitle="Bad requests, 404s, auth failures"
          color={metrics.total4xx > 0 ? "orange" : "green"}
          icon={Icons.audit}
        />
        <StatCard
          title="P50 / P95 / P99"
          value={`${metrics.latencyP50Ms.toFixed(1)}ms`}
          subtitle={`P95: ${metrics.latencyP95Ms.toFixed(1)}ms · P99: ${metrics.latencyP99Ms.toFixed(1)}ms`}
          color="purple"
          icon={Icons.clock}
        />
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Status Code Breakdown */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">Status Code Distribution</h3>
          {statusEntries.length === 0 ? (
            <p className="text-gray-400 text-sm">No requests recorded yet</p>
          ) : (
            <div className="space-y-3">
              {statusEntries.map(([code, count]) => {
                const pct = metrics.totalRequests > 0 ? count / metrics.totalRequests : 0;
                const color = code === "2xx" ? "bg-green-500" : code === "3xx" ? "bg-blue-500" :
                  code === "4xx" ? "bg-orange-500" : code === "5xx" ? "bg-red-500" : "bg-gray-500";
                return (
                  <div key={code}>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="font-mono text-gray-700 dark:text-gray-300">{code}</span>
                      <span className="text-gray-500">{count.toLocaleString()} ({formatPercent(pct)})</span>
                    </div>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div className={`${color} h-2 rounded-full transition-all`} style={{ width: `${pct * 100}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
        {/* Method Breakdown */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">HTTP Method Distribution</h3>
          {methodEntries.length === 0 ? (
            <p className="text-gray-400 text-sm">No requests recorded yet</p>
          ) : (
            <div className="space-y-3">
              {methodEntries.map(([method, count]) => {
                const pct = metrics.totalRequests > 0 ? count / metrics.totalRequests : 0;
                const color = method === "GET" ? "bg-blue-500" : method === "POST" ? "bg-green-500" :
                  method === "PUT" || method === "PATCH" ? "bg-orange-500" : method === "DELETE" ? "bg-red-500" : "bg-gray-500";
                return (
                  <div key={method}>
                    <div className="flex justify-between text-sm mb-1">
                      <span className="font-mono text-gray-700 dark:text-gray-300">{method}</span>
                      <span className="text-gray-500">{count.toLocaleString()} ({formatPercent(pct)})</span>
                    </div>
                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                      <div className={`${color} h-2 rounded-full transition-all`} style={{ width: `${pct * 100}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
      {/* Granular Status Codes + Top Endpoints */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
        {/* Granular Status Codes */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">Status Code Detail</h3>
          {detailEntries.length === 0 ? (
            <p className="text-gray-400 text-sm">No requests recorded yet</p>
          ) : (
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {detailEntries.map(([code, count]) => {
                const num = parseInt(code, 10);
                const color = num < 300 ? "text-green-600" : num < 400 ? "text-blue-600" : num < 500 ? "text-orange-600" : "text-red-600";
                return (
                  <div key={code} className="flex justify-between text-sm">
                    <span className={`font-mono font-medium ${color}`}>{code}</span>
                    <span className="text-gray-500 dark:text-gray-400">{count.toLocaleString()}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
        {/* Top Endpoints */}
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-6 shadow-sm">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">Top Endpoints</h3>
          {topEndpoints.length === 0 ? (
            <p className="text-gray-400 text-sm">No endpoint data yet</p>
          ) : (
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {topEndpoints.map((ep) => (
                <div key={ep.route} className="flex items-center justify-between text-sm gap-2">
                  <span className="font-mono text-gray-700 dark:text-gray-300 truncate flex-1" title={ep.route}>{ep.route}</span>
                  <span className="text-gray-500 whitespace-nowrap">{ep.count.toLocaleString()} · {ep.avgLatencyMs.toFixed(1)}ms</span>
                  {ep.errors > 0 && <span className="text-red-500 text-xs">{ep.errors} err</span>}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
