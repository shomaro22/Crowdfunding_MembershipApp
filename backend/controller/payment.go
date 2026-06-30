package controller

import (
	"io"
	"net/http"
	"strings"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PaymentController struct {
	Payments     usecase.IPaymentUsecase
	Purchases    usecase.IPurchaseUsecase
	StripeAppURL string
}

func NewPaymentController(payments usecase.IPaymentUsecase, purchases usecase.IPurchaseUsecase, stripeAppURL string) *PaymentController {
	return &PaymentController{Payments: payments, Purchases: purchases, StripeAppURL: stripeAppURL}
}

type checkoutRequestJSON struct {
	ProjectID     string  `json:"project_id"`
	PlanID        string  `json:"plan_id"`
	PaymentMethod *string `json:"payment_method"`
	SuccessURL    *string `json:"success_url"`
	CancelURL     *string `json:"cancel_url"`
}

type intentRequestJSON struct {
	ProjectID string `json:"project_id"`
	PlanID    string `json:"plan_id"`
}

func (h *PaymentController) Checkout(c echo.Context) error {
	var (
		buyerID uuid.UUID
		body    checkoutRequestJSON
		in      usecase.CheckoutIn
		out     usecase.CheckoutOut
		err     error
	)
	buyerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	in, err = body.toInput(buyerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	in.Meta = requestMeta(c)
	in.FallbackSuccessURL = strings.TrimRight(h.StripeAppURL, "/") + "/support/success?session_id={CHECKOUT_SESSION_ID}"
	in.FallbackCancelURL = strings.TrimRight(h.StripeAppURL, "/") + "/support/cancel"
	out, err = h.Payments.CreateCheckout(reqCtx(c), in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]string{
		"session_id": out.SessionID,
		"url":        out.URL,
	})
}

func (h *PaymentController) Intent(c echo.Context) error {
	var (
		buyerID uuid.UUID
		body    intentRequestJSON
		in      usecase.IntentIn
		out     usecase.IntentOut
		err     error
	)
	buyerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	in, err = body.toInput(buyerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	in.Meta = requestMeta(c)
	out, err = h.Payments.CreateIntent(reqCtx(c), in)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]string{
		"client_secret":     out.ClientSecret,
		"payment_intent_id": out.PaymentIntentID,
	})
}

func (h *PaymentController) Webhook(c echo.Context) error {
	var (
		payload []byte
		err     error
	)
	payload, err = io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid payload", "BAD_REQUEST"))
	}
	err = h.Payments.HandleWebhook(reqCtx(c), payload, c.Request().Header.Get("Stripe-Signature"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("webhook rejected", "BAD_REQUEST"))
	}
	return noContent(c)
}

type verifyCheckoutJSON struct {
	SessionID string `json:"session_id"`
}

type verifyIntentJSON struct {
	PaymentIntentID string `json:"payment_intent_id"`
}

func (h *PaymentController) VerifyCheckout(c echo.Context) error {
	var (
		buyerID uuid.UUID
		body    verifyCheckoutJSON
		view    usecase.SupporterView
		err     error
	)
	buyerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	view, err = h.Payments.VerifyCheckout(reqCtx(c), buyerID, body.SessionID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, supporterToJSON(view))
}

func (h *PaymentController) VerifyIntent(c echo.Context) error {
	var (
		buyerID uuid.UUID
		body    verifyIntentJSON
		view    usecase.SupporterView
		err     error
	)
	buyerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	view, err = h.Payments.VerifyIntent(reqCtx(c), buyerID, body.PaymentIntentID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, supporterToJSON(view))
}

func (body checkoutRequestJSON) toInput(buyerID uuid.UUID) (usecase.CheckoutIn, error) {
	var (
		projectID uuid.UUID
		planID    uuid.UUID
		err       error
	)
	projectID, err = uuid.Parse(body.ProjectID)
	if err != nil {
		return usecase.CheckoutIn{}, err
	}
	planID, err = uuid.Parse(body.PlanID)
	if err != nil {
		return usecase.CheckoutIn{}, err
	}
	in := usecase.CheckoutIn{
		BuyerID:    buyerID,
		ProjectID:  projectID,
		PlanID:     planID,
		SuccessURL: body.SuccessURL,
		CancelURL:  body.CancelURL,
	}
	if body.PaymentMethod != nil {
		in.PaymentMethod = *body.PaymentMethod
	}
	return in, nil
}

func (body intentRequestJSON) toInput(buyerID uuid.UUID) (usecase.IntentIn, error) {
	var (
		projectID uuid.UUID
		planID    uuid.UUID
		err       error
	)
	projectID, err = uuid.Parse(body.ProjectID)
	if err != nil {
		return usecase.IntentIn{}, err
	}
	planID, err = uuid.Parse(body.PlanID)
	if err != nil {
		return usecase.IntentIn{}, err
	}
	return usecase.IntentIn{BuyerID: buyerID, ProjectID: projectID, PlanID: planID}, nil
}

func (h *PaymentController) ListPurchases(c echo.Context) error {
	var (
		userID uuid.UUID
		limit  int
		offset int
		rows   []usecase.PurchaseView
		total  int64
		err    error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	limit, offset = queryPage(c)
	rows, total, err = h.Purchases.ListMine(reqCtx(c), userID, limit, offset)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listPurchasesJSON(rows, total))
}
