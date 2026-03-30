package models

import (
	"encoding/json"
	"time"
)

// AuditEvent represents an immutable audit log entry.
type AuditEvent struct {
	ID         string          `json:"id"`
	OrgID      string          `json:"orgId"`
	ActorID    *string         `json:"actorId,omitempty"`
	Action     string          `json:"action"`
	TargetType string          `json:"targetType"`
	TargetID   string          `json:"targetId"`
	IP         *string         `json:"ip,omitempty"`
	UserAgent  *string         `json:"userAgent,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// Common audit action constants.
const (
	AuditActionAuthLogin   = "auth.login"
	AuditActionAuthLogout  = "auth.logout"
	AuditActionAuthFailed  = "auth.failed"

	AuditActionBoardCreate  = "board.create"
	AuditActionBoardUpdate  = "board.update"
	AuditActionBoardDelete  = "board.delete"
	AuditActionBoardView    = "board.view"
	AuditActionBoardArchive = "board.archive"

	AuditActionVersionCreate  = "version.create"
	AuditActionVersionRestore = "version.restore"

	AuditActionOrgCreate = "org.create"
	AuditActionOrgUpdate = "org.update"
	AuditActionOrgDelete = "org.delete"

	AuditActionMemberInvite = "member.invite"
	AuditActionMemberUpdate = "member.update"
	AuditActionMemberRemove = "member.remove"

	AuditActionShareCreate = "share.create"
	AuditActionShareRevoke = "share.revoke"
	AuditActionShareAccess = "share.access"

	AuditActionFileUpload = "file.upload"
	AuditActionFileDelete = "file.delete"

	AuditActionBackupCreate         = "backup.create"
	AuditActionBackupRestore        = "backup.restore"
	AuditActionBackupDelete         = "backup.delete"
	AuditActionBackupScheduleUpdate = "backup.schedule_update"
)
