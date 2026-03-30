/**
 * Application-wide constants and limits
 *
 * These constants define size limits, timeouts, and configuration values
 * used throughout the application. Centralizing these values:
 *
 * - Ensures consistent behavior across components
 * - Makes limits easy to adjust for scaling
 * - Provides documentation for architectural decisions
 * - Enables environment-based configuration
 */

// =============================================================================
// Storage Limits (in bytes)
// =============================================================================

/**
 * Maximum size for a single markdown card content.
 * Based on typical markdown document sizes and database field limits.
 * 100KB allows for extensive documentation while preventing abuse.
 */
export const MAX_MARKDOWN_CARD_SIZE = 1000 * 1024; // 1000 KB

/**
 * Maximum size for a single rich text card content (Tiptap JSON).
 * Tiptap JSON is more verbose than markdown, so allow larger size.
 * 200KB supports complex formatted documents with tables.
 */
export const MAX_RICH_TEXT_CARD_SIZE = 2000 * 1024; // 2000 KB

/**
 * Maximum size for a single embedded image (base64).
 * 5MB allows high-quality images while keeping board size manageable.
 * Consider compression for larger images.
 */
export const MAX_IMAGE_SIZE = 50 * 1024 * 1024; // 50 MB

/**
 * Maximum total board size (all elements + files + app state).
 * 50MB is a reasonable limit for a single board.
 * Users should create multiple boards for very large projects.
 */
export const MAX_BOARD_SIZE = 500 * 1024 * 1024; // 500 MB

/**
 * Warning threshold for board size.
 * Show warning when board reaches 80% of max size.
 */
export const BOARD_SIZE_WARNING_THRESHOLD = 0.8;

/**
 * Maximum thumbnail size (base64 data URL).
 * Thumbnails are auto-generated at 400x300 which typically produces ~50KB.
 */
export const MAX_THUMBNAIL_SIZE = 100 * 1024; // 100 KB

// =============================================================================
// Element Limits
// =============================================================================

/**
 * Maximum number of elements per board.
 * This includes all Excalidraw shapes, images, cards, and search text elements.
 * High limit to support complex diagrams while preventing performance issues.
 */
export const MAX_ELEMENTS_PER_BOARD = 10000;

/**
 * Maximum number of markdown cards per board.
 * Each card creates 2 elements (card + search text), so effective limit is lower.
 */
export const MAX_MARKDOWN_CARDS_PER_BOARD = 500;

/**
 * Maximum number of rich text cards per board.
 * Each card creates 2 elements (card + search text), so effective limit is lower.
 */
export const MAX_RICH_TEXT_CARDS_PER_BOARD = 500;

/**
 * Maximum number of images per board.
 * Images are the heaviest elements, so limit more aggressively.
 */
export const MAX_IMAGES_PER_BOARD = 200;

// =============================================================================
// Version History
// =============================================================================

/**
 * Maximum number of versions to retain per board.
 * Older versions are automatically pruned to manage storage.
 * 100 versions covers ~2 weeks of heavy editing (assuming 5-10 saves/day).
 */
export const MAX_VERSIONS_PER_BOARD = 50;

/**
 * Number of versions shown in the version history sidebar.
 * Pagination is used for viewing older versions.
 */
export const VERSIONS_PAGE_SIZE = 20;

// =============================================================================
// Autosave Configuration
// =============================================================================

/**
 * Autosave interval in milliseconds.
 * Saves are only triggered if there are actual changes.
 * Can be overridden via NEXT_PUBLIC_AUTOSAVE_INTERVAL_MS environment variable.
 */
export const DEFAULT_AUTOSAVE_INTERVAL_MS = 10000; // 10 seconds

/**
 * Debounce delay for change detection in milliseconds.
 * Prevents too-frequent hash calculations during rapid editing.
 */
export const CHANGE_DEBOUNCE_MS = 500;

/**
 * Minimum time between saves in milliseconds.
 * Prevents save spam even with frequent changes.
 */
export const MIN_SAVE_INTERVAL_MS = 3000; // 3 seconds

// =============================================================================
// Card Dimensions
// =============================================================================

/**
 * Default width for new markdown cards in pixels.
 */
export const DEFAULT_MARKDOWN_CARD_WIDTH = 400;

/**
 * Default height for new markdown cards in pixels.
 */
export const DEFAULT_MARKDOWN_CARD_HEIGHT = 300;

/**
 * Default width for new rich text cards in pixels.
 */
export const DEFAULT_RICH_TEXT_CARD_WIDTH = 400;

/**
 * Default height for new rich text cards in pixels.
 */
export const DEFAULT_RICH_TEXT_CARD_HEIGHT = 300;

/**
 * Minimum card dimension in pixels.
 */
export const MIN_CARD_DIMENSION = 100;

/**
 * Maximum card dimension in pixels.
 */
export const MAX_CARD_DIMENSION = 2000;

// =============================================================================
// UI Configuration
// =============================================================================

/**
 * Font size for search text elements.
 * Set to 1 to minimize the yellow highlight box Excalidraw draws on search matches.
 */
export const SEARCH_TEXT_FONT_SIZE = 1;

/**
 * Zoom threshold above which to limit zoom during search navigation.
 * Prevents extreme zoom when navigating to tiny search text elements.
 */
export const MAX_SEARCH_ZOOM = 3;

/**
 * Target zoom level after correcting extreme search zoom.
 */
export const TARGET_SEARCH_ZOOM = 1;

/**
 * Delay before zoom correction after search navigation.
 */
export const ZOOM_CORRECTION_DELAY_MS = 50;

// =============================================================================
// Thumbnail Configuration
// =============================================================================

/**
 * Maximum width or height for generated thumbnails.
 */
export const THUMBNAIL_MAX_DIMENSION = 400;

/**
 * Thumbnail dimensions (width x height).
 */
export const THUMBNAIL_WIDTH = 400;
export const THUMBNAIL_HEIGHT = 300;

// =============================================================================
// Rate Limiting
// =============================================================================

/**
 * Maximum API requests per window (for rate limiting).
 * Default: 100 requests per minute.
 */
export const RATE_LIMIT_MAX_REQUESTS = 100;

/**
 * Rate limit window duration in milliseconds.
 */
export const RATE_LIMIT_WINDOW_MS = 60 * 1000; // 1 minute

// =============================================================================
// Validation Helper Functions
// =============================================================================

/**
 * Check if content size is within limits for a markdown card.
 */
export function isMarkdownSizeValid(content: string): boolean {
  const size = new TextEncoder().encode(content).length;
  return size <= MAX_MARKDOWN_CARD_SIZE;
}

/**
 * Check if content size is within limits for a rich text card.
 */
export function isRichTextSizeValid(content: string): boolean {
  const size = new TextEncoder().encode(content).length;
  return size <= MAX_RICH_TEXT_CARD_SIZE;
}

/**
 * Check if an image is within size limits.
 */
export function isImageSizeValid(dataUrl: string): boolean {
  const size = new TextEncoder().encode(dataUrl).length;
  return size <= MAX_IMAGE_SIZE;
}

/**
 * Format bytes to human-readable string.
 * Shared utility for consistent formatting across the app.
 */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
}

/**
 * Get content size in bytes.
 */
export function getContentSize(content: string | object): number {
  const str = typeof content === "string" ? content : JSON.stringify(content);
  return new TextEncoder().encode(str).length;
}
