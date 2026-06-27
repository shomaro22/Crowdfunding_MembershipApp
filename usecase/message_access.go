package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func (m *MessageUseCase) CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error {
	var (
		project   *entity.Project
		supporter *entity.ProjectSupporter
		err       error
		now       time.Time
	)
	project, _, _, err = m.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return ErrNotFound
	}
	if project.SellerID == userID {
		return nil
	}
	supporter, err = m.Supporters.FindMembership(ctx, projectID, userID)
	if err != nil || supporter == nil {
		return ErrForbidden
	}
	now = time.Now().UTC()
	if err = entity.CanMessage(*supporter, now); err != nil {
		return ErrForbidden
	}
	return nil
}

func (m *MessageUseCase) authorizeSend(
	ctx context.Context,
	senderID, projectID, receiverID uuid.UUID,
) (*entity.Project, error) {
	var (
		project   *entity.Project
		supporter *entity.ProjectSupporter
		err       error
		now       time.Time
	)
	project, _, _, err = m.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return nil, ErrNotFound
	}
	if project.SellerID == senderID {
		if receiverID == senderID {
			return nil, ErrBadRequest
		}
		supporter, err = m.Supporters.FindMembership(ctx, projectID, receiverID)
		if err != nil || supporter == nil {
			return nil, ErrForbidden
		}
		now = time.Now().UTC()
		if err = entity.CanMessage(*supporter, now); err != nil {
			return nil, ErrForbidden
		}
		return project, nil
	}
	if receiverID != project.SellerID {
		return nil, ErrForbidden
	}
	supporter, err = m.Supporters.FindMembership(ctx, projectID, senderID)
	if err != nil || supporter == nil {
		return nil, ErrForbidden
	}
	now = time.Now().UTC()
	if err = entity.CanMessage(*supporter, now); err != nil {
		return nil, ErrForbidden
	}
	return project, nil
}

func (m *MessageUseCase) authorizeReadParticipant(msg *entity.Message, userID uuid.UUID) error {
	if msg == nil {
		return ErrNotFound
	}
	if msg.SenderID != userID && msg.ReceiverID != userID {
		return ErrForbidden
	}
	return nil
}
