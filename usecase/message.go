package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

const (
	defaultThreadsPage          = 1
	defaultThreadsPerPage       = 20
	maxThreadsPerPage           = 50
	defaultMessagesPerThread    = 20
	maxMessagesPerThread        = 50
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
	events     MessageEventHook
}

type MessageEventHook interface {
	OnMessageCreated(ctx context.Context, view MessageItemView)
	OnMessageRead(ctx context.Context, projectID, messageID, readerID uuid.UUID)
}

func AttachMessageEvents(m *MessageUseCase, h MessageEventHook) {
	if m != nil {
		m.events = h
	}
}

func normalizeListThreadsQuery(q ListThreadsQuery) ListThreadsQuery {
	if q.Page < 1 {
		q.Page = defaultThreadsPage
	}
	if q.PerPage < 1 {
		q.PerPage = defaultThreadsPerPage
	}
	if q.PerPage > maxThreadsPerPage {
		q.PerPage = maxThreadsPerPage
	}
	if q.MessagesPerThread < 1 {
		q.MessagesPerThread = defaultMessagesPerThread
	}
	if q.MessagesPerThread > maxMessagesPerThread {
		q.MessagesPerThread = maxMessagesPerThread
	}
	return q
}

func (m *MessageUseCase) ListThreads(ctx context.Context, userID uuid.UUID, q ListThreadsQuery) (ListThreadsResult, error) {
	var (
		qNorm    ListThreadsQuery
		summaries []repository.MessageThreadSummary
		total    int64
		offset   int
		rows     []entity.Message
		out      ListThreadsResult
		i        int
		s        repository.MessageThreadSummary
		t        MessageThreadView
		err      error
	)
	qNorm = normalizeListThreadsQuery(q)
	offset = (qNorm.Page - 1) * qNorm.PerPage
	summaries, total, err = m.Messages.ListThreadSummaries(ctx, userID, offset, qNorm.PerPage)
	if err != nil {
		return ListThreadsResult{}, err
	}
	out = ListThreadsResult{
		Total:   total,
		Page:    qNorm.Page,
		PerPage: qNorm.PerPage,
		Threads: make([]MessageThreadView, 0, len(summaries)),
	}
	for i = range summaries {
		s = summaries[i]
		if err = m.CanSubscribeProject(ctx, s.ProjectID, userID); err != nil {
			continue
		}
		rows, err = m.Messages.ListMessagesInThread(ctx, userID, s.ProjectID, s.PartnerID, qNorm.MessagesPerThread)
		if err != nil {
			return ListThreadsResult{}, err
		}
		t = MessageThreadView{
			ProjectID:   s.ProjectID,
			Partner:     UserView{ID: s.PartnerID},
			UnreadCount: s.UnreadCount,
			Messages:    make([]MessageItemView, 0, len(rows)),
		}
		for _, msg := range rows {
			t.Messages = append(t.Messages, MessageItemFrom(&msg))
		}
		if err = m.enrichThread(ctx, &t); err != nil {
			return ListThreadsResult{}, err
		}
		out.Threads = append(out.Threads, t)
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
	if err != nil {
		return mapRepoNotFound(err)
	}
	t.ProjectTitle = project.Title
	u, err = m.Users.FindID(ctx, t.Partner.ID)
	if err != nil {
		return mapRepoNotFound(err)
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
		sender   *entity.User
		receiver *entity.User
		row      *entity.Message
		now      time.Time
		out      MessageItemView
		err      error
	)
	in.Content = strings.TrimSpace(in.Content)
	if err = validator.ValidateMessage(in.Content); err != nil {
		return MessageItemView{}, mapMessageEntityErr(err)
	}
	if _, err = m.authorizeSend(ctx, senderID, in.ProjectID, in.ReceiverID); err != nil {
		return MessageItemView{}, err
	}
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
	if err != nil {
		return mapRepoNotFound(err)
	}
	if err = m.authorizeReadParticipant(msg, userID); err != nil {
		return err
	}
	if err = m.CanSubscribeProject(ctx, msg.ProjectID, userID); err != nil {
		return err
	}
	if msg.ReceiverID != userID {
		return ErrForbidden
	}
	if msg.IsRead {
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

func (m *MessageUseCase) ConfirmWSSubscribe(ctx context.Context, userID, projectID uuid.UUID, meta RequestMeta) error {
	if err := m.CanSubscribeProject(ctx, projectID, userID); err != nil {
		return err
	}
	LogActivity(ctx, m.Activity, &userID, entity.SecWSSubscribe, "project", projectID.String(), meta, nil)
	return nil
}
