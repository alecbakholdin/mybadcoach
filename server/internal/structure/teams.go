package structure

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	teamsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1/teamsv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/slugify"
)

// TeamService implements the TeamService Connect handler: teams, their
// memberships, and the ticket 05 team-record read.
type TeamService struct {
	base
}

var _ teamsv1connect.TeamServiceHandler = (*TeamService)(nil)

// NewTeamService builds a TeamService over the given database.
func NewTeamService(sqlDB *sql.DB, now func() time.Time) *TeamService {
	return &TeamService{base: newBase(sqlDB, now)}
}

// CreateTeam creates a team with a globally unique slug and makes the caller
// its owner.
func (s *TeamService) CreateTeam(
	ctx context.Context,
	req *connect.Request[teamsv1.CreateTeamRequest],
) (*connect.Response[teamsv1.CreateTeamResponse], error) {
	user, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	name, validationErr := requiredText(req.Msg.GetName(), "name")
	if validationErr != nil {
		return nil, validationErr
	}
	id, err := newID()
	if err != nil {
		return nil, internalError()
	}
	slug, err := uniqueSlug(
		slugify.Slug(name),
		id,
		func(candidate string) (bool, error) {
			count, countErr := s.queries.TeamSlugExists(ctx, candidate)
			return count > 0, countErr
		},
	)
	if err != nil {
		return nil, internalError()
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, internalError()
	}
	defer func() { _ = tx.Rollback() }()

	queries := s.queries.WithTx(tx)
	team, err := queries.CreateTeam(ctx, db.CreateTeamParams{
		ID:        id,
		Slug:      slug,
		Name:      name,
		CreatedAt: s.timestamp(),
	})
	if err != nil {
		return nil, internalError()
	}
	membershipID, err := newID()
	if err != nil {
		return nil, internalError()
	}
	if _, err := queries.CreateMembership(ctx, db.CreateMembershipParams{
		ID:        membershipID,
		UserID:    user.ID,
		TeamID:    team.ID,
		Role:      roleOwner,
		CreatedAt: s.timestamp(),
	}); err != nil {
		return nil, internalError()
	}
	if err := tx.Commit(); err != nil {
		return nil, internalError()
	}

	return connect.NewResponse(&teamsv1.CreateTeamResponse{
		Team: teamMessage(team),
	}), nil
}

// GetTeam returns a team to any member.
func (s *TeamService) GetTeam(
	ctx context.Context,
	req *connect.Request[teamsv1.GetTeamRequest],
) (*connect.Response[teamsv1.GetTeamResponse], error) {
	_, team, _, err := s.authorizeTeam(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&teamsv1.GetTeamResponse{
		Team: teamMessage(team),
	}), nil
}

// UpdateTeam renames a team. Renaming never changes the slug and is
// owner-only.
func (s *TeamService) UpdateTeam(
	ctx context.Context,
	req *connect.Request[teamsv1.UpdateTeamRequest],
) (*connect.Response[teamsv1.UpdateTeamResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	if raw, present := stringPointer(req.Msg.Name); present {
		name, validationErr := requiredText(raw, "name")
		if validationErr != nil {
			return nil, validationErr
		}
		if err := s.queries.UpdateTeamName(ctx, db.UpdateTeamNameParams{
			Name: name,
			ID:   team.ID,
		}); err != nil {
			return nil, internalError()
		}
		team.Name = name
	}
	return connect.NewResponse(&teamsv1.UpdateTeamResponse{
		Team: teamMessage(team),
	}), nil
}

// DeleteTeam hard-deletes a team and every row that cascades from it. It is
// owner-only.
func (s *TeamService) DeleteTeam(
	ctx context.Context,
	req *connect.Request[teamsv1.DeleteTeamRequest],
) (*connect.Response[teamsv1.DeleteTeamResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	if err := s.queries.DeleteTeam(ctx, team.ID); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&teamsv1.DeleteTeamResponse{}), nil
}

// ListMyTeams returns every team the caller belongs to, with their role.
func (s *TeamService) ListMyTeams(
	ctx context.Context,
	_ *connect.Request[teamsv1.ListMyTeamsRequest],
) (*connect.Response[teamsv1.ListMyTeamsResponse], error) {
	user, err := currentUser(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListTeamsForUser(ctx, user.ID)
	if err != nil {
		return nil, internalError()
	}
	memberships := make([]*teamsv1.TeamMembership, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		memberships = append(memberships, &teamsv1.TeamMembership{
			Team: &teamsv1.Team{Id: row.ID, Slug: row.Slug, Name: row.Name},
			Role: row.Role,
		})
	}
	return connect.NewResponse(&teamsv1.ListMyTeamsResponse{
		Memberships: memberships,
	}), nil
}

// AddMembership adds an existing user, addressed by email, to a team as a
// member. It is owner-only.
func (s *TeamService) AddMembership(
	ctx context.Context,
	req *connect.Request[teamsv1.AddMembershipRequest],
) (*connect.Response[teamsv1.AddMembershipResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	email := strings.ToLower(strings.TrimSpace(req.Msg.GetEmail()))
	if email == "" {
		return nil, invalidArgument(fieldViolation("email", "must not be empty"))
	}
	user, err := s.queries.GetUserByEmail(ctx, email)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, userNotFound()
	case err != nil:
		return nil, internalError()
	}

	_, err = s.queries.GetMembership(ctx, db.GetMembershipParams{
		TeamID: team.ID,
		UserID: user.ID,
	})
	switch {
	case err == nil:
		return nil, alreadyExistsError("user is already a member")
	case !errors.Is(err, sql.ErrNoRows):
		return nil, internalError()
	}

	membershipID, err := newID()
	if err != nil {
		return nil, internalError()
	}
	if _, err := s.queries.CreateMembership(ctx, db.CreateMembershipParams{
		ID:        membershipID,
		UserID:    user.ID,
		TeamID:    team.ID,
		Role:      roleMember,
		CreatedAt: s.timestamp(),
	}); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&teamsv1.AddMembershipResponse{
		Membership: membershipForUser(user, roleMember),
	}), nil
}

// UpdateMembership changes a team member's role. It is owner-only and refuses
// to demote the team's last owner.
func (s *TeamService) UpdateMembership(
	ctx context.Context,
	req *connect.Request[teamsv1.UpdateMembershipRequest],
) (*connect.Response[teamsv1.UpdateMembershipResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	role := strings.TrimSpace(req.Msg.GetRole())
	if role != roleOwner && role != roleMember {
		return nil, invalidArgument(fieldViolation("role", `must be "owner" or "member"`))
	}
	userID := req.Msg.GetUserId()
	membership, err := s.queries.GetMembership(ctx, db.GetMembershipParams{
		TeamID: team.ID,
		UserID: userID,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("membership")
	case err != nil:
		return nil, internalError()
	}
	if membership.Role == roleOwner && role == roleMember {
		owners, countErr := s.queries.CountOwners(ctx, team.ID)
		if countErr != nil {
			return nil, internalError()
		}
		if owners <= 1 {
			return nil, lastOwnerError()
		}
	}
	if updateErr := s.queries.UpdateMembershipRole(ctx, db.UpdateMembershipRoleParams{
		Role:   role,
		TeamID: team.ID,
		UserID: userID,
	}); updateErr != nil {
		return nil, internalError()
	}
	row, err := s.queries.GetMembershipWithUser(ctx, db.GetMembershipWithUserParams{
		TeamID: team.ID,
		UserID: userID,
	})
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&teamsv1.UpdateMembershipResponse{
		Membership: membershipMessage(&row),
	}), nil
}

// RemoveMembership removes a member from a team. It is owner-only and refuses
// to remove the team's last owner.
func (s *TeamService) RemoveMembership(
	ctx context.Context,
	req *connect.Request[teamsv1.RemoveMembershipRequest],
) (*connect.Response[teamsv1.RemoveMembershipResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	userID := req.Msg.GetUserId()
	membership, err := s.queries.GetMembership(ctx, db.GetMembershipParams{
		TeamID: team.ID,
		UserID: userID,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, notFound("membership")
	case err != nil:
		return nil, internalError()
	}
	if membership.Role == roleOwner {
		owners, countErr := s.queries.CountOwners(ctx, team.ID)
		if countErr != nil {
			return nil, internalError()
		}
		if owners <= 1 {
			return nil, lastOwnerError()
		}
	}
	if deleteErr := s.queries.DeleteMembership(ctx, db.DeleteMembershipParams{
		TeamID: team.ID,
		UserID: userID,
	}); deleteErr != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&teamsv1.RemoveMembershipResponse{}), nil
}

// GetTeamRecord is owned by ticket 05 (gameplay reads).
func (s *TeamService) GetTeamRecord(
	_ context.Context,
	_ *connect.Request[teamsv1.GetTeamRecordRequest],
) (*connect.Response[teamsv1.GetTeamRecordResponse], error) {
	return nil, connect.NewError(
		connect.CodeUnimplemented,
		errors.New("GetTeamRecord is implemented by ticket 05"),
	)
}
