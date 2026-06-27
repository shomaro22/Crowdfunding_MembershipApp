package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)


func (m *MessageUseCase) loadProject(ctx context.Context, projectID uuid.UUID) (*entity.Project, error) {
	var (
		project *entity.Project
		err     error
	)
	project, _, _, err = m.Projects.FindID(ctx, projectID)
	if err != nil || project == nil {
		return nil, ErrNotFound
	}
	return project, nil
}

func (m *MessageUseCase) requireActiveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	var (
		supporter *entity.ProjectSupporter
		err       error
	)
	supporter, err = m.Supporters.FindMembership(ctx, projectID, userID)
	if err != nil || supporter == nil {
		return ErrForbidden
	}
	if err = entity.CanMessage(*supporter, time.Now().UTC()); err != nil {
		return ErrForbidden
	}
	return nil
}

func (m *MessageUseCase) CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error {
	var (
		project *entity.Project
		err     error
	)
	project, err = m.loadProject(ctx, projectID)
	if err != nil {
		return err
	}
	if project.SellerID == userID {
		return nil
	}
	return m.requireActiveMember(ctx, projectID, userID)
}

func (m *MessageUseCase) authorizeSend(
	ctx context.Context,
	senderID, projectID, receiverID uuid.UUID,
) (*entity.Project, error) {
	var (
		project   *entity.Project
		err       error
	)
	project, err = m.loadProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.SellerID == senderID {
		if receiverID == senderID {
			return nil, ErrBadRequest
		}
		if err = m.requireActiveMember(ctx, projectID, receiverID); err != nil {
			return nil, err
		}
		return project, nil
	}
	if receiverID != project.SellerID {
		return nil, ErrForbidden
	}
	if err = m.requireActiveMember(ctx, projectID, senderID); err != nil {
		return nil, err
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
