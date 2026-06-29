package usecase

import (
	"context"

	"crowdfund-app/entity"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type UpdateProfileIn struct {
	Name      *string
	AvatarURL *string
}

func (a *AuthUseCase) GetUser(ctx context.Context, userID uuid.UUID) (UserView, error) {
	var (
		u   *entity.User
		err error
	)

	u, err = a.Users.FindID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	return UserViewFrom(u), nil
}

func (a *AuthUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileIn) (UserView, error) {
	var (
		norm validator.UpdateProfileNormalized
		u    *entity.User
		err  error
	)

	norm, err = validator.ValidateUpdateProfile(validator.UpdateProfileInput{
		Name:      in.Name,
		AvatarURL: in.AvatarURL,
	})
	if err != nil {
		return UserView{}, mapProfileEntityErr(err)
	}
	u, err = a.Users.FindID(ctx, userID)
	if err != nil {
		return UserView{}, mapRepoNotFound(err)
	}
	if norm.NameSet {
		u.Name = norm.Name
	}
	if norm.AvatarURLSet {
		u.AvatarURL = norm.AvatarURL
	}
	if err = a.Users.Update(ctx, u); err != nil {
		return UserView{}, err
	}
	return UserViewFrom(u), nil
}
