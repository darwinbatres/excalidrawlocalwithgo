import React, { useState, useEffect } from "react";
import type { BoardVersion } from "@/types";
import { boardApi, ApiError } from "@/services/api.client";
import { formatRelativeTime } from "@/lib/utils";
import { Button } from "@/components/ui/Button";
import { Spinner } from "@/components/ui/Spinner";
import { EmptyState } from "@/components/ui/EmptyState";

interface VersionHistoryProps {
  boardId: string;
  onClose: () => void;
  onRestore: (version: BoardVersion) => void;
}

export function VersionHistory({
  boardId,
  onClose,
  onRestore,
}: VersionHistoryProps) {
  const [versions, setVersions] = useState<BoardVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<BoardVersion | null>(
    null
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadVersions = async () => {
      setLoading(true);
      setError(null);

      try {
        const result = await boardApi.getVersions(boardId, { limit: 50 });
        setVersions(result.items);
        if (result.items.length > 0) {
          setSelectedVersion(result.items[0]);
        }
      } catch (err) {
        console.error("Failed to load versions:", err);
        setError(
          err instanceof ApiError ? err.message : "Failed to load versions"
        );
      } finally {
        setLoading(false);
      }
    };

    loadVersions();
  }, [boardId]);

  return (
    <div className="absolute top-0 right-0 h-full w-80 bg-white dark:bg-gray-900 border-l border-gray-100 dark:border-gray-800 shadow-xl z-20 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-gray-800">
        <h3 className="font-semibold text-gray-900 dark:text-gray-100">
          Version History
        </h3>
        <button
          onClick={onClose}
          className="p-1.5 text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-800 rounded-lg transition-colors"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      {/* Version list */}
      <div className="flex-1 overflow-y-auto">
        {loading ? (
          <div className="p-4 flex justify-center">
            <Spinner size="sm" />
          </div>
        ) : error ? (
          <div className="p-4 text-center text-red-600 dark:text-red-400">
            {error}
          </div>
        ) : versions.length === 0 ? (
          <EmptyState
            title="No versions yet"
            description="Start drawing to create your first version."
          />
        ) : (
          <div className="divide-y divide-gray-100 dark:divide-gray-800">
            {versions.map((version, index) => (
              <button
                key={version.id}
                onClick={() => setSelectedVersion(version)}
                className={`w-full text-left px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors ${
                  selectedVersion?.id === version.id
                    ? "bg-indigo-50 dark:bg-indigo-900/20 border-l-2 border-indigo-600"
                    : ""
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium text-gray-900 dark:text-gray-100">
                    v{version.version}
                    {index === 0 && (
                      <span className="ml-2 text-xs px-1.5 py-0.5 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-md">
                        Current
                      </span>
                    )}
                  </span>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {formatRelativeTime(version.createdAt)}
                  </span>
                </div>
                <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  Version {version.version}
                </div>
                {version.label && (
                  <div className="text-xs text-indigo-600 dark:text-indigo-400 mt-1">
                    {version.label}
                  </div>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Actions */}
      {selectedVersion && versions[0]?.id !== selectedVersion.id && (
        <div className="p-4 border-t border-gray-100 dark:border-gray-800">
          <Button onClick={() => onRestore(selectedVersion)} className="w-full">
            Restore v{selectedVersion.version}
          </Button>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-2 text-center">
            This will create a new version with the restored content.
          </p>
        </div>
      )}
    </div>
  );
}
