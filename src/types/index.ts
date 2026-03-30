// Core types for the Excalidraw Enterprise app
// These mirror the Go backend models

// We'll use a simplified type for elements since the internal types aren't exported
export type ExcalidrawElement = {
  id: string;
  type: string;
  x: number;
  y: number;
  width: number;
  height: number;
  [key: string]: unknown;
};

// ============ Enums ============
export type OrgRole = "OWNER" | "ADMIN" | "MEMBER" | "VIEWER";
export type BoardRole = "OWNER" | "EDITOR" | "VIEWER";

// ============ Core Entities ============

export interface User {
  id: string;
  email: string;
  name: string | null;
  image: string | null;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  createdAt: string;
  updatedAt: string;
}

export interface Membership {
  id: string;
  orgId: string;
  userId: string;
  role: OrgRole;
  user?: User;
  createdAt: string;
  updatedAt: string;
}

export interface Board {
  id: string;
  orgId: string;
  ownerId: string;
  title: string;
  description: string | null;
  tags: string[];
  isArchived: boolean;
  thumbnail: string | null; // Base64 data URL for preview
  currentVersionId: string | null;
  versionNumber: number;
  etag: string;
  createdAt: string;
  updatedAt: string;
}

// Stripped AppState - we don't want to save volatile fields
export interface PersistedAppState {
  viewBackgroundColor?: string;
  gridSize?: number | null;
  gridStep?: number;
  gridModeEnabled?: boolean;
  theme?: "light" | "dark";
  zenModeEnabled?: boolean;
  viewModeEnabled?: boolean;
  // Add other non-volatile fields as needed
}

export interface BoardVersion {
  id: string;
  boardId: string;
  version: number;
  createdById: string;
  createdAt: string;
  sceneJson: {
    elements: ExcalidrawElement[];
  };
  appStateJson: PersistedAppState | null;
  thumbnailUrl: string | null;
  label?: string; // Optional checkpoint label
}

export interface BoardPermission {
  id: string;
  boardId: string;
  membershipId: string;
  role: BoardRole;
}

export interface AuditEvent {
  id: string;
  orgId: string;
  actorId: string | null;
  action: string;
  targetType: string;
  targetId: string;
  ip: string | null;
  userAgent: string | null;
  metadata: Record<string, unknown> | null;
  createdAt: string;
}

// ============ Extended Types with Relations ============

export interface BoardWithDetails extends Board {
  owner?: User;
  org?: Organization;
  latestVersion?: BoardVersion;
}

export interface MembershipWithDetails extends Membership {
  user?: User;
  org?: Organization;
}

// ============ API Request/Response Types ============

export interface SaveBoardRequest {
  elements: ExcalidrawElement[];
  appState: Record<string, unknown>;
  clientEtag: string;
  label?: string; // Optional checkpoint label
}

export interface SaveBoardResponse {
  success: boolean;
  board: Board;
  version: BoardVersion;
  conflict?: boolean;
  latestEtag?: string;
}

export interface BoardListParams {
  search?: string;
  tag?: string;
  archived?: boolean;
  page?: number;
  pageSize?: number;
  sort?: "updatedAt" | "createdAt" | "title";
  sortDir?: "asc" | "desc";
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

// ============ Save Status ============

export type SaveStatus =
  | "idle"
  | "saving"
  | "saved"
  | "error"
  | "conflict"
  | "offline";

export interface SaveState {
  status: SaveStatus;
  lastSaved: string | null;
  error: string | null;
  conflictData?: {
    serverVersion: BoardVersion;
    localElements: ExcalidrawElement[];
  };
}

// ============ Files/Assets ============

export interface FileManifestEntry {
  id: string;
  mimeType: string;
  dataURL?: string;
  created?: number;
}

// ============ Draft Type ============

export interface LocalDraft {
  elements: ExcalidrawElement[];
  appState: PersistedAppState;
  timestamp: number;
}

// ============ Share Links ============

export interface ShareLink {
  id: string;
  boardId: string;
  token: string;
  role: BoardRole;
  expiresAt: string | null;
  createdAt: string;
  createdBy: string;
}

export interface SharedBoardData {
  board: Board & {
    latestVersion?: {
      id: string;
      version: number;
      sceneJson: unknown;
      appStateJson?: unknown;
      createdAt: string;
      createdById: string;
      label?: string;
    };
  };
  shareRole: BoardRole;
}

// ============ File Assets ============

export interface FileAsset {
  id: string;
  boardId: string;
  fileId: string;
  mimeType: string;
  sizeBytes: number;
  storageKey: string;
  sha256: string;
  createdAt: string;
}

export interface UploadedFile {
  fileId: string;
  storageKey: string;
  mimeType: string;
  sizeBytes: number;
  sha256: string;
  url: string;
}

export interface StorageInfo {
  boardId?: string;
  orgId?: string;
  totalBytes: number;
  assetBytes?: number;
  sceneBytes?: number;
  appStateBytes?: number;
  thumbnailBytes?: number;
  assetCount?: number;
  versionCount?: number;
}

// ============ System Stats Types ============

export interface TableSizeInfo {
  table: string;
  totalBytes: number;
  dataBytes: number;
  indexBytes: number;
  rowCount: number;
}

export interface DatabaseStats {
  databaseSizeBytes: number;
  tables: TableSizeInfo[];
  activeConnections: number;
  maxConnections: number;
  cacheHitRatio: number;
  uptime: string;
}

export interface SystemCounts {
  users: number;
  organizations: number;
  boards: number;
  boardsActive: number;
  boardsArchived: number;
  boardVersions: number;
  boardAssets: number;
  auditEvents: number;
  memberships: number;
  shareLinks: number;
  shareLinksActive: number;
  shareLinksExpired: number;
  accounts: number;
  refreshTokens: number;
  refreshTokensActive: number;
  backups: number;
  backupsCompleted: number;
  backupsFailed: number;
  backupsInProgress: number;
}

export interface AuditActionCount {
  action: string;
  count: number;
}

export interface CRUDBreakdown {
  byAction: AuditActionCount[];
  totalEvents: number;
}

export interface OrgCounts {
  boards: number;
  boardsActive: number;
  boardsArchived: number;
  boardVersions: number;
  boardAssets: number;
  members: number;
  auditEvents: number;
  shareLinks: number;
  shareLinksActive: number;
  shareLinksExpired: number;
}

export interface OrgStatsResult {
  orgId: string;
  orgName: string;
  counts: OrgCounts;
  crudBreakdown?: CRUDBreakdown;
}

export interface BackupStats {
  totalBackups: number;
  lastBackupAt?: string;
  lastBackupSizeBytes?: number;
  lastBackupStatus?: string;
  scheduleEnabled: boolean;
  scheduleCron?: string;
}

export interface SystemStatsResult {
  counts: SystemCounts;
  database: DatabaseStats;
  pool: PoolStats;
  runtime: RuntimeStats;
  storage: S3StorageStats;
  process: ProcessStats;
  container: ContainerStats;
  build: BuildInfo;
  requests?: RequestMetricsSnapshot;
  bruteForce?: BruteForceStats;
  crudBreakdown?: CRUDBreakdown;
  backupInfo?: BackupStats;
  logs?: LogLevelSummary;
}

export interface ContainerStats {
  isContainer: boolean;
  memoryLimitBytes?: number;
  memoryUsageBytes?: number;
  memoryUsagePercent?: number;
  cpuQuota?: number;
  cpuPeriod?: number;
  effectiveCpus?: number;
  containerId?: string;
  // CPU usage metrics from cgroup cpu.stat
  cpuUsageUsec?: number;
  throttledUsec?: number;
  throttledPeriods?: number;
  // Memory detail from cgroup
  memoryCacheBytes?: number;
  swapUsageBytes?: number;
  oomKills?: number;
  // PID limits
  pidsCurrent?: number;
  pidsLimit?: number;
  // Network I/O
  networkRxBytes?: number;
  networkTxBytes?: number;
  networkRxPackets?: number;
  networkTxPackets?: number;
}

export interface RequestMetricsSnapshot {
  totalRequests: number;
  totalErrors: number;
  total4xx: number;
  errorRate: number;
  uptimeSeconds: number;
  requestsPerSec: number;
  latencyP50Ms: number;
  latencyP95Ms: number;
  latencyP99Ms: number;
  statusCodes: Record<string, number>;
  statusDetail: Record<string, number>;
  methodCounts: Record<string, number>;
  topEndpoints: EndpointSnapshot[];
}

export interface EndpointSnapshot {
  route: string;
  count: number;
  avgLatencyMs: number;
  errors: number;
}

export interface PoolStats {
  acquireCount: number;
  acquiredConns: number;
  idleConns: number;
  totalConns: number;
  constructingConns: number;
  maxConns: number;
  emptyAcquireCount: number;
  canceledAcquireCount: number;
}

export interface RuntimeStats {
  goroutines: number;
  heapAlloc: number;
  heapSys: number;
  heapInuse: number;
  stackInuse: number;
  gcPauseNs: number;
  numGC: number;
  numCPU: number;
  goVersion: string;
}

export interface ProcessStats {
  pid: number;
  rss: number;
  openFDs: number;
  uptimeSeconds: number;
  startTime: string;
}

export interface S3StorageStats {
  buckets: BucketInfo[];
  totalBytes: number;
  totalObjects: number;
}

export interface BucketInfo {
  bucket: string;
  objectCount: number;
  totalBytes: number;
  largestBytes: number;
}

export interface BuildInfo {
  version: string;
  commitSHA: string;
  buildTime: string;
  goVersion: string;
}

export interface HubStats {
  activeRooms: number;
  totalClients: number;
  messagesIn: number;
  messagesOut: number;
}

export interface BruteForceStats {
  trackedIPs: number;
  lockedIPs: number;
}

// ============ Backend Log Types ============

export interface BackendLogEntry {
  timestamp: string;
  level: string;
  message: string;
  caller?: string;
  fields?: Record<string, unknown>;
  raw: string;
}

export interface LogLevelSummary {
  debug: number;
  info: number;
  warn: number;
  error: number;
  fatal: number;
  total: number;
}

// ============ Auth Types ============

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export interface AuthResponse {
  user: User;
  tokens: AuthTokens;
}

// ============ WebSocket Types ============

export interface WSMessage {
  type: string;
  payload: unknown;
  senderId?: string;
}

export interface CursorPayload {
  x: number;
  y: number;
  userId: string;
  name: string;
  color: string;
}

export interface ViewerInfo {
  userId: string;
  name: string;
  color: string;
  role: string;
  joinedAt: string;
  isAnon: boolean;
}

export interface PresencePayload {
  viewers: ViewerInfo[];
  count: number;
}

export interface WelcomePayload {
  viewer: ViewerInfo;
}

// ============ Go API Envelope ============

export interface ApiEnvelope<T> {
  data: T;
  error: null;
  meta?: {
    total: number;
    limit: number;
    offset: number;
  };
}

export interface ApiErrorEnvelope {
  data: null;
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
}
