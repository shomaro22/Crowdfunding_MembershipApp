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
	Projects repository.ProjectRepo
	Users    repository.UserRepo
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
		return nil, nil, mapProjectEntityErr(err)
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
		project        model.Project
		seller      *UserView
		ok          bool
		u           *model.User
		plans       []model.ProjectPlan
		images      []model.ProjectImage
		err         error
		projectCopy model.Project
	)

	out = make([]ProjectView, 0, len(rows))
	sellerCache = map[uuid.UUID]*UserView{}
	for i = range rows {
		project = rows[i]
		seller, ok = sellerCache[project.SellerID]
		if !ok {
			u, err = p.Users.FindID(ctx, project.SellerID)
			if err != nil {
				return nil, 0, err
			}
			seller = UserViewPtr(u)
			sellerCache[project.SellerID] = seller
		}
		_, plans, images, err = p.Projects.FindID(ctx, project.ID)
		if err != nil {
			return nil, 0, err
		}
		projectCopy = project
		out = append(out, AssembleProjectView(&projectCopy, plans, images, seller))
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
		u     *model.User
		err   error
		rows  []model.Project
		total int64
	)

	u, err = p.Users.FindID(ctx, sellerID)
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
		project   *model.Project
		plans  []model.ProjectPlan
		images []model.ProjectImage
		u      *model.User
		seller *UserView
		err    error
		now    = time.Now().UTC()
	)

	project, plans, images, err = p.Projects.FindID(ctx, id)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if !canViewProject(project, viewerID, parseUserRole(viewerRole), now) {
		return nil, ErrNotFound
	}
	u, err = p.Users.FindID(ctx, project.SellerID)
	if err != nil {
		return nil, err
	}
	seller = UserViewPtr(u)
	v := AssembleProjectView(project, plans, images, seller)
	return &v, nil
}

func (p *ProjectUseCase) ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole string) ([]ProjectPlanView, error) {
	var (
		project  *model.Project
		plans []model.ProjectPlan
		err   error
		now   time.Time
	)

	project, plans, _, err = p.Projects.FindID(ctx, projectID)
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

func canViewProject(project *model.Project, viewerID *uuid.UUID, role model.UserRole, now time.Time) bool {
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
