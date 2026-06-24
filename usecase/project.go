package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

type PlanIn struct {
	MembershipType entity.MembershipType
	Price          int
	MaxSupporters  *int
	Benefits       string
}

type CreateProjectIn struct {
	Title        string
	Description  string
	TargetAmount int64
	DurationType entity.DurationType
	StartDate    time.Time
	EndDate      *time.Time
	ImageURL     *string
	Plans        []PlanIn
}

type UpdateProjectIn struct {
	Title        *string
	Description  *string
	TargetAmount *int64
	DurationType *entity.DurationType
	StartDate    *time.Time
	EndDate      *time.Time
	ImageURL     *string
	IsReapply    bool
}

type ProjectView struct {
	Project *entity.Project
	Plans   []entity.ProjectPlan
	Images  []entity.ProjectImage
	Seller  *entity.User
}

type ProjectUseCase struct {
	Projects repository.ProjectRepo
	Users    repository.UserRepo
}

func (p *ProjectUseCase) Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error) {
	var (
		u          *entity.User
		err        error
		planInputs []entity.PlanInput
		now        time.Time
		proj       *entity.Project
		plans      []entity.ProjectPlan
		images     []entity.ProjectImage
		i          int
		pl         PlanIn
		url        string
	)

	u, err = p.Users.FindByID(ctx, sellerID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if u.Role != entity.UserRoleSeller || u.SellerStatus != entity.SellerStatusActive {
		return nil, ErrForbidden
	}
	planInputs = make([]entity.PlanInput, len(in.Plans))
	for i, pl = range in.Plans {
		planInputs[i] = entity.PlanInput{
			MembershipType: pl.MembershipType,
			Price:          pl.Price,
			MaxSupporters:  pl.MaxSupporters,
			Benefits:       pl.Benefits,
		}
	}
	if err = entity.ValidateProject(in.Title, in.Description, in.TargetAmount, in.DurationType, in.StartDate, in.EndDate, planInputs); err != nil {
		return nil, mapEntityErr(err)
	}
	now = time.Now().UTC()
	proj = &entity.Project{
		ID:             uuid.New(),
		SellerID:       sellerID,
		Title:          strings.TrimSpace(in.Title),
		Description:    strings.TrimSpace(in.Description),
		TargetAmount:   in.TargetAmount,
		DurationType:   in.DurationType,
		StartDate:      in.StartDate.UTC(),
		EndDate:        in.EndDate,
		Status:         entity.ProjectStatusPending,
		MainImageURL:   in.ImageURL,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	plans = make([]entity.ProjectPlan, 0, len(in.Plans))
	for _, pl = range in.Plans {
		plans = append(plans, entity.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: pl.MembershipType,
			Price:          pl.Price,
			MaxSupporters:  pl.MaxSupporters,
			Benefits:       strings.TrimSpace(pl.Benefits),
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	if in.ImageURL != nil && strings.TrimSpace(*in.ImageURL) != "" {
		url = strings.TrimSpace(*in.ImageURL)
		proj.MainImageURL = &url
		images = append(images, entity.ProjectImage{
			ID:        uuid.New(),
			ImageURL:  url,
			IsMain:    true,
			SortOrder: 0,
			CreatedAt: now,
		})
	}
	if err = p.Projects.Create(ctx, proj, plans, images); err != nil {
		return nil, err
	}
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: u}, nil
}

func (p *ProjectUseCase) ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]ProjectView, int64, error) {
	var (
		rows  []entity.Project
		total int64
		err   error
	)

	rows, total, err = p.Projects.ListPublic(ctx, repository.ProjectListFilter{
		Search: search,
		Sort:   sort,
		Limit:  limit,
		Offset: offset,
		AsOf:   time.Now().UTC(),
	})
	if err != nil {
		return nil, 0, err
	}
	return p.enrichList(ctx, rows, total)
}

func (p *ProjectUseCase) ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]ProjectView, int64, error) {
	var (
		u     *entity.User
		err   error
		rows  []entity.Project
		total int64
	)

	u, err = p.Users.FindByID(ctx, sellerID)
	if err != nil || u == nil || u.Role != entity.UserRoleSeller {
		return nil, 0, ErrForbidden
	}
	rows, total, err = p.Projects.ListBySeller(ctx, sellerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return p.enrichList(ctx, rows, total)
}

func (p *ProjectUseCase) FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    = time.Now().UTC()
	)

	proj, plans, images, err = p.Projects.FindByID(ctx, id)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if !canViewProject(proj, viewerID, viewerRole, now) {
		return nil, ErrNotFound
	}
	seller, err = p.Users.FindByID(ctx, proj.SellerID)
	if err != nil {
		return nil, err
	}
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (p *ProjectUseCase) ListPlans(ctx context.Context, projectID uuid.UUID) ([]entity.ProjectPlan, error) {
	var (
		proj  *entity.Project
		plans []entity.ProjectPlan
		err   error
		now   time.Time
	)

	proj, plans, _, err = p.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	now = time.Now().UTC()
	if !proj.IsPublic(now) {
		return nil, ErrNotFound
	}
	return plans, nil
}

func (p *ProjectUseCase) Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    time.Time
		dur    entity.DurationType
		start  time.Time
		end    *time.Time
		url    string
	)

	proj, plans, images, err = p.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if proj.SellerID != sellerID {
		return nil, ErrForbidden
	}
	now = time.Now().UTC()
	if in.Title != nil {
		if err = entity.ValidateTitle(*in.Title); err != nil {
			return nil, mapEntityErr(err)
		}
		proj.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		if strings.TrimSpace(*in.Description) == "" {
			return nil, ErrBadRequest
		}
		proj.Description = strings.TrimSpace(*in.Description)
	}
	if in.TargetAmount != nil {
		if *in.TargetAmount < 0 {
			return nil, ErrBadRequest
		}
		proj.TargetAmount = *in.TargetAmount
	}
	dur = proj.DurationType
	start = proj.StartDate
	end = proj.EndDate
	if in.DurationType != nil {
		dur = *in.DurationType
	}
	if in.StartDate != nil {
		start = in.StartDate.UTC()
	}
	if in.EndDate != nil {
		end = in.EndDate
	}
	if in.DurationType != nil || in.StartDate != nil || in.EndDate != nil {
		if err = entity.ValidateDuration(dur, start, end); err != nil {
			return nil, mapEntityErr(err)
		}
		proj.DurationType = dur
		proj.StartDate = start
		proj.EndDate = end
	}
	if in.ImageURL != nil {
		url = strings.TrimSpace(*in.ImageURL)
		if url == "" {
			proj.MainImageURL = nil
		} else {
			proj.MainImageURL = &url
		}
	}
	if in.IsReapply && proj.Status == entity.ProjectStatusActive {
		proj.Status = entity.ProjectStatusPending
	}
	proj.UpdatedAt = now
	if err = p.Projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	seller, _ = p.Users.FindByID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (p *ProjectUseCase) RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    time.Time
	)

	proj, plans, images, err = p.Projects.FindByID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if proj.SellerID != sellerID {
		return nil, ErrForbidden
	}
	if proj.Status != entity.ProjectStatusActive {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	proj.DeleteRequested = true
	proj.Status = entity.ProjectStatusDeleteRequested
	proj.UpdatedAt = now
	if err = p.Projects.Update(ctx, proj); err != nil {
		return nil, err
	}
	seller, _ = p.Users.FindByID(ctx, proj.SellerID)
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (p *ProjectUseCase) enrichList(ctx context.Context, rows []entity.Project, total int64) ([]ProjectView, int64, error) {
	var (
		out         []ProjectView
		sellerCache map[uuid.UUID]*entity.User
		i           int
		proj        entity.Project
		seller      *entity.User
		ok          bool
		u           *entity.User
		plans       []entity.ProjectPlan
		images      []entity.ProjectImage
		err         error
		pcopy       entity.Project
	)

	out = make([]ProjectView, 0, len(rows))
	sellerCache = map[uuid.UUID]*entity.User{}
	for i = range rows {
		proj = rows[i]
		seller, ok = sellerCache[proj.SellerID]
		if !ok {
			u, err = p.Users.FindByID(ctx, proj.SellerID)
			if err != nil {
				return nil, 0, err
			}
			seller = u
			sellerCache[proj.SellerID] = u
		}
		_, plans, images, err = p.Projects.FindByID(ctx, proj.ID)
		if err != nil {
			return nil, 0, err
		}
		pcopy = proj
		out = append(out, ProjectView{Project: &pcopy, Plans: plans, Images: images, Seller: seller})
	}
	return out, total, nil
}

func canViewProject(p *entity.Project, viewerID *uuid.UUID, role entity.UserRole, now time.Time) bool {
	if p.IsPublic(now) {
		return true
	}
	if viewerID != nil && *viewerID == p.SellerID {
		return true
	}
	if role == entity.UserRoleAdmin {
		return true
	}
	return false
}

func mapEntityErr(err error) error {
	switch err {
	case entity.ErrBadTitle, entity.ErrBadProject, entity.ErrBadDuration, entity.ErrBadMembership, entity.ErrBadPrice:
		return ErrBadRequest
	default:
		return err
	}
}
