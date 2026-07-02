package usecase

import (
	"context"

	"crowdfund-app/model"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type UpdateProfileIn struct {
	Name      *string
	AvatarURL *string
}

func (pr *ProfileUseCase) GetUser(ctx context.Context, userID uuid.UUID) (UserView, error) {
	var (
		u   model.User
		err error
	)

	u, err = pr.deps.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	return UserViewFrom(u), nil
}

func (pr *ProfileUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileIn) (UserView, error) {
	var (
		norm validator.UpdateProfileNormalized
		u    model.User
		err  error
	)

	norm, err = validator.ValidateUpdateProfile(validator.UpdateProfileInput{
		Name:      in.Name,
		AvatarURL: in.AvatarURL,
	})
	if err != nil {
		return UserView{}, mapProfileModelErr(err)
	}
	u, err = pr.deps.Users.GetID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if norm.NameSet {
		u.Name = norm.Name
	}
	if norm.AvatarURLSet {
		u.AvatarURL = norm.AvatarURL
	}
	if err = pr.deps.Users.Update(ctx, &u); err != nil {
		return UserView{}, err
	}
	return UserViewFrom(u), nil
}
