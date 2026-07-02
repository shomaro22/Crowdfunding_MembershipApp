package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/model"
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
	Messages   repository.IMessageRepo
	Projects   repository.IProjectRepo
	Supporters repository.ISupporterRepo
	Sellers    repository.ISellerRepo
	Users      repository.IUserRepo
	Activity   repository.IActivityRepo
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
		qNorm ListThreadsQuery
		u     model.User
		err   error
	)
	qNorm = normalizeListThreadsQuery(q)
	u, err = m.Users.GetID(ctx, userID)
	if err != nil {
		return ListThreadsResult{}, mapRepoNotFound(err)
	}
	if u.Role == model.UserRoleSeller && m.Sellers != nil {
		return m.listSellerThreads(ctx, userID, qNorm)
	}
	return m.listBuyerThreads(ctx, userID, qNorm)
}

func (m *MessageUseCase) listBuyerThreads(ctx context.Context, userID uuid.UUID, qNorm ListThreadsQuery) (ListThreadsResult, error) {
	var (
		summaries []repository.MessageThreadSummary
		total    int64
		offset   int
		rows     []model.Message
		out      ListThreadsResult
		i        int
		j        int
		s        repository.MessageThreadSummary
		t        MessageThreadView
		msg      model.Message
		err      error
	)
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
			continue
		}
		t = MessageThreadView{
			ProjectID:   s.ProjectID,
			Partner:     UserView{ID: s.PartnerID},
			UnreadCount: s.UnreadCount,
			Messages:    make([]MessageItemView, 0, len(rows)),
		}
		for j = range rows {
			msg = rows[j]
			t.Messages = append(t.Messages, MessageItemFrom(msg))
		}
		if err = m.enrichThread(ctx, &t); err != nil {
			continue
		}
		out.Threads = append(out.Threads, t)
	}
	return out, nil
}

func (m *MessageUseCase) listSellerThreads(ctx context.Context, sellerID uuid.UUID, qNorm ListThreadsQuery) (ListThreadsResult, error) {
	var (
		supporters []model.ProjectSupporter
		total      int64
		offset     int
		rows       []model.Message
		out        ListThreadsResult
		now        time.Time
		i          int
		j          int
		sup        model.ProjectSupporter
		t          MessageThreadView
		msg        model.Message
		unread     int
		err        error
	)
	offset = (qNorm.Page - 1) * qNorm.PerPage
	supporters, total, err = m.Sellers.ListSupporters(ctx, sellerID, nil, qNorm.PerPage, offset)
	if err != nil {
		return ListThreadsResult{}, err
	}
	now = time.Now().UTC()
	out = ListThreadsResult{
		Total:   total,
		Page:    qNorm.Page,
		PerPage: qNorm.PerPage,
		Threads: make([]MessageThreadView, 0, len(supporters)),
	}
	for i = range supporters {
		sup = supporters[i]
		if !sup.HasBenefits(now) {
			continue
		}
		rows, err = m.Messages.ListMessagesInThread(ctx, sellerID, sup.ProjectID, sup.UserID, qNorm.MessagesPerThread)
		if err != nil {
			rows = nil
		}
		unread = 0
		for j = range rows {
			msg = rows[j]
			if !msg.IsRead && msg.ReceiverID == sellerID {
				unread++
			}
		}
		t = MessageThreadView{
			ProjectID:   sup.ProjectID,
			Partner:     UserView{ID: sup.UserID},
			UnreadCount: unread,
			Messages:    make([]MessageItemView, 0, len(rows)),
		}
		for j = range rows {
			msg = rows[j]
			t.Messages = append(t.Messages, MessageItemFrom(msg))
		}
		if err = m.enrichThread(ctx, &t); err != nil {
			continue
		}
		out.Threads = append(out.Threads, t)
	}
	return out, nil
}

func (m *MessageUseCase) enrichThread(ctx context.Context, t *MessageThreadView) error {
	var (
		project model.Project
		u       model.User
		err     error
	)
	if t == nil {
		return nil
	}
	project, _, _, err = m.Projects.GetID(ctx, t.ProjectID)
	if err != nil {
		return mapRepoNotFound(err)
	}
	t.ProjectTitle = project.Title
	u, err = m.Users.GetID(ctx, t.Partner.ID)
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
		sender   model.User
		receiver model.User
		row      *model.Message
		now      time.Time
		out      MessageItemView
		err      error
	)
	in.Content = strings.TrimSpace(in.Content)
	if err = validator.ValidateMessage(in.Content); err != nil {
		return MessageItemView{}, mapMessageModelErr(err)
	}
	if _, err = m.authorizeSend(ctx, senderID, in.ProjectID, in.ReceiverID); err != nil {
		return MessageItemView{}, err
	}
	now = time.Now().UTC()
	row = &model.Message{
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
	LogActivity(ctx, m.Activity, &senderID, model.MessageSent, "message", row.ID.String(), in.Meta, activityMeta(map[string]string{
		"project_id":  in.ProjectID.String(),
		"receiver_id": in.ReceiverID.String(),
	}))
	sender, err = m.Users.GetID(ctx, senderID)
	if err != nil {
		return MessageItemView{}, mapRepoNotFound(err)
	}
	receiver, err = m.Users.GetID(ctx, in.ReceiverID)
	if err != nil {
		return MessageItemView{}, mapRepoNotFound(err)
	}
	out = MessageItemWithUsers(*row, UserViewPtr(sender), UserViewPtr(receiver))
	if m.events != nil {
		m.events.OnMessageCreated(ctx, out)
	}
	return out, nil
}

func (m *MessageUseCase) MarkRead(ctx context.Context, userID, messageID uuid.UUID, meta RequestMeta) error {
	var (
		msg model.Message
		now time.Time
		err error
	)
	msg, err = m.Messages.GetID(ctx, messageID)
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
	LogActivity(ctx, m.Activity, &userID, model.MessageRead, "message", msg.ID.String(), meta, activityMeta(map[string]string{
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
	LogActivity(ctx, m.Activity, &userID, model.SecWSSubscribe, "project", projectID.String(), meta, nil)
	return nil
}

func (m *MessageUseCase) loadProject(ctx context.Context, projectID uuid.UUID) (model.Project, error) {
	var (
		project model.Project
		err     error
	)
	project, _, _, err = m.Projects.GetID(ctx, projectID)
	if err != nil {
		return model.Project{}, mapRepoNotFound(err)
	}
	return project, nil
}

func (m *MessageUseCase) requireActiveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	var (
		supporter model.ProjectSupporter
		err       error
	)
	supporter, err = m.Supporters.GetMembership(ctx, projectID, userID)
	if err != nil {
		return mapRepoNotFoundAs(err, ErrForbidden)
	}
	if err = model.CanMessage(supporter, time.Now().UTC()); err != nil {
		return ErrForbidden
	}
	return nil
}

func (m *MessageUseCase) CanSubscribeProject(ctx context.Context, projectID, userID uuid.UUID) error {
	var (
		project model.Project
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
) (model.Project, error) {
	var (
		project model.Project
		err     error
	)
	project, err = m.loadProject(ctx, projectID)
	if err != nil {
		return model.Project{}, err
	}
	if project.SellerID == senderID {
		if receiverID == senderID {
			return model.Project{}, ErrBadRequest
		}
		if err = m.requireActiveMember(ctx, projectID, receiverID); err != nil {
			return model.Project{}, err
		}
		return project, nil
	}
	if receiverID != project.SellerID {
		return model.Project{}, ErrForbidden
	}
	if err = m.requireActiveMember(ctx, projectID, senderID); err != nil {
		return model.Project{}, err
	}
	return project, nil
}

func (m *MessageUseCase) authorizeReadParticipant(msg model.Message, userID uuid.UUID) error {
	if msg.ID == uuid.Nil {
		return ErrNotFound
	}
	if msg.SenderID != userID && msg.ReceiverID != userID {
		return ErrForbidden
	}
	return nil
}
