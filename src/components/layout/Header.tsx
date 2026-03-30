import React, { useState, useRef } from "react";
import Link from "next/link";
import { useRouter } from "next/router";
import { toast } from "sonner";
import { useApp } from "@/contexts/AppContext";
import { Button } from "@/components/ui/Button";
import { Modal } from "@/components/ui/Modal";
import { Input } from "@/components/ui/Input";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { ErrorAlert } from "@/components/ui/ErrorAlert";
import { useAsyncAction, useModal, useOutsideClick } from "@/lib/hooks";
import { slugify } from "@/lib/utils";

export function Header() {
  const {
    user,
    currentOrg,
    userOrgs,
    switchOrg,
    createOrg,
    renameOrg,
    deleteOrg,
    logout,
  } = useApp();
  const [showOrgDropdown, setShowOrgDropdown] = useState(false);
  const [showSettingsDropdown, setShowSettingsDropdown] = useState(false);
  const settingsDropdownRef = useRef<HTMLDivElement>(null);
  const router = useRouter();

  // Modal state for create, rename, delete
  const createModal = useModal();
  const renameModal = useModal<{ id: string; name: string }>();
  const deleteModal = useModal<{ id: string; name: string; boardCount: number }>();

  // Form state
  const [newOrgName, setNewOrgName] = useState("");
  const [newOrgSlug, setNewOrgSlug] = useState("");
  const [renameValue, setRenameValue] = useState("");

  // Async actions
  const createAction = useAsyncAction(async (name: string, slug: string) => {
    await createOrg(name, slug);
  });
  const renameAction = useAsyncAction(async (id: string, name: string) => {
    await renameOrg(id, name);
  });
  const deleteAction = useAsyncAction(async (id: string) => {
    await deleteOrg(id);
  });

  // Click-outside for settings dropdown
  useOutsideClick(settingsDropdownRef, () => setShowSettingsDropdown(false), showSettingsDropdown);

  // Don't render the header if user is not authenticated
  if (!user) {
    return null;
  }

  const handleNameChange = (name: string) => {
    setNewOrgName(name);
    setNewOrgSlug(slugify(name));
  };

  const handleCreateOrg = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newOrgName.trim() || !newOrgSlug.trim()) return;

    try {
      const orgName = newOrgName.trim();
      await createAction.run(orgName, newOrgSlug.trim());
      setNewOrgName("");
      setNewOrgSlug("");
      createModal.close();
      toast("Workspace created", { description: orgName });
    } catch {
      // error displayed by useAsyncAction
    }
  };

  const handleDeleteOrg = async () => {
    if (!deleteModal.data) return;

    try {
      const deletedName = deleteModal.data.name;
      await deleteAction.run(deleteModal.data.id);
      deleteModal.close();
      toast("Workspace deleted", { description: deletedName });
    } catch {
      // error displayed by useAsyncAction
    }
  };

  const openRenameModal = (org: { id: string; name: string }) => {
    setRenameValue(org.name);
    renameAction.clearError();
    renameModal.open(org);
    setShowOrgDropdown(false);
  };

  const handleRenameOrg = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!renameModal.data || !renameValue.trim()) return;

    try {
      const newName = renameValue.trim();
      await renameAction.run(renameModal.data.id, newName);
      setRenameValue("");
      renameModal.close();
      toast("Workspace renamed", { description: newName });
    } catch {
      // error displayed by useAsyncAction
    }
  };

  const openDeleteConfirm = (org: {
    id: string;
    name: string;
    boardCount?: number;
  }) => {
    deleteAction.clearError();
    deleteModal.open({
      id: org.id,
      name: org.name,
      boardCount: org.boardCount || 0,
    });
    setShowOrgDropdown(false);
  };

  // Can delete if: user owns the org, org has no boards, and user has more than 1 org
  const canDeleteOrg = (org: {
    id: string;
    role?: string;
    boardCount?: number;
  }) => {
    return (
      org.role === "OWNER" && (org.boardCount || 0) === 0 && userOrgs.length > 1
    );
  };

  // Can rename if user owns the org
  const canRenameOrg = (org: { role?: string }) => {
    return org.role === "OWNER";
  };

  return (
    <>
      <header className="h-14 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 flex items-center px-4 justify-between">
        <div className="flex items-center gap-4">
          <Link
            href="/"
            className="text-xl font-bold text-indigo-600 dark:text-indigo-400 flex items-center gap-2"
          >
            <img src="/favicon-32x32.png" alt="Drawgo" className="w-6 h-6" />
            Drawgo
          </Link>

          {currentOrg && (
            <div className="relative">
              <button
                onClick={() => setShowOrgDropdown(!showOrgDropdown)}
                className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
              >
                <span className="font-medium">{currentOrg.name}</span>
                <svg
                  className={`w-4 h-4 transition-transform ${
                    showOrgDropdown ? "rotate-180" : ""
                  }`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 9l-7 7-7-7"
                  />
                </svg>
              </button>

              {showOrgDropdown && (
                <div className="absolute top-full left-0 mt-1 w-72 bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 py-2 z-50 overflow-hidden">
                  <div className="px-4 py-2 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                    Workspaces
                  </div>
                  {(userOrgs || []).map((org) => (
                    <div
                      key={org.id}
                      className={`group flex items-center justify-between px-4 py-2.5 text-sm hover:bg-gray-50 dark:hover:bg-gray-700/50 ${
                        org.id === currentOrg.id
                          ? "text-indigo-600 dark:text-indigo-400 bg-indigo-50 dark:bg-indigo-900/20"
                          : "text-gray-700 dark:text-gray-300"
                      }`}
                    >
                      <button
                        onClick={() => {
                          switchOrg(org.id);
                          setShowOrgDropdown(false);
                        }}
                        className="flex-1 text-left flex items-center gap-2"
                      >
                        <span className="truncate">{org.name}</span>
                        {org.id === currentOrg.id && (
                          <svg
                            className="w-4 h-4 shrink-0"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                        )}
                      </button>
                      <div className="flex items-center gap-0.5">
                        {canRenameOrg(org) && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              openRenameModal(org);
                            }}
                            className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-all"
                            title="Rename workspace"
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
                                d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                              />
                            </svg>
                          </button>
                        )}
                        {canDeleteOrg(org) && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              openDeleteConfirm(org);
                            }}
                            className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-all"
                            title="Delete workspace"
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
                        )}
                      </div>
                    </div>
                  ))}
                  <div className="border-t border-gray-100 dark:border-gray-700 my-2" />
                  <button
                    onClick={() => {
                      setShowOrgDropdown(false);
                      createModal.open();
                    }}
                    className="w-full text-left px-4 py-2.5 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/20 flex items-center gap-2"
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
                        d="M12 4v16m8-8H4"
                      />
                    </svg>
                    Create workspace
                  </button>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex items-center gap-3">
          {user && (
            <>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {user.name || user.email}
              </span>

              {/* Settings Dropdown */}
              <div className="relative" ref={settingsDropdownRef}>
                <button
                  onClick={() => setShowSettingsDropdown(!showSettingsDropdown)}
                  className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
                  aria-label="Settings menu"
                  aria-expanded={showSettingsDropdown}
                  aria-haspopup="true"
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
                      d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                    />
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                  </svg>
                </button>

                {showSettingsDropdown && (
                  <div
                    className="absolute right-0 top-full mt-1 w-56 bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700 py-1 z-50 overflow-hidden animate-in fade-in slide-in-from-top-1 duration-150"
                    role="menu"
                    aria-orientation="vertical"
                  >
                    <div className="px-3 py-2 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Settings
                    </div>

                    <button
                      onClick={() => {
                        setShowSettingsDropdown(false);
                        router.push("/settings");
                      }}
                      className="w-full text-left px-3 py-2.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700/50 flex items-center gap-3 transition-colors"
                      role="menuitem"
                    >
                      <svg
                        className="w-4 h-4 text-gray-400"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                        />
                      </svg>
                      <div>
                        <p className="font-medium">Statistics</p>
                        <p className="text-xs text-gray-400 dark:text-gray-500">
                          Storage & usage stats
                        </p>
                      </div>
                    </button>

                    <div className="border-t border-gray-100 dark:border-gray-700 my-1" />

                    <div className="px-3 py-2 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Account
                    </div>

                    <button
                      onClick={() => {
                        setShowSettingsDropdown(false);
                        logout();
                      }}
                      className="w-full text-left px-3 py-2.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700/50 flex items-center gap-3 transition-colors"
                      role="menuitem"
                    >
                      <svg
                        className="w-4 h-4 text-gray-400"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
                        />
                      </svg>
                      <div>
                        <p className="font-medium">Sign out</p>
                        <p className="text-xs text-gray-400 dark:text-gray-500">
                          Log out of your account
                        </p>
                      </div>
                    </button>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </header>

      {/* Create Org Modal */}
      <Modal
        isOpen={createModal.isOpen}
        onClose={createModal.close}
        title="Create Workspace"
      >
        <form onSubmit={handleCreateOrg}>
          {createAction.error && (
            <ErrorAlert message={createAction.error} className="mb-4" />
          )}
          <Input
            label="Workspace name"
            value={newOrgName}
            onChange={(e) => handleNameChange(e.target.value)}
            placeholder="My Workspace"
            autoFocus
            disabled={createAction.loading}
          />
          <div className="mt-3">
            <Input
              label="URL slug"
              value={newOrgSlug}
              onChange={(e) =>
                setNewOrgSlug(
                  e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "")
                )
              }
              placeholder="my-workspace"
              disabled={createAction.loading}
            />
            <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
              Lowercase letters, numbers, and hyphens only
            </p>
          </div>
          <div className="mt-4 flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={createModal.close}
              type="button"
              disabled={createAction.loading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!newOrgName.trim() || !newOrgSlug.trim() || createAction.loading}
              isLoading={createAction.loading}
            >
              {createAction.loading ? "Creating..." : "Create"}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Workspace Confirmation */}
      <ConfirmDialog
        isOpen={deleteModal.isOpen}
        onClose={deleteModal.close}
        onConfirm={handleDeleteOrg}
        title="Delete Workspace"
        message={
          <>
            Are you sure you want to delete{" "}
            <strong className="text-gray-900 dark:text-gray-100">
              {deleteModal.data?.name}
            </strong>
            ?
          </>
        }
        detail="This action cannot be undone. All workspace settings and memberships will be permanently removed."
        confirmLabel="Delete Workspace"
        confirmingLabel="Deleting..."
        confirming={deleteAction.loading}
        error={deleteAction.error}
        variant="danger"
      />

      {/* Rename Workspace Modal */}
      <Modal
        isOpen={renameModal.isOpen}
        onClose={renameModal.close}
        title="Rename Workspace"
      >
        <form onSubmit={handleRenameOrg}>
          {renameAction.error && (
            <ErrorAlert message={renameAction.error} className="mb-4" />
          )}
          <Input
            label="Workspace name"
            value={renameValue}
            onChange={(e) => setRenameValue(e.target.value)}
            placeholder="My Workspace"
            autoFocus
            disabled={renameAction.loading}
          />
          <div className="mt-4 flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={renameModal.close}
              type="button"
              disabled={renameAction.loading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!renameValue.trim() || renameAction.loading}
              isLoading={renameAction.loading}
            >
              {renameAction.loading ? "Renaming..." : "Rename"}
            </Button>
          </div>
        </form>
      </Modal>
    </>
  );
}
