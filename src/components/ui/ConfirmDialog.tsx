import React from "react";
import { Modal } from "@/components/ui/Modal";
import { Button } from "@/components/ui/Button";
import { ErrorAlert } from "@/components/ui/ErrorAlert";

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: React.ReactNode;
  detail?: string;
  confirmLabel?: string;
  confirmingLabel?: string;
  confirming?: boolean;
  error?: string | null;
  variant?: "danger" | "primary";
  size?: "sm" | "md";
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  detail,
  confirmLabel = "Confirm",
  confirmingLabel = "Processing...",
  confirming = false,
  error = null,
  variant = "danger",
  size = "sm",
}: ConfirmDialogProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} size={size}>
      <div className="space-y-4">
        {error && <ErrorAlert message={error} />}
        <p className="text-gray-600 dark:text-gray-400">{message}</p>
        {detail && (
          <p className="text-sm text-gray-500 dark:text-gray-500">{detail}</p>
        )}
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={onClose} disabled={confirming}>
            Cancel
          </Button>
          <Button
            variant={variant === "danger" ? "danger" : "primary"}
            onClick={onConfirm}
            disabled={confirming}
            isLoading={confirming}
          >
            {confirming ? confirmingLabel : confirmLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
