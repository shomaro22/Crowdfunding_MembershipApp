package controller

import (
	"context"
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type MessageUseCase interface {
	ListThreads(ctx context.Context, userID uuid.UUID) ([]usecase.MessageThreadView, error)
	Send(ctx context.Context, senderID uuid.UUID, in usecase.SendMessageIn) (usecase.MessageItemView, error)
	MarkRead(ctx context.Context, userID, messageID uuid.UUID, meta usecase.RequestMeta) error
	GetNotifications(ctx context.Context, userID uuid.UUID) (usecase.NotificationsView, error)
}

type MessageController struct {
	Messages MessageUseCase
}

func NewMessageController(messages MessageUseCase) *MessageController {
	return &MessageController{Messages: messages}
}

type sendMessageRequestJSON struct {
	ProjectID  string `json:"project_id"`
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
}

func (h *MessageController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		rows   []usecase.MessageThreadView
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	rows, err = h.Messages.ListThreads(c.Request().Context(), userID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, listMessageThreadsJSON(rows))
}

func (h *MessageController) Send(c echo.Context) error {
	var (
		userID uuid.UUID
		body   sendMessageRequestJSON
		in     usecase.SendMessageIn
		out    usecase.MessageItemView
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	in, err = body.toInput()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	in.Meta = requestMeta(c)
	out, err = h.Messages.Send(c.Request().Context(), userID, in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusCreated, messageItemToJSON(out))
}

func (h *MessageController) MarkRead(c echo.Context) error {
	var (
		userID    uuid.UUID
		messageID uuid.UUID
		err       error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	messageID, err = uuid.Parse(c.Param("messageId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid message id", "BAD_REQUEST"))
	}
	err = h.Messages.MarkRead(c.Request().Context(), userID, messageID, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.NoContent(http.StatusOK)
}

func (h *MessageController) Notifications(c echo.Context) error {
	var (
		userID uuid.UUID
		view   usecase.NotificationsView
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	view, err = h.Messages.GetNotifications(c.Request().Context(), userID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, notificationsResponse{UnreadMessages: view.UnreadMessages})
}

func (body sendMessageRequestJSON) toInput() (usecase.SendMessageIn, error) {
	var (
		projectID  uuid.UUID
		receiverID uuid.UUID
		err        error
	)
	projectID, err = uuid.Parse(body.ProjectID)
	if err != nil {
		return usecase.SendMessageIn{}, err
	}
	receiverID, err = uuid.Parse(body.ReceiverID)
	if err != nil {
		return usecase.SendMessageIn{}, err
	}
	return usecase.SendMessageIn{
		ProjectID:  projectID,
		ReceiverID: receiverID,
		Content:    body.Content,
	}, nil
}
