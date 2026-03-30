package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
)

// ShareService handles share link lifecycle operations.
type ShareService struct {
	shares repository.ShareLinkRepo
	boards repository.BoardRepo
	audit  repository.AuditRepo
	access *AccessService
	log    zerolog.Logger
}

// NewShareService creates a ShareService.
func NewShareService(
	shares repository.ShareLinkRepo,
	boards repository.BoardRepo,
	audit repository.AuditRepo,
	access *AccessService,
	log zerolog.Logger,
) *ShareService {
	return &ShareService{
		shares: shares,
		boards: boards,
		audit:  audit,
		access: access,
		log:    log,
	}
}

// CreateShareLinkInput contains the parameters for creating a share link.
type CreateShareLinkInput struct {
	BoardID   string
	UserID    string
	Role      models.BoardRole
	ExpiresIn *time.Duration // nil = no expiry
}

// CreateShareLink generates a new share link for a board. Requires EDITOR+ on the board.
func (s *ShareService) CreateShareLink(ctx context.Context, input CreateShareLinkInput) (*models.ShareLink, *apierror.Error) {
	// Verify the user can edit the board
	board, apiErr := s.access.RequireBoardEdit(ctx, input.UserID, input.BoardID)
	if apiErr != nil {
		return nil, apiErr
	}

	// Only allow VIEWER or EDITOR roles for share links (not OWNER)
	if input.Role != models.BoardRoleViewer && input.Role != models.BoardRoleEditor {
		return nil, apierror.ErrBadRequest.WithMessage("Share link role must be VIEWER or EDITOR")
	}

	var expiresAt *time.Time
	if input.ExpiresIn != nil {
		t := time.Now().Add(*input.ExpiresIn)
		expiresAt = &t
	}

	link, err := s.shares.Create(ctx, input.BoardID, input.UserID, input.Role, expiresAt)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to create share link")
		return nil, apierror.ErrInternal
	}

	// Audit
	_ = s.audit.Log(ctx, board.OrgID, &input.UserID,
		models.AuditActionShareCreate, "share_link", link.ID, nil, nil,
		map[string]any{"boardId": input.BoardID, "role": string(input.Role)})

	return link, nil
}

// ListShareLinks returns all share links for a board. Requires EDITOR+ on the board.
func (s *ShareService) ListShareLinks(ctx context.Context, userID, boardID string) ([]models.ShareLink, *apierror.Error) {
	if _, apiErr := s.access.RequireBoardEdit(ctx, userID, boardID); apiErr != nil {
		return nil, apiErr
	}

	links, err := s.shares.ListByBoard(ctx, boardID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list share links")
		return nil, apierror.ErrInternal
	}

	return links, nil
}

// RevokeShareLink deletes a share link. Requires EDITOR+ on the associated board.
func (s *ShareService) RevokeShareLink(ctx context.Context, userID, linkID string) *apierror.Error {
	link, err := s.shares.GetByID(ctx, linkID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get share link")
		return apierror.ErrInternal
	}
	if link == nil {
		return apierror.ErrNotFound.WithMessage("Share link not found")
	}

	board, apiErr := s.access.RequireBoardEdit(ctx, userID, link.BoardID)
	if apiErr != nil {
		return apiErr
	}

	if err := s.shares.Delete(ctx, linkID); err != nil {
		s.log.Error().Err(err).Msg("failed to delete share link")
		return apierror.ErrInternal
	}

	_ = s.audit.Log(ctx, board.OrgID, &userID,
		models.AuditActionShareRevoke, "share_link", linkID, nil, nil,
		map[string]any{"boardId": link.BoardID})

	return nil
}

// SharedBoardData is the response payload for accessing a board via share token.
type SharedBoardData struct {
	Board     *models.BoardWithVersion `json:"board"`
	ShareRole models.BoardRole         `json:"shareRole"`
}

// GetSharedBoard retrieves a board via share token. No user authentication required.
// Validates token existence and expiry before granting access.
func (s *ShareService) GetSharedBoard(ctx context.Context, tok string) (*SharedBoardData, *apierror.Error) {
	link, err := s.shares.GetByToken(ctx, tok)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get share link by token")
		return nil, apierror.ErrInternal
	}
	if link == nil {
		return nil, apierror.ErrNotFound.WithMessage("Invalid or expired share link")
	}
	if link.IsExpired() {
		return nil, apierror.ErrNotFound.WithMessage("Invalid or expired share link")
	}

	bv, err := s.boards.GetByIDWithVersion(ctx, link.BoardID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get shared board")
		return nil, apierror.ErrInternal
	}

	// Audit anonymous share access (no actor)
	_ = s.audit.Log(ctx, bv.OrgID, nil,
		models.AuditActionShareAccess, "board", link.BoardID, nil, nil,
		map[string]any{"shareRole": string(link.Role)})

	return &SharedBoardData{
		Board:     bv,
		ShareRole: link.Role,
	}, nil
}

// ValidateShareToken checks if a share token is valid and returns the link details.
// Used by WebSocket auth to verify share-token-based connections.
func (s *ShareService) ValidateShareToken(ctx context.Context, tok string) (*models.ShareLink, bool) {
	link, err := s.shares.GetByToken(ctx, tok)
	if err != nil || link == nil {
		return nil, false
	}
	if link.IsExpired() {
		return nil, false
	}
	return link, true
}
