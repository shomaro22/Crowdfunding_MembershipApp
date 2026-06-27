package controller

import (
	"net/http"
	"strconv"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type MessageController struct {
	Messages MessageUseCase
}

func NewMessageController(messages MessageUseCase) *MessageController {
	return &MessageController{Messages: messages}
}

func (h *MessageController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		result usecase.ListThreadsResult
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	result, err = h.Messages.ListThreads(c.Request().Context(), userID, listThreadsQueryFrom(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, ListMessageThreadsWireFrom(result))
}

func listThreadsQueryFrom(c echo.Context) usecase.ListThreadsQuery {
	var (
		page              int
		perPage           int
		messagesPerThread int
		err               error
	)
	page, err = strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage, err = strconv.Atoi(c.QueryParam("per_page"))
	if err != nil || perPage < 1 {
		perPage = 20
	}
	messagesPerThread, err = strconv.Atoi(c.QueryParam("messages_per_thread"))
	if err != nil || messagesPerThread < 1 {
		messagesPerThread = 20
	}
	return usecase.ListThreadsQuery{
		Page:              page,
		PerPage:           perPage,
		MessagesPerThread: messagesPerThread,
	}
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
	return c.JSON(http.StatusCreated, MessageItemWireFrom(out))
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
	return c.JSON(http.StatusOK, NotificationsWire{UnreadMessages: view.UnreadMessages})
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
