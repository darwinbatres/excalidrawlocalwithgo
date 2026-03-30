package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/darwinbatres/drawgo/backend/internal/models"
	"github.com/darwinbatres/drawgo/backend/internal/pkg/apierror"
	"github.com/darwinbatres/drawgo/backend/internal/repository"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)

// OrgService handles organization and member business logic.
type OrgService struct {
	pool        *pgxpool.Pool
	orgs        repository.OrgRepo
	memberships repository.MembershipRepo
	users       repository.UserRepo
	audit       repository.AuditRepo
	access      *AccessService
	log         zerolog.Logger
}

// NewOrgService creates an OrgService.
func NewOrgService(
	pool *pgxpool.Pool,
	orgs repository.OrgRepo,
	memberships repository.MembershipRepo,
	users repository.UserRepo,
	audit repository.AuditRepo,
	access *AccessService,
	log zerolog.Logger,
) *OrgService {
	return &OrgService{
		pool:        pool,
		orgs:        orgs,
		memberships: memberships,
		users:       users,
		audit:       audit,
		access:      access,
		log:         log,
	}
}

// ValidateSlug checks slug format: 3-50 chars, lowercase alphanumeric + hyphens.
func ValidateSlug(slug string) *apierror.Error {
	if len(slug) < 3 || len(slug) > 50 {
		return apierror.ErrBadRequest.WithMessage("Slug must be between 3 and 50 characters")
	}
	if !slugRegex.MatchString(slug) {
		return apierror.ErrBadRequest.WithMessage("Slug must be lowercase alphanumeric with hyphens only")
	}
	return nil
}

// CreateOrg creates an organization and an OWNER membership in a transaction.
func (s *OrgService) CreateOrg(ctx context.Context, userID, name, slug string) (*models.Organization, *apierror.Error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	name = strings.TrimSpace(name)

	if apiErr := ValidateSlug(slug); apiErr != nil {
		return nil, apiErr
	}

	exists, err := s.orgs.SlugExists(ctx, slug)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to check slug")
		return nil, apierror.ErrInternal
	}
	if exists {
		return nil, apierror.ErrSlugTaken
	}

	// Transaction: create org + owner membership atomically
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to begin transaction")
		return nil, apierror.ErrInternal
	}
	defer tx.Rollback(ctx)

	org, err := s.orgs.CreateInTx(ctx, tx, name, slug)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to create org")
		return nil, apierror.ErrInternal
	}

	if _, err := s.memberships.CreateInTx(ctx, tx, org.ID, userID, models.OrgRoleOwner); err != nil {
		s.log.Error().Err(err).Msg("failed to create owner membership")
		return nil, apierror.ErrInternal
	}

	if err := tx.Commit(ctx); err != nil {
		s.log.Error().Err(err).Msg("failed to commit org creation")
		return nil, apierror.ErrInternal
	}

	s.logAuditAsync(userID, org.ID, models.AuditActionOrgCreate, "organization", org.ID)

	return org, nil
}

// ListOrgs returns all organizations the user belongs to.
func (s *OrgService) ListOrgs(ctx context.Context, userID string) ([]repository.OrgWithCounts, *apierror.Error) {
	orgs, err := s.orgs.ListByUser(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list orgs")
		return nil, apierror.ErrInternal
	}
	return orgs, nil
}

// UpdateOrg renames an organization. OWNER only.
func (s *OrgService) UpdateOrg(ctx context.Context, userID, orgID, name string) (*models.Organization, *apierror.Error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierror.ErrBadRequest.WithMessage("Name is required")
	}

	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleOwner); apiErr != nil {
		return nil, apiErr
	}

	org, err := s.orgs.Update(ctx, orgID, name)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to update org")
		return nil, apierror.ErrInternal
	}

	s.logAuditAsync(userID, orgID, models.AuditActionOrgUpdate, "organization", orgID)

	return org, nil
}

// DeleteOrg deletes an organization. OWNER only, must have >=2 orgs and 0 boards.
func (s *OrgService) DeleteOrg(ctx context.Context, userID, orgID string) *apierror.Error {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleOwner); apiErr != nil {
		return apiErr
	}

	// Must keep at least one org
	orgCount, err := s.memberships.CountByUser(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to count user orgs")
		return apierror.ErrInternal
	}
	if orgCount <= 1 {
		return apierror.ErrLastOrg
	}

	// Must have zero boards
	boardCount, err := s.orgs.BoardCount(ctx, orgID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to count boards")
		return apierror.ErrInternal
	}
	if boardCount > 0 {
		return apierror.ErrOrgHasBoards
	}

	if err := s.orgs.Delete(ctx, orgID); err != nil {
		s.log.Error().Err(err).Msg("failed to delete org")
		return apierror.ErrInternal
	}

	s.logAuditAsync(userID, orgID, models.AuditActionOrgDelete, "organization", orgID)

	return nil
}

// ListMembers returns all members of an org. VIEWER+ access required.
func (s *OrgService) ListMembers(ctx context.Context, userID, orgID string) ([]models.MembershipWithUser, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, userID, orgID, models.OrgRoleViewer); apiErr != nil {
		return nil, apiErr
	}

	members, err := s.memberships.ListByOrg(ctx, orgID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to list members")
		return nil, apierror.ErrInternal
	}
	return members, nil
}

// InviteMember adds a user to an org by email. ADMIN+ required.
func (s *OrgService) InviteMember(ctx context.Context, actorID, orgID, email string, role models.OrgRole) (*models.MembershipWithUser, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, actorID, orgID, models.OrgRoleAdmin); apiErr != nil {
		return nil, apiErr
	}

	// Cannot invite as OWNER
	if role == models.OrgRoleOwner {
		return nil, apierror.ErrBadRequest.WithMessage("Cannot invite as OWNER. Use ownership transfer instead.")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, apierror.ErrUserNotFound.WithMessage("User with this email not found. They must create an account first.")
	}

	exists, err := s.memberships.Exists(ctx, orgID, user.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to check membership")
		return nil, apierror.ErrInternal
	}
	if exists {
		return nil, apierror.ErrAlreadyMember
	}

	m, err := s.memberships.Create(ctx, orgID, user.ID, role)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to create membership")
		return nil, apierror.ErrInternal
	}

	s.logAuditAsync(actorID, orgID, models.AuditActionMemberInvite, "membership", m.ID)

	return &models.MembershipWithUser{
		Membership: *m,
		User:       user.ToPublic(),
	}, nil
}

// UpdateMemberRole changes a member's role. ADMIN+ required, cannot change OWNER.
func (s *OrgService) UpdateMemberRole(ctx context.Context, actorID, orgID, membershipID string, role models.OrgRole) (*models.Membership, *apierror.Error) {
	if _, apiErr := s.access.RequireOrgRole(ctx, actorID, orgID, models.OrgRoleAdmin); apiErr != nil {
		return nil, apiErr
	}

	if role == models.OrgRoleOwner {
		return nil, apierror.ErrBadRequest.WithMessage("Cannot set role to OWNER. Use ownership transfer instead.")
	}

	target, err := s.memberships.GetByID(ctx, membershipID)
	if err != nil {
		return nil, apierror.ErrMemberNotFound
	}
	if target.OrgID != orgID {
		return nil, apierror.ErrMemberNotFound
	}
	if target.Role == models.OrgRoleOwner {
		return nil, apierror.ErrCannotChangeOwner
	}

	updated, err := s.memberships.UpdateRole(ctx, membershipID, role)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to update member role")
		return nil, apierror.ErrInternal
	}

	s.logAuditAsync(actorID, orgID, models.AuditActionMemberUpdate, "membership", membershipID)

	return updated, nil
}

// RemoveMember removes a member from an org. ADMIN+ required, cannot remove OWNER.
func (s *OrgService) RemoveMember(ctx context.Context, actorID, orgID, membershipID string) *apierror.Error {
	if _, apiErr := s.access.RequireOrgRole(ctx, actorID, orgID, models.OrgRoleAdmin); apiErr != nil {
		return apiErr
	}

	target, err := s.memberships.GetByID(ctx, membershipID)
	if err != nil {
		return apierror.ErrMemberNotFound
	}
	if target.OrgID != orgID {
		return apierror.ErrMemberNotFound
	}
	if target.Role == models.OrgRoleOwner {
		return apierror.ErrCannotRemoveOwner
	}

	if err := s.memberships.Delete(ctx, membershipID); err != nil {
		s.log.Error().Err(err).Msg("failed to remove member")
		return apierror.ErrInternal
	}

	s.logAuditAsync(actorID, orgID, models.AuditActionMemberRemove, "membership", membershipID)

	return nil
}

func (s *OrgService) logAuditAsync(actorID, orgID, action, targetType, targetID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.audit.Log(ctx, orgID, &actorID, action, targetType, targetID, nil, nil, nil); err != nil {
			s.log.Error().Err(err).Str("action", action).Msg("failed to log audit event")
		}
	}()
}
