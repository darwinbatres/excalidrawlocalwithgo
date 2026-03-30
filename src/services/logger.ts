// =============================================================================
// Frontend Observability Service — Structured Logging, Metrics, Error Tracking
// =============================================================================

type LogLevel = "debug" | "info" | "warn" | "error";

interface LogEntry {
  level: LogLevel;
  message: string;
  timestamp: string;
  context?: Record<string, unknown>;
}

interface ApiMetric {
  method: string;
  path: string;
  status: number;
  durationMs: number;
  timestamp: string;
}

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

// In-memory ring buffer for recent logs (viewable from admin dashboard)
const MAX_LOG_ENTRIES = 500;
const MAX_API_METRICS = 200;

class Logger {
  private entries: LogEntry[] = [];
  private apiMetrics: ApiMetric[] = [];
  private minLevel: LogLevel = process.env.NODE_ENV === "production" ? "info" : "debug";

  private push(level: LogLevel, message: string, context?: Record<string, unknown>) {
    if (LOG_LEVELS[level] < LOG_LEVELS[this.minLevel]) return;

    const entry: LogEntry = {
      level,
      message,
      timestamp: new Date().toISOString(),
      context,
    };

    this.entries.push(entry);
    if (this.entries.length > MAX_LOG_ENTRIES) {
      this.entries = this.entries.slice(-MAX_LOG_ENTRIES);
    }

    // Also output to browser console with structured data
    const consoleFn = level === "error" ? console.error
      : level === "warn" ? console.warn
      : level === "debug" ? console.debug
      : console.info;

    consoleFn(`[${level.toUpperCase()}] ${message}`, context ?? "");
  }

  debug(message: string, context?: Record<string, unknown>) {
    this.push("debug", message, context);
  }

  info(message: string, context?: Record<string, unknown>) {
    this.push("info", message, context);
  }

  warn(message: string, context?: Record<string, unknown>) {
    this.push("warn", message, context);
  }

  error(message: string, context?: Record<string, unknown>) {
    this.push("error", message, context);
  }

  // Record an API call metric
  trackApi(method: string, path: string, status: number, durationMs: number) {
    const metric: ApiMetric = {
      method,
      path,
      status,
      durationMs,
      timestamp: new Date().toISOString(),
    };

    this.apiMetrics.push(metric);
    if (this.apiMetrics.length > MAX_API_METRICS) {
      this.apiMetrics = this.apiMetrics.slice(-MAX_API_METRICS);
    }

    // Log slow requests
    if (durationMs > 2000) {
      this.warn("Slow API call", { method, path, status, durationMs });
    }
  }

  // Get recent log entries (for admin dashboard)
  getEntries(level?: LogLevel, limit = 100): LogEntry[] {
    let filtered = this.entries;
    if (level) {
      const minLvl = LOG_LEVELS[level];
      filtered = filtered.filter((e) => LOG_LEVELS[e.level] >= minLvl);
    }
    return filtered.slice(-limit);
  }

  // Get API metrics summary
  getApiMetrics(): {
    recent: ApiMetric[];
    avgDurationMs: number;
    errorRate: number;
    totalCalls: number;
  } {
    const recent = this.apiMetrics.slice(-50);
    const total = this.apiMetrics.length;
    const errors = this.apiMetrics.filter((m) => m.status >= 400).length;
    const avgDuration =
      total > 0
        ? this.apiMetrics.reduce((sum, m) => sum + m.durationMs, 0) / total
        : 0;

    return {
      recent,
      avgDurationMs: Math.round(avgDuration),
      errorRate: total > 0 ? errors / total : 0,
      totalCalls: total,
    };
  }

  // Clear all entries
  clear() {
    this.entries = [];
    this.apiMetrics = [];
  }
}

// Singleton
export const logger = new Logger();

// =============================================================================
// Byte Formatting Utility
// =============================================================================

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function formatPercent(ratio: number): string {
  return `${(ratio * 100).toFixed(1)}%`;
}
