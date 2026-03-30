/**
 * Admin Dashboard — System statistics, storage, database health, observability
 */

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/router";
import Head from "next/head";
import { Layout } from "@/components/layout/Layout";
import { useApp } from "@/contexts/AppContext";
import { auditApi, orgApi, wsApi } from "@/services/api.client";
import { logger } from "@/services/logger";
import { Icons } from "@/components/ui/Icons";
import {
  OverviewSection,
  BuildInfoSection,
  ProcessSection,
  S3StorageSection,
  RuntimeSection,
  PoolSection,
  DatabaseHealthSection,
  TableStorageSection,
  WebSocketSection,
  ContainerSection,
  BruteForceSection,
  RequestMetricsSection,
  BackendLogsSection,
  ClientLogsSection,
  CRUDBreakdownSection,
  BackupInfoSection,
  LogSummarySection,
} from "@/components/dashboard";
import type { SystemStatsResult, OrgStatsResult, HubStats } from "@/types";

export default function SettingsPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading: authLoading, currentOrg } = useApp();
  const [stats, setStats] = useState<SystemStatsResult | null>(null);
  const [orgStats, setOrgStats] = useState<OrgStatsResult | null>(null);
  const [hubStats, setHubStats] = useState<HubStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(false);

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push("/");
    }
  }, [authLoading, isAuthenticated, router]);

  const fetchStats = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const start = performance.now();
      const [data, wsData, orgData] = await Promise.all([
        auditApi.systemStats(),
        wsApi.stats().catch(() => null),
        currentOrg ? orgApi.stats(currentOrg.id).catch(() => null) : Promise.resolve(null),
      ]);
      const duration = Math.round(performance.now() - start);
      logger.info("System stats loaded", { durationMs: duration });
      logger.trackApi("GET", "/api/v1/stats", 200, duration);
      setStats(data);
      setHubStats(wsData);
      setOrgStats(orgData);
      setLastRefresh(new Date());
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Unknown error";
      logger.error("Failed to load system stats", { error: msg });
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [currentOrg]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchStats();
    }
  }, [isAuthenticated, fetchStats]);

  useEffect(() => {
    if (!autoRefresh) return;
    const id = setInterval(fetchStats, 30_000);
    return () => clearInterval(id);
  }, [autoRefresh, fetchStats]);

  if (authLoading || !isAuthenticated) return null;

  return (
    <Layout>
      <Head>
        <title>Settings - Drawgo</title>
      </Head>
      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8 flex items-start justify-between">
          <div>
            <button
              onClick={() => router.push("/")}
              className="flex items-center gap-2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 mb-4 transition-colors"
            >
              {Icons.back}
              <span className="text-sm font-medium">Back to Dashboard</span>
            </button>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
              Admin Dashboard
            </h1>
            <p className="text-gray-500 dark:text-gray-400 mt-1">
              System health, storage analytics, and observability
            </p>
          </div>
          <div className="flex items-center gap-3">
            {lastRefresh && (
              <span className="text-xs text-gray-400">
                Updated {lastRefresh.toLocaleTimeString()}
              </span>
            )}
            <button
              onClick={fetchStats}
              disabled={loading}
              className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 transition-colors"
            >
              <span className={loading ? "animate-spin" : ""}>{Icons.refresh}</span>
              Refresh
            </button>
            <button
              onClick={() => setAutoRefresh(!autoRefresh)}
              className={`inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-colors ${
                autoRefresh
                  ? "border-green-400 dark:border-green-600 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300"
                  : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
              }`}
              title={autoRefresh ? "Auto-refresh every 30s (click to stop)" : "Enable auto-refresh (30s)"}
            >
              <span className={autoRefresh ? "animate-pulse" : ""}>●</span>
              {autoRefresh ? "Live" : "Auto"}
            </button>
          </div>
        </div>

        {loading && !stats ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600" />
          </div>
        ) : error ? (
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl p-6 text-center">
            <p className="text-red-600 dark:text-red-400">{error}</p>
            <button
              onClick={fetchStats}
              className="mt-4 text-sm text-red-600 dark:text-red-400 underline"
            >
              Try again
            </button>
          </div>
        ) : stats ? (
          <div className="space-y-8">
            <OverviewSection stats={stats} orgStats={orgStats} orgName={currentOrg?.name} />
            {stats.crudBreakdown && <CRUDBreakdownSection breakdown={stats.crudBreakdown} orgBreakdown={orgStats?.crudBreakdown} orgName={currentOrg?.name} />}
            {stats.backupInfo && <BackupInfoSection backup={stats.backupInfo} />}
            {stats.logs && <LogSummarySection logs={stats.logs} />}
            <BuildInfoSection stats={stats} />
            <ProcessSection stats={stats} />
            <S3StorageSection stats={stats} />
            <RuntimeSection stats={stats} />
            <PoolSection stats={stats} />
            <DatabaseHealthSection stats={stats} />
            <TableStorageSection tables={stats.database.tables} />
            <WebSocketSection hubStats={hubStats} />
            <ContainerSection container={stats.container} />
            {stats.bruteForce && <BruteForceSection stats={stats.bruteForce} />}
            {stats.requests && <RequestMetricsSection metrics={stats.requests} />}
            <BackendLogsSection />
            <ClientLogsSection />
          </div>
        ) : null}
      </div>
    </Layout>
  );
}
