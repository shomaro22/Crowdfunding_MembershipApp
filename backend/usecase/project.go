package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlanIn struct {
	MembershipType string
	Price          int
	MaxSupporters  *int
	Benefits       string
}

type ImageIn struct {
	ImageURL string
	IsMain   bool
	Order    int
}

type CreateProjectIn struct {
	Title        string
	Description  string
	TargetAmount int64
	DurationType string
	StartDate    time.Time
	EndDate      *time.Time
	ImageURL     *string
	Images       []ImageIn
	Plans        []PlanIn
}

type UpdateProjectIn struct {
	Title        *string
	Description  *string
	TargetAmount *int64
	DurationType *string
	StartDate    *time.Time
	EndDate      *time.Time
	ImageURL     *string
	Images       *[]ImageIn
	IsReapply    bool
}

type ProjectUseCase struct {
	Projects repository.IProjectRepo
	Users    repository.IUserRepo
	DB       *gorm.DB
}

func buildProjectImages(ins []ImageIn, legacyURL *string, now time.Time) ([]model.ProjectImage, *string, error) {
	var (
		inputs  []model.ImageInput
		images  []model.ProjectImage
		mainURL *string
		err     error
		i       int
		url     string
		orders  []int
	)

	if len(ins) > 0 {
		inputs = make([]model.ImageInput, len(ins))
		for i = range ins {
			inputs[i] = model.ImageInput{
				ImageURL: ins[i].ImageURL,
				IsMain:   ins[i].IsMain,
				Order:    ins[i].Order,
			}
		}
	} else if legacyURL != nil && strings.TrimSpace(*legacyURL) != "" {
		url = strings.TrimSpace(*legacyURL)
		inputs = []model.ImageInput{{ImageURL: url, IsMain: true, Order: 0}}
	}
	if err = validator.ValidateProjectImages(inputs); err != nil {
		return nil, nil, mapProjectModelErr(err)
	}
	if len(inputs) == 0 {
		return nil, nil, nil
	}
	orders = imageSortOrders(inputs)
	images = make([]model.ProjectImage, len(inputs))
	for i = range inputs {
		url = strings.TrimSpace(inputs[i].ImageURL)
		images[i] = model.ProjectImage{
			ID:        uuid.New(),
			ImageURL:  url,
			IsMain:    inputs[i].IsMain,
			SortOrder: orders[i],
			CreatedAt: now,
		}
		if inputs[i].IsMain {
			mainURL = &url
		}
	}
	return images, mainURL, nil
}

func imageSortOrders(inputs []model.ImageInput) []int {
	var (
		orders  = make([]int, len(inputs))
		allZero = true
		i       int
	)
	for i = range inputs {
		orders[i] = inputs[i].Order
		if inputs[i].Order != 0 {
			allZero = false
		}
	}
	if allZero {
		for i = range orders {
			orders[i] = i
		}
	}
	return orders
}

func (p *ProjectUseCase) EnrichList(ctx context.Context, rows []model.Project, total int64) ([]ProjectView, int64, error) {
	var (
		out         []ProjectView
		sellerCache map[uuid.UUID]*UserView
		i           int
		project     model.Project
		seller      *UserView
		ok          bool
		u           model.User
		plans       []model.ProjectPlan
		images      []model.ProjectImage
		err         error
	)

	out = make([]ProjectView, 0, len(rows))
	sellerCache = map[uuid.UUID]*UserView{}
	for i = range rows {
		project = rows[i]
		seller, ok = sellerCache[project.SellerID]
		if !ok {
			u, err = p.Users.GetID(ctx, project.SellerID)
			if err != nil {
				return nil, 0, mapRepoNotFound(err)
			}
			seller = UserViewPtr(u)
			sellerCache[project.SellerID] = seller
		}
		_, plans, images, err = p.Projects.GetID(ctx, project.ID)
		if err != nil {
			return nil, 0, mapRepoNotFound(err)
		}
		out = append(out, AssembleProjectView(project, plans, images, seller))
	}
	return out, total, nil
}

func (p *ProjectUseCase) ListPublic(ctx context.Context, search, sort string, limit, offset int) ([]ProjectView, int64, error) {
	var (
		rows  []model.Project
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
	return p.EnrichList(ctx, rows, total)
}

func (p *ProjectUseCase) ListMine(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]ProjectView, int64, error) {
	var (
		u     model.User
		err   error
		rows  []model.Project
		total int64
	)

	u, err = p.Users.GetID(ctx, sellerID)
	if err != nil {
		return nil, 0, mapRepoNotFoundAs(err, ErrForbidden)
	}
	if u.Role != model.UserRoleSeller {
		return nil, 0, ErrForbidden
	}
	rows, total, err = p.Projects.ListSeller(ctx, sellerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return p.EnrichList(ctx, rows, total)
}

func (p *ProjectUseCase) FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole string) (*ProjectView, error) {
	var (
		project model.Project
		plans   []model.ProjectPlan
		images  []model.ProjectImage
		u       model.User
		seller  *UserView
		err     error
		now     = time.Now().UTC()
		v       ProjectView
	)

	project, plans, images, err = p.Projects.GetID(ctx, id)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if !canViewProject(project, viewerID, parseUserRole(viewerRole), now) {
		return nil, ErrNotFound
	}
	u, err = p.Users.GetID(ctx, project.SellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	seller = UserViewPtr(u)
	v = AssembleProjectView(project, plans, images, seller)
	return &v, nil
}

func (p *ProjectUseCase) ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole string) ([]ProjectPlanView, error) {
	var (
		project model.Project
		plans   []model.ProjectPlan
		err     error
		now     time.Time
	)

	project, plans, _, err = p.Projects.GetID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	now = time.Now().UTC()
	if !canViewProject(project, viewerID, parseUserRole(viewerRole), now) {
		return nil, ErrNotFound
	}
	return ProjectPlansFrom(plans), nil
}

func parseUserRole(raw string) model.UserRole {
	var role model.UserRole
	role = model.UserRole(strings.TrimSpace(raw))
	if role.Valid() {
		return role
	}
	return ""
}

func canViewProject(project model.Project, viewerID *uuid.UUID, role model.UserRole, now time.Time) bool {
	if project.IsPublic(now) {
		return true
	}
	if viewerID != nil && *viewerID == project.SellerID {
		return true
	}
	if role == model.UserRoleAdmin {
		return true
	}
	return false
}

func (p *ProjectUseCase) Create(ctx context.Context, sellerID uuid.UUID, in CreateProjectIn) (*ProjectView, error) {
	var (
		u          model.User
		err        error
		planInputs []model.PlanInput
		now        time.Time
		project    *model.Project
		plans      []model.ProjectPlan
		images     []model.ProjectImage
		i          int
		plan       PlanIn
	)

	u, err = p.Users.GetID(ctx, sellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleSeller || u.SellerStatus != model.SellerStatusActive {
		return nil, ErrForbidden
	}
	planInputs = make([]model.PlanInput, len(in.Plans))
	for i, plan = range in.Plans {
		planInputs[i] = model.PlanInput{
			MembershipType: model.MembershipType(plan.MembershipType),
			Price:          plan.Price,
			MaxSupporters:  plan.MaxSupporters,
			Benefits:       plan.Benefits,
		}
	}
	if err = validator.ValidateProject(in.Title, in.Description, in.TargetAmount, model.DurationType(in.DurationType), in.StartDate, in.EndDate, planInputs); err != nil {
		return nil, mapProjectModelErr(err)
	}
	now = time.Now().UTC()
	project = &model.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        strings.TrimSpace(in.Title),
		Description:  strings.TrimSpace(in.Description),
		TargetAmount: in.TargetAmount,
		DurationType: model.DurationType(in.DurationType),
		StartDate:    in.StartDate.UTC(),
		EndDate:      in.EndDate,
		Status:       model.ProjectStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	plans = make([]model.ProjectPlan, 0, len(in.Plans))
	for _, plan = range in.Plans {
		plans = append(plans, model.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: model.MembershipType(plan.MembershipType),
			Price:          plan.Price,
			MaxSupporters:  plan.MaxSupporters,
			Benefits:       strings.TrimSpace(plan.Benefits),
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	images, project.MainImageURL, err = buildProjectImages(in.Images, in.ImageURL, now)
	if err != nil {
		return nil, mapProjectModelErr(err)
	}
	if err = p.Projects.Create(ctx, project, plans, images); err != nil {
		return nil, err
	}
	return AssembleProjectViewPtr(*project, plans, images, UserViewPtr(u)), nil
}

func (p *ProjectUseCase) Update(ctx context.Context, sellerID, projectID uuid.UUID, in UpdateProjectIn) (*ProjectView, error) {
	var (
		project model.Project
		plans   []model.ProjectPlan
		images  []model.ProjectImage
		seller  model.User
		err     error
		now     time.Time
		dur     model.DurationType
		start   time.Time
		end     *time.Time
	)

	project, plans, images, err = p.Projects.GetID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if project.SellerID != sellerID {
		return nil, ErrForbidden
	}
	now = time.Now().UTC()
	if in.Title != nil {
		if err = validator.ValidateTitle(*in.Title); err != nil {
			return nil, mapProjectModelErr(err)
		}
		project.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		if strings.TrimSpace(*in.Description) == "" {
			return nil, ErrBadRequest
		}
		project.Description = strings.TrimSpace(*in.Description)
	}
	if in.TargetAmount != nil {
		if *in.TargetAmount < 0 {
			return nil, ErrBadRequest
		}
		project.TargetAmount = *in.TargetAmount
	}
	dur = project.DurationType
	start = project.StartDate
	end = project.EndDate
	if in.DurationType != nil {
		dur = model.DurationType(*in.DurationType)
	}
	if in.StartDate != nil {
		start = in.StartDate.UTC()
	}
	if in.EndDate != nil {
		end = in.EndDate
	}
	if in.DurationType != nil || in.StartDate != nil || in.EndDate != nil {
		if err = validator.ValidateDuration(dur, start, end); err != nil {
			return nil, mapProjectModelErr(err)
		}
		project.DurationType = dur
		project.StartDate = start
		project.EndDate = end
	}
	if in.Images != nil {
		images, project.MainImageURL, err = buildProjectImages(*in.Images, nil, now)
		if err != nil {
			return nil, mapProjectModelErr(err)
		}
	} else if in.ImageURL != nil {
		images, project.MainImageURL, err = buildProjectImages(nil, in.ImageURL, now)
		if err != nil {
			return nil, mapProjectModelErr(err)
		}
	}
	if in.IsReapply && project.Status == model.ProjectStatusActive {
		project.Status = model.ProjectStatusPending
	}
	project.UpdatedAt = now
	err = repository.WithTx(ctx, p.DB, func(txCtx context.Context) error {
		if err = p.Projects.Update(txCtx, &project); err != nil {
			return err
		}
		if in.Images != nil || in.ImageURL != nil {
			return p.Projects.ReplaceImages(txCtx, projectID, images)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	seller, err = p.Users.GetID(ctx, project.SellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}

func (p *ProjectUseCase) RequestDelete(ctx context.Context, sellerID, projectID uuid.UUID) (*ProjectView, error) {
	var (
		project model.Project
		plans   []model.ProjectPlan
		images  []model.ProjectImage
		seller  model.User
		err     error
		now     time.Time
	)

	project, plans, images, err = p.Projects.GetID(ctx, projectID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if project.SellerID != sellerID {
		return nil, ErrForbidden
	}
	if project.Status != model.ProjectStatusActive {
		return nil, ErrInvalidState
	}
	now = time.Now().UTC()
	project.DeleteRequested = true
	project.Status = model.ProjectStatusDeleteRequested
	project.UpdatedAt = now
	if err = p.Projects.Update(ctx, &project); err != nil {
		return nil, err
	}
	seller, err = p.Users.GetID(ctx, project.SellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	return AssembleProjectViewPtr(project, plans, images, UserViewPtr(seller)), nil
}
