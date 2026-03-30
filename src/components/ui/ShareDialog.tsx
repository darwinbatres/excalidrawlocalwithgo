/**
 * ShareDialog — Generate, copy, and revoke share links for a board.
 *
 * Reusable modal for sharing boards with external users.
 * Supports VIEWER/EDITOR roles and optional expiration.
 */

import React, { useState, useCallback, useEffect } from "react";
import { Modal } from "@/components/ui/Modal";
import { Button } from "@/components/ui/Button";
import { Spinner } from "@/components/ui/Spinner";
import { shareApi } from "@/services/api.client";
import type { ShareLink } from "@/types";
import { toast } from "sonner";
import { formatApiError } from "@/lib/hooks";

interface ShareDialogProps {
  isOpen: boolean;
  onClose: () => void;
  boardId: string;
}

export function ShareDialog({ isOpen, onClose, boardId }: ShareDialogProps) {
  const [links, setLinks] = useState<ShareLink[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [role, setRole] = useState<"VIEWER" | "EDITOR">("VIEWER");

  const fetchLinks = useCallback(async () => {
    setLoading(true);
    try {
      const result = await shareApi.list(boardId);
      setLinks(result.items);
    } catch (err) {
      console.error("Failed to fetch share links:", err);
    } finally {
      setLoading(false);
    }
  }, [boardId]);

  useEffect(() => {
    if (isOpen) {
      fetchLinks();
    }
  }, [isOpen, fetchLinks]);

  const handleCreate = useCallback(async () => {
    setCreating(true);
    try {
      const link = await shareApi.create(boardId, role);
      setLinks((prev) => [link, ...prev]);
      toast.success("Share link created");
    } catch (err) {
      toast.error(formatApiError(err, "Failed to create link"));
    } finally {
      setCreating(false);
    }
  }, [boardId, role]);

  const handleRevoke = useCallback(
    async (linkId: string) => {
      try {
        await shareApi.revoke(boardId, linkId);
        setLinks((prev) => prev.filter((l) => l.id !== linkId));
        toast.success("Share link revoked");
      } catch (err) {
        toast.error(formatApiError(err, "Failed to revoke link"));
      }
    },
    [boardId]
  );

  const copyLink = useCallback((token: string) => {
    const url = `${window.location.origin}/boards/shared/${token}`;
    navigator.clipboard.writeText(url).then(() => {
      toast.success("Link copied to clipboard");
    });
  }, []);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Share Board" size="md">
      <div className="space-y-4">
        {/* Create new link */}
        <div className="flex items-center gap-3">
          <select
            value={role}
            onChange={(e) => setRole(e.target.value as "VIEWER" | "EDITOR")}
            className="flex-1 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm"
          >
            <option value="VIEWER">View only</option>
            <option value="EDITOR">Can edit</option>
          </select>
          <Button
            variant="primary"
            onClick={handleCreate}
            disabled={creating}
          >
            {creating ? "Creating..." : "Create Link"}
          </Button>
        </div>

        {/* Existing links */}
        <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
            Active Links
          </h4>

          {loading ? (
            <div className="flex justify-center py-4">
              <Spinner size="sm" />
            </div>
          ) : links.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
              No share links yet
            </p>
          ) : (
            <div className="space-y-2 max-h-60 overflow-y-auto">
              {links.map((link) => (
                <ShareLinkRow
                  key={link.id}
                  link={link}
                  onCopy={() => copyLink(link.token)}
                  onRevoke={() => handleRevoke(link.id)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </Modal>
  );
}

function ShareLinkRow({
  link,
  onCopy,
  onRevoke,
}: {
  link: ShareLink;
  onCopy: () => void;
  onRevoke: () => void;
}) {
  const isExpired =
    link.expiresAt && new Date(link.expiresAt) < new Date();

  return (
    <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-gray-50 dark:bg-gray-800/50">
      <div className="flex items-center gap-2 min-w-0">
        <span
          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
            link.role === "EDITOR"
              ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
              : "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300"
          }`}
        >
          {link.role === "EDITOR" ? "Editor" : "Viewer"}
        </span>
        <span className="text-xs text-gray-500 dark:text-gray-400 truncate">
          {link.token.slice(0, 12)}...
        </span>
        {isExpired && (
          <span className="text-xs text-red-500">Expired</span>
        )}
      </div>
      <div className="flex items-center gap-1">
        <button
          onClick={onCopy}
          className="p-1.5 text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 rounded transition-colors"
          title="Copy link"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
            />
          </svg>
        </button>
        <button
          onClick={onRevoke}
          className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded transition-colors"
          title="Revoke link"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
        </button>
      </div>
    </div>
  );
}
