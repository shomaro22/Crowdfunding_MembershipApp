package entity

import "strings"

type UserRole string

const (
	UserRoleBuyer  UserRole = "BUYER"
	UserRoleSeller UserRole = "SELLER"
	UserRoleAdmin  UserRole = "ADMIN"
)

//roleのs
func (r UserRole) Valid() bool {
	switch r {
	case UserRoleBuyer, UserRoleSeller, UserRoleAdmin:
		return true
	default:
		return false
	}
}

type AccountStatus string

const (
	AccountStatusPending   AccountStatus = "PENDING"
	AccountStatusActive    AccountStatus = "ACTIVE"
	AccountStatusSuspended AccountStatus = "SUSPENDED"
	AccountStatusBanned    AccountStatus = "BANNED"
)

//SellerStatusのs
func (s AccountStatus) Valid() bool {
	switch s {
	case AccountStatusPending, AccountStatusActive, AccountStatusSuspended, AccountStatusBanned:
		return true
	default:
		return false
	}
}

func (s AccountStatus) CanLogin() bool {
	return s == AccountStatusActive
}

type SellerStatus string

const (
	SellerStatusNotApplicable SellerStatus = "NOT_APPLICABLE"
	SellerStatusPending       SellerStatus = "PENDING"
	SellerStatusActive        SellerStatus = "ACTIVE"
	SellerStatusRejected      SellerStatus = "REJECTED"
	SellerStatusSuspended     SellerStatus = "SUSPENDED"
)

func (s SellerStatus) Valid() bool {
	switch s {
	case SellerStatusNotApplicable, SellerStatusPending, SellerStatusActive,
		SellerStatusRejected, SellerStatusSuspended:
		return true
	default:
		return false
	}
}

type AdminRole string

const (
	AdminRoleSuperAdmin   AdminRole = "SUPER_ADMIN"
	AdminRoleReviewAdmin  AdminRole = "REVIEW_ADMIN"
	AdminRoleSupportAdmin AdminRole = "SUPPORT_ADMIN"
	AdminRoleFinanceAdmin AdminRole = "FINANCE_ADMIN"
)

func (r AdminRole) Valid() bool {
	switch r {
	case AdminRoleSuperAdmin, AdminRoleReviewAdmin, AdminRoleSupportAdmin, AdminRoleFinanceAdmin:
		return true
	default:
		return false
	}
}

type OAuthProvider string

const OAuthProviderGoogle OAuthProvider = "google"

//providerのp
func (p OAuthProvider) Valid() bool {
	return p == OAuthProviderGoogle
}

type ProjectStatus string

const (
	ProjectStatusPending         ProjectStatus = "PENDING"
	ProjectStatusActive          ProjectStatus = "ACTIVE"
	ProjectStatusCompleted       ProjectStatus = "COMPLETED"
	ProjectStatusExpired         ProjectStatus = "EXPIRED"
	ProjectStatusCancelled       ProjectStatus = "CANCELLED"
	ProjectStatusDeleteRequested ProjectStatus = "DELETE_REQUESTED"
	ProjectStatusDeleted         ProjectStatus = "DELETED"
)

func (s ProjectStatus) Valid() bool {
	switch s {
	case ProjectStatusPending, ProjectStatusActive, ProjectStatusCompleted,
		ProjectStatusExpired, ProjectStatusCancelled, ProjectStatusDeleteRequested, ProjectStatusDeleted:
		return true
	default:
		return false
	}
}

type DurationType string

const (
	DurationUnlimited DurationType = "UNLIMITED"
	DurationLimited   DurationType = "LIMITED"
)

//DurationTypeのd
func (d DurationType) Valid() bool {
	return d == DurationUnlimited || d == DurationLimited
}

type MembershipType string

const (
	MembershipBronze MembershipType = "BRONZE"
	MembershipSilver MembershipType = "SILVER"
	MembershipGold   MembershipType = "GOLD"
)

//MembershipTypeのm
func (m MembershipType) Valid() bool {
	switch m {
	case MembershipBronze, MembershipSilver, MembershipGold:
		return true
	default:
		return false
	}
}

func (m MembershipType) Rank() int {
	switch m {
	case MembershipBronze:
		return 1
	case MembershipSilver:
		return 2
	case MembershipGold:
		return 3
	default:
		return 0
	}
}

func (m MembershipType) MeetsMinimum(required MembershipType) bool {
	return m.Rank() >= required.Rank()
}

type SupporterStatus string

const (
	SupporterStatusActive         SupporterStatus = "ACTIVE"
	SupporterStatusPendingRenewal SupporterStatus = "PENDING_RENEWAL" // KONBINI 更新未払い。周期満了まで特典利用可
	SupporterStatusExpired         SupporterStatus = "EXPIRED"
	SupporterStatusCancelled       SupporterStatus = "CANCELLED"
)

func (s SupporterStatus) Valid() bool {
	switch s {
	case SupporterStatusActive, SupporterStatusPendingRenewal,
		SupporterStatusExpired, SupporterStatusCancelled:
		return true
	default:
		return false
	}
}

type BillingMode string

const (
	BillingModeAuto    BillingMode = "AUTO"
	BillingModeKonbini BillingMode = "KONBINI"
)

//BillingModeのb
func (b BillingMode) Valid() bool {
	return b == BillingModeAuto || b == BillingModeKonbini
}

type PurchaseStatus string

const (
	PurchaseStatusPending   PurchaseStatus = "PENDING"
	PurchaseStatusPaid      PurchaseStatus = "PAID"
	PurchaseStatusCancelled PurchaseStatus = "CANCELLED"
	PurchaseStatusRefunded  PurchaseStatus = "REFUNDED"
	PurchaseStatusDisputed  PurchaseStatus = "DISPUTED" // OpenAPI 未定義、仕様準拠
	PurchaseStatusFailed    PurchaseStatus = "FAILED"   // OpenAPI 未定義、仕様準拠
)

func (s PurchaseStatus) Valid() bool {
	switch s {
	case PurchaseStatusPending, PurchaseStatusPaid, PurchaseStatusCancelled,
		PurchaseStatusRefunded, PurchaseStatusDisputed, PurchaseStatusFailed:
		return true
	default:
		return false
	}
}

type PaymentMethod string

const (
	PaymentMethodCard      PaymentMethod = "CARD"
	PaymentMethodKonbini   PaymentMethod = "KONBINI"
	PaymentMethodApplePay  PaymentMethod = "APPLE_PAY"
	PaymentMethodGooglePay PaymentMethod = "GOOGLE_PAY"
)

func (m PaymentMethod) Valid() bool {
	switch m {
	case PaymentMethodCard, PaymentMethodKonbini, PaymentMethodApplePay, PaymentMethodGooglePay:
		return true
	default:
		return false
	}
}

type RegStatus string

const (
	RegPending  RegStatus = "PENDING"
	RegApproved RegStatus = "APPROVED"
	RegRejected RegStatus = "REJECTED"
)

func (s RegStatus) Valid() bool {
	switch s {
	case RegPending, RegApproved, RegRejected:
		return true
	default:
		return false
	}
}

type IDDocType string

const (
	IDDriversLicense IDDocType = "DRIVERS_LICENSE"
	IDMyNumberCard   IDDocType = "MY_NUMBER_CARD"
	IDResidenceCard  IDDocType = "RESIDENCE_CARD"
	IDPassport       IDDocType = "PASSPORT"
	IDResidentRecord IDDocType = "RESIDENT_RECORD"
)

func (t IDDocType) Valid() bool {
	switch t {
	case IDDriversLicense, IDMyNumberCard,
		IDResidenceCard, IDPassport, IDResidentRecord:
		return true
	default:
		return false
	}
}

func (t IDDocType) NeedsBack() bool {
	return t == IDDriversLicense || t == IDResidenceCard
}

type ReportStatus string

const (
	ReportStatusPending  ReportStatus = "PENDING"
	ReportStatusInReview ReportStatus = "IN_REVIEW"
	ReportStatusResolved ReportStatus = "RESOLVED"
	ReportStatusDismissed ReportStatus = "DISMISSED"
)

func (s ReportStatus) Valid() bool {
	switch s {
	case ReportStatusPending, ReportStatusInReview, ReportStatusResolved, ReportStatusDismissed:
		return true
	default:
		return false
	}
}

type ReportTargetType string

const (
	ReportTargetProject ReportTargetType = "PROJECT"
	ReportTargetMessage ReportTargetType = "MESSAGE"
	ReportTargetUser    ReportTargetType = "USER"
)

func (t ReportTargetType) Valid() bool {
	switch t {
	case ReportTargetProject, ReportTargetMessage, ReportTargetUser:
		return true
	default:
		return false
	}
}

type BatchStatus string

const (
	BatchRunning BatchStatus = "RUNNING"
	BatchSuccess BatchStatus = "SUCCESS"
	BatchFailed  BatchStatus = "FAILED"
)

func (s BatchStatus) Valid() bool {
	switch s {
	case BatchRunning, BatchSuccess, BatchFailed:
		return true
	default:
		return false
	}
}

type ActivityAction string

const (
	UserLogin          ActivityAction = "user.login"
	UserLoginFail      ActivityAction = "user.login_failed"
	UserLogout         ActivityAction = "user.logout"
	UserRegister       ActivityAction = "user.register"
	SellerRegister     ActivityAction = "user.register_seller"
	FavoriteAdd        ActivityAction = "favorite.add"
	FavoriteRemove     ActivityAction = "favorite.remove"
	PurchaseCreated    ActivityAction = "purchase.created"
	PurchaseCancelled  ActivityAction = "purchase.cancelled"
	PurchasePlanChange ActivityAction = "purchase.plan_change"
	PurchaseRefunded   ActivityAction = "purchase.refunded"
	PurchaseDisputed   ActivityAction = "purchase.disputed"
	MessageSent        ActivityAction = "message.sent"
	MessageRead        ActivityAction = "message.read"
	ProjectCreated     ActivityAction = "project.created"
	ProjectUpdated     ActivityAction = "project.updated"
	PaymentCheckout    ActivityAction = "payment.checkout_started"
	PaymentWebhook     ActivityAction = "payment.webhook_received"
	ReportCreated      ActivityAction = "report.created"
	SecRateLimit       ActivityAction = "security.rate_limited"
	SecWSConnect       ActivityAction = "security.ws_connect"
	SecWSSubscribe     ActivityAction = "security.ws_subscribe"
	Sec2FAOn           ActivityAction = "security.2fa_enabled"
	Sec2FAFail         ActivityAction = "security.2fa_failed"
)

func (a ActivityAction) Valid() bool {
	switch a {
	case UserLogin, UserLoginFail, UserLogout,
		UserRegister, SellerRegister,
		FavoriteAdd, FavoriteRemove,
		PurchaseCreated, PurchaseCancelled, PurchasePlanChange,
		PurchaseRefunded, PurchaseDisputed,
		MessageSent, MessageRead,
		ProjectCreated, ProjectUpdated,
		PaymentCheckout, PaymentWebhook,
		ReportCreated,
		SecRateLimit, SecWSConnect, SecWSSubscribe,
		Sec2FAOn, Sec2FAFail:
		return true
	default:
		return false
	}
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
