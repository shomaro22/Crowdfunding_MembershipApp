package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

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

func mapEntityErr(err error) error {
	switch {
	case errors.Is(err, entity.ErrBadTitle),
		errors.Is(err, entity.ErrBadProject),
		errors.Is(err, entity.ErrBadDuration),
		errors.Is(err, entity.ErrBadMembership),
		errors.Is(err, entity.ErrBadPrice),
		errors.Is(err, entity.ErrBadImages):
		return ErrBadRequest
	default:
		return err
	}
}

func buildProjectImages(ins []ImageIn, legacyURL *string, now time.Time) ([]entity.ProjectImage, *string, error) {
	var (
		inputs  []entity.ImageInput
		images  []entity.ProjectImage
		mainURL *string
		err     error
		i       int
		url     string
		orders  []int
	)

	if len(ins) > 0 {
		inputs = make([]entity.ImageInput, len(ins))
		for i = range ins {
			inputs[i] = entity.ImageInput{
				ImageURL: ins[i].ImageURL,
				IsMain:   ins[i].IsMain,
				Order:    ins[i].Order,
			}
		}
	} else if legacyURL != nil && strings.TrimSpace(*legacyURL) != "" {
		url = strings.TrimSpace(*legacyURL)
		inputs = []entity.ImageInput{{ImageURL: url, IsMain: true, Order: 0}}
	}
	if err = entity.ValidateProjectImages(inputs); err != nil {
		return nil, nil, err
	}
	if len(inputs) == 0 {
		return nil, nil, nil
	}
	orders = imageSortOrders(inputs)
	images = make([]entity.ProjectImage, len(inputs))
	for i = range inputs {
		url = strings.TrimSpace(inputs[i].ImageURL)
		images[i] = entity.ProjectImage{
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

func imageSortOrders(inputs []entity.ImageInput) []int {
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

func (p *ProjectUseCase) enrichList(ctx context.Context, rows []entity.Project, total int64) ([]ProjectView, int64, error) {
	var (
		out         []ProjectView
		sellerCache map[uuid.UUID]*UserView
		i           int
		project        entity.Project
		seller      *UserView
		ok          bool
		u           *entity.User
		plans       []entity.ProjectPlan
		images      []entity.ProjectImage
		err         error
		projectCopy entity.Project
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

	u, err = p.Users.FindID(ctx, sellerID)
	if err != nil || u == nil || u.Role != entity.UserRoleSeller {
		return nil, 0, ErrForbidden
	}
	rows, total, err = p.Projects.ListSeller(ctx, sellerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return p.enrichList(ctx, rows, total)
}

func (p *ProjectUseCase) FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole string) (*ProjectView, error) {
	var (
		project   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		u      *entity.User
		seller *UserView
		err    error
		now    = time.Now().UTC()
	)

	project, plans, images, err = p.Projects.FindID(ctx, id)
	if err != nil || project == nil {
		return nil, ErrNotFound
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
		project  *entity.Project
		plans []entity.ProjectPlan
		err   error
		now   time.Time
	)

	project, plans, _, err = p.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return nil, ErrNotFound
	}
	now = time.Now().UTC()
	if !canViewProject(project, viewerID, parseUserRole(viewerRole), now) {
		return nil, ErrNotFound
	}
	return ProjectPlansFrom(plans), nil
}

func parseUserRole(raw string) entity.UserRole {
	var role entity.UserRole
	role = entity.UserRole(strings.TrimSpace(raw))
	if role.Valid() {
		return role
	}
	return ""
}

func canViewProject(project *entity.Project, viewerID *uuid.UUID, role entity.UserRole, now time.Time) bool {
	if project.IsPublic(now) {
		return true
	}
	if viewerID != nil && *viewerID == project.SellerID {
		return true
	}
	if role == entity.UserRoleAdmin {
		return true
	}
	return false
}
