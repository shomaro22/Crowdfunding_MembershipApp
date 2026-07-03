package usecase

import (
	"time"

	"crowdfund-app/model"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

func checkoutMetadata(sc supportContext, purchaseID uuid.UUID, pm model.PaymentMethod) map[string]string {
	var mode model.BillingMode
	if pm == model.PaymentMethodKonbini {
		mode = model.BillingModeKonbini
	} else {
		mode = model.BillingModeAuto
	}
	meta := paymentMetadata(
		purchaseID,
		sc.project.ID,
		sc.plan.ID,
		sc.buyer.ID,
		mode,
	)
	meta[stripe.MetaCheckoutPurpose] = string(sc.purpose)
	if sc.hasMembership {
		meta[stripe.MetaSupporterID] = sc.membership.ID.String()
	}
	return meta
}

func paymentMetadata(purchaseID, projectID, planID, buyerID uuid.UUID, mode model.BillingMode) map[string]string {
	return map[string]string{
		stripe.MetaPurchaseID:  purchaseID.String(),
		stripe.MetaProjectID:   projectID.String(),
		stripe.MetaPlanID:      planID.String(),
		stripe.MetaBuyerID:     buyerID.String(),
		stripe.MetaBillingMode: string(mode),
	}
}

func checkoutURL(custom *string, fallback string) string {
	if custom != nil && *custom != "" {
		return *custom
	}
	return fallback
}

func parseCheckoutPurpose(meta map[string]string) checkoutPurpose {
	if meta == nil {
		return checkoutFirstSupport
	}
	switch meta[stripe.MetaCheckoutPurpose] {
	case stripe.PurposeKonbiniRenewal:
		return checkoutKonbiniRenewal
	default:
		return checkoutFirstSupport
	}
}

func isKonbiniPurchase(purchase model.Purchase) bool {
	return purchase.PaymentMethod == model.PaymentMethodKonbini
}

func verifyPurchaseBuyer(purchase model.Purchase, buyerID uuid.UUID) error {
	if purchase.BuyerID != buyerID {
		return ErrForbidden
	}
	return nil
}

func checkPlanCapacity(plan model.ProjectPlan) error {
	if plan.MaxSupporters != nil && plan.SupporterCount >= *plan.MaxSupporters {
		return ErrForbidden
	}
	return nil
}

func canIssueKonbiniRenewal(s model.ProjectSupporter, now time.Time) bool {
	var windowStart time.Time
	if s.Status == model.SupporterStatusPendingRenewal {
		return true
	}
	if s.Status != model.SupporterStatusActive || s.BillingMode != model.BillingModeKonbini {
		return false
	}
	windowStart = s.CurrentPeriodEnd.AddDate(0, 0, -konbiniRenewalWindowDays)
	return !now.Before(windowStart)
}

func billingModeForPurchase(purchase model.Purchase) model.BillingMode {
	if purchase.PaymentMethod == model.PaymentMethodKonbini {
		return model.BillingModeKonbini
	}
	return model.BillingModeAuto
}

func findPlanForPurchase(plans []model.ProjectPlan, purchase model.Purchase) uuid.UUID {
	if len(plans) == 1 {
		return plans[0].ID
	}
	for _, plan := range plans {
		if plan.Price == purchase.TotalPrice {
			return plan.ID
		}
	}
	if len(plans) > 0 {
		return plans[0].ID
	}
	return uuid.Nil
}

func stripeString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func defaultPeriod(start, end *time.Time) periodRange {
	var now time.Time
	now = time.Now().UTC()
	if start != nil && end != nil {
		return periodRange{*start, *end}
	}
	return periodRange{now, now.AddDate(0, 0, supportPeriodDays)}
}

func konbiniFirstPeriod(paidAt time.Time) periodRange {
	return periodRange{start: paidAt, end: paidAt.AddDate(0, 0, supportPeriodDays)}
}

func konbiniRenewalPeriod(supporter model.ProjectSupporter, paidAt time.Time) periodRange {
	var start time.Time
	start = supporter.CurrentPeriodEnd
	if paidAt.After(start) {
		start = paidAt
	}
	return periodRange{start: start, end: start.AddDate(0, 0, supportPeriodDays)}
}
