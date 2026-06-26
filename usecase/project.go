package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlanIn struct {
	MembershipType entity.MembershipType
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
	DurationType entity.DurationType
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
	DurationType *entity.DurationType
	StartDate    *time.Time
	EndDate      *time.Time
	ImageURL     *string
	Images       *[]ImageIn
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
	DB       *gorm.DB
}

func mapEntityErr(err error) error {
	switch err {
	case entity.ErrBadTitle, entity.ErrBadProject, entity.ErrBadDuration, entity.ErrBadMembership, entity.ErrBadPrice, entity.ErrBadImages:
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
			u, err = p.Users.FindID(ctx, proj.SellerID)
			if err != nil {
				return nil, 0, err
			}
			seller = u
			sellerCache[proj.SellerID] = u
		}
		_, plans, images, err = p.Projects.FindID(ctx, proj.ID)
		if err != nil {
			return nil, 0, err
		}
		pcopy = proj
		out = append(out, ProjectView{Project: &pcopy, Plans: plans, Images: images, Seller: seller})
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

func (p *ProjectUseCase) FindView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) (*ProjectView, error) {
	var (
		proj   *entity.Project
		plans  []entity.ProjectPlan
		images []entity.ProjectImage
		seller *entity.User
		err    error
		now    = time.Now().UTC()
	)

	proj, plans, images, err = p.Projects.FindID(ctx, id)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	if !canViewProject(proj, viewerID, viewerRole, now) {
		return nil, ErrNotFound
	}
	seller, err = p.Users.FindID(ctx, proj.SellerID)
	if err != nil {
		return nil, err
	}
	return &ProjectView{Project: proj, Plans: plans, Images: images, Seller: seller}, nil
}

func (p *ProjectUseCase) ListPlans(ctx context.Context, projectID uuid.UUID, viewerID *uuid.UUID, viewerRole entity.UserRole) ([]entity.ProjectPlan, error) {
	var (
		proj  *entity.Project
		plans []entity.ProjectPlan
		err   error
		now   time.Time
	)

	proj, plans, _, err = p.Projects.FindID(ctx, projectID)
	if err != nil || proj == nil {
		return nil, ErrNotFound
	}
	now = time.Now().UTC()
	if !canViewProject(proj, viewerID, viewerRole, now) {
		return nil, ErrNotFound
	}
	return plans, nil
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
