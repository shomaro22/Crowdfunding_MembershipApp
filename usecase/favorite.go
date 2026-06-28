package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type FavoriteStatusView struct {
	ProjectID   uuid.UUID
	IsFavorited bool
}

type ToggleFavoriteIn struct {
	ProjectID uuid.UUID
	Action    string
	Meta      RequestMeta
}

type FavoriteUseCase struct {
	Favorites repository.FavoriteRepo
	Projects  *ProjectUseCase
	Activity  repository.ActivityRepo
}

func (f *FavoriteUseCase) List(ctx context.Context, userID uuid.UUID) ([]ProjectView, error) {
	var (
		rows     []entity.Favorite
		projects []entity.Project
		i        int
		fav      entity.Favorite
		project  *entity.Project
		err      error
	)
	if f == nil || f.Favorites == nil || f.Projects == nil {
		return nil, ErrBadRequest
	}
	rows, err = f.Favorites.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	projects = make([]entity.Project, 0, len(rows))
	for i = range rows {
		fav = rows[i]
		project, _, _, err = f.Projects.Projects.FindID(ctx, fav.ProjectID)
		if err != nil {
			return nil, err
		}
		if project == nil {
			continue
		}
		projects = append(projects, *project)
	}
	if len(projects) == 0 {
		return []ProjectView{}, nil
	}
	out, _, err := f.Projects.EnrichList(ctx, projects, int64(len(projects)))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (f *FavoriteUseCase) Toggle(ctx context.Context, userID uuid.UUID, in ToggleFavoriteIn) (FavoriteStatusView, error) {
	var (
		projectID uuid.UUID
		action    string
		err       error
	)
	if f == nil || f.Favorites == nil || f.Projects == nil {
		return FavoriteStatusView{}, ErrBadRequest
	}
	projectID, action, err = validator.ValidateFavoriteToggle(validator.FavoriteToggleInput{
		ProjectID: in.ProjectID,
		Action:    in.Action,
	})
	if err != nil {
		return FavoriteStatusView{}, ErrBadRequest
	}
	switch action {
	case validator.FavoriteActionAdd:
		return f.addFavorite(ctx, userID, projectID, in.Meta)
	case validator.FavoriteActionRemove:
		return f.removeFavorite(ctx, userID, projectID, in.Meta)
	default:
		return FavoriteStatusView{}, ErrBadRequest
	}
}

func (f *FavoriteUseCase) Check(ctx context.Context, userID, projectID uuid.UUID) (FavoriteStatusView, error) {
	var (
		exists bool
		err    error
	)
	if f == nil || f.Favorites == nil {
		return FavoriteStatusView{}, ErrBadRequest
	}
	if projectID == uuid.Nil {
		return FavoriteStatusView{}, ErrBadRequest
	}
	exists, err = f.Favorites.Exists(ctx, userID, projectID)
	if err != nil {
		return FavoriteStatusView{}, err
	}
	return FavoriteStatusView{ProjectID: projectID, IsFavorited: exists}, nil
}

func (f *FavoriteUseCase) addFavorite(ctx context.Context, userID, projectID uuid.UUID, meta RequestMeta) (FavoriteStatusView, error) {
	var (
		exists bool
		now    time.Time
		err    error
	)
	_, err = f.Projects.FindView(ctx, projectID, &userID, string(entity.UserRoleBuyer))
	if err != nil {
		return FavoriteStatusView{}, err
	}
	exists, err = f.Favorites.Exists(ctx, userID, projectID)
	if err != nil {
		return FavoriteStatusView{}, err
	}
	if exists {
		return FavoriteStatusView{ProjectID: projectID, IsFavorited: true}, ErrFavoriteExists
	}
	now = time.Now().UTC()
	err = f.Favorites.Create(ctx, &entity.Favorite{
		ID:        uuid.New(),
		UserID:    userID,
		ProjectID: projectID,
		CreatedAt: now,
	})
	if err != nil {
		if repository.IsFavoriteDuplicate(err) {
			return FavoriteStatusView{ProjectID: projectID, IsFavorited: true}, ErrFavoriteExists
		}
		return FavoriteStatusView{}, err
	}
	LogActivity(ctx, f.Activity, &userID, entity.FavoriteAdd, "project", projectID.String(), meta, ActivityMeta(map[string]string{
		"project_id": projectID.String(),
	}))
	return FavoriteStatusView{ProjectID: projectID, IsFavorited: true}, nil
}

func (f *FavoriteUseCase) removeFavorite(ctx context.Context, userID, projectID uuid.UUID, meta RequestMeta) (FavoriteStatusView, error) {
	var (
		removed bool
		err     error
	)
	removed, err = f.Favorites.Delete(ctx, userID, projectID)
	if err != nil {
		return FavoriteStatusView{}, err
	}
	if !removed {
		return FavoriteStatusView{}, ErrNotFound
	}
	LogActivity(ctx, f.Activity, &userID, entity.FavoriteRemove, "project", projectID.String(), meta, ActivityMeta(map[string]string{
		"project_id": projectID.String(),
	}))
	return FavoriteStatusView{ProjectID: projectID, IsFavorited: false}, nil
}
