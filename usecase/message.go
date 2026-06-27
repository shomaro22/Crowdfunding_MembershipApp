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

type SendMessageIn struct {
	ProjectID  uuid.UUID
	ReceiverID uuid.UUID
	Content    string
	Meta       RequestMeta
}

type MessageUseCase struct {
	Messages   repository.MessageRepo
	Projects   repository.ProjectRepo
	Supporters repository.SupporterRepo
	Users      repository.UserRepo
	Activity   repository.ActivityRepo
	DB         *gorm.DB
	events     MessageEventHook
}

func (m *MessageUseCase) SetEventHook(h MessageEventHook) {
	m.events = h
}

func threadKey(projectID, partnerID uuid.UUID) string {
	return projectID.String() + "|" + partnerID.String()
}

func (m *MessageUseCase) ListThreads(ctx context.Context, userID uuid.UUID) ([]MessageThreadView, error) {
	var (
		rows    []entity.Message
		err     error
		threads map[string]*MessageThreadView
		order   []string
		i       int
		msg     entity.Message
		partner uuid.UUID
		key     string
		t       *MessageThreadView
		out     []MessageThreadView
	)
	rows, err = m.Messages.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	threads = make(map[string]*MessageThreadView)
	for i = range rows {
		msg = rows[i]
		if msg.SenderID == userID {
			partner = msg.ReceiverID
		} else {
			partner = msg.SenderID
		}
		key = threadKey(msg.ProjectID, partner)
		t = threads[key]
		if t == nil {
			t = &MessageThreadView{
				ProjectID: msg.ProjectID,
				Partner:   UserView{ID: partner},
				Messages:  make([]MessageItemView, 0),
			}
			threads[key] = t
			order = append(order, key)
		}
		if !msg.IsRead && msg.ReceiverID == userID {
			t.UnreadCount++
		}
		t.Messages = append(t.Messages, MessageItemFrom(&msg))
	}
	out = make([]MessageThreadView, 0, len(order))
	for i = range order {
		t = threads[order[i]]
		if err = m.enrichThread(ctx, t); err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, nil
}

func (m *MessageUseCase) enrichThread(ctx context.Context, t *MessageThreadView) error {
	var (
		project *entity.Project
		u       *entity.User
		err     error
	)
	if t == nil {
		return nil
	}
	project, _, _, err = m.Projects.FindID(ctx, t.ProjectID)
	if err != nil || project == nil {
		return ErrNotFound
	}
	t.ProjectTitle = project.Title
	u, err = m.Users.FindID(ctx, t.Partner.ID)
	if err != nil || u == nil {
		return ErrNotFound
	}
	t.Partner = UserViewFrom(u)
	reverseMessageItems(t.Messages)
	return nil
}

func reverseMessageItems(items []MessageItemView) {
	var (
		i int
		j int
	)
	for i, j = 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

func (m *MessageUseCase) Send(ctx context.Context, senderID uuid.UUID, in SendMessageIn) (MessageItemView, error) {
	var (
		project *entity.Project
		sender  *entity.User
		receiver *entity.User
		row     *entity.Message
		now     time.Time
		out     MessageItemView
		err     error
	)
	in.Content = strings.TrimSpace(in.Content)
	if err = entity.ValidateMessage(in.Content); err != nil {
		return MessageItemView{}, mapMessageEntityErr(err)
	}
	project, err = m.authorizeSend(ctx, senderID, in.ProjectID, in.ReceiverID)
	if err != nil {
		return MessageItemView{}, err
	}
	_ = project
	now = time.Now().UTC()
	row = &entity.Message{
		ID:         uuid.New(),
		ProjectID:  in.ProjectID,
		SenderID:   senderID,
		ReceiverID: in.ReceiverID,
		Content:    in.Content,
		IsRead:     false,
		CreatedAt:  now,
	}
	if err = m.Messages.Create(ctx, row); err != nil {
		return MessageItemView{}, err
	}
	LogActivity(ctx, m.Activity, &senderID, entity.MessageSent, "message", row.ID.String(), in.Meta, activityMeta(map[string]string{
		"project_id":  in.ProjectID.String(),
		"receiver_id": in.ReceiverID.String(),
	}))
	sender, _ = m.Users.FindID(ctx, senderID)
	receiver, _ = m.Users.FindID(ctx, in.ReceiverID)
	out = MessageItemWithUsers(row, UserViewPtr(sender), UserViewPtr(receiver))
	if m.events != nil {
		m.events.OnMessageCreated(ctx, out)
	}
	return out, nil
}

func (m *MessageUseCase) MarkRead(ctx context.Context, userID, messageID uuid.UUID, meta RequestMeta) error {
	var (
		msg *entity.Message
		now time.Time
		err error
	)
	msg, err = m.Messages.FindID(ctx, messageID)
	if err != nil || msg == nil {
		return ErrNotFound
	}
	if err = m.authorizeReadParticipant(msg, userID); err != nil {
		return err
	}
	if msg.ReceiverID != userID {
		return ErrForbidden
	}
	if msg.IsRead {
		if m.events != nil {
			m.events.OnMessageRead(ctx, msg.ProjectID, msg.ID, userID)
		}
		return nil
	}
	now = time.Now().UTC()
	if err = m.Messages.MarkRead(ctx, messageID, now); err != nil {
		return err
	}
	LogActivity(ctx, m.Activity, &userID, entity.MessageRead, "message", msg.ID.String(), meta, activityMeta(map[string]string{
		"project_id": msg.ProjectID.String(),
	}))
	if m.events != nil {
		m.events.OnMessageRead(ctx, msg.ProjectID, msg.ID, userID)
	}
	return nil
}

func (m *MessageUseCase) GetNotifications(ctx context.Context, userID uuid.UUID) (NotificationsView, error) {
	var (
		n   int64
		err error
	)
	n, err = m.Messages.CountUnread(ctx, userID)
	if err != nil {
		return NotificationsView{}, err
	}
	return NotificationsView{UnreadMessages: n}, nil
}
