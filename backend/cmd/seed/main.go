// seed は DB 投入用 CLI。
//
//	go run ./cmd/seed dev              # buyer@example.com のみ
//	go run ./cmd/seed dev -team        # buyer / seller / admin + デモプロジェクト 3 件
//	go run ./cmd/seed dev -team -reset # 既存ユーザーを上書き
//
// 環境変数: DATABASE_URL, JWT_SECRET, TEST_USER_PASSWORD,
//
//	SEED_TEST_USER_TEAM=1, SEED_TEST_USER_RESET=1
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/db"
	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

const devTOTPSecret = "JBSWY3DPEHPK3PXP"

func main() {
	_ = godotenv.Load()
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "dev":
		runDevSeed(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/seed dev [-team] [-reset]")
}

type seedAccount struct {
	email     string
	name      string
	role      model.UserRole
	adminRole *model.AdminRole
	with2FA   bool
}

type demoProject struct {
	title        string
	description  string
	targetAmount int64
	plans        []demoPlan
}

type demoPlan struct {
	membership model.MembershipType
	price      int
	benefits   string
}

func parseDevFlags(args []string) (reset, team bool) {
	reset = strings.TrimSpace(os.Getenv("SEED_TEST_USER_RESET")) == "1"
	team = strings.TrimSpace(os.Getenv("SEED_TEST_USER_TEAM")) == "1"
	for _, a := range args {
		switch a {
		case "-reset", "--reset":
			reset = true
		case "-team", "--team":
			team = true
		}
	}
	return reset, team
}

func runDevSeed(flagArgs []string) {
	var (
		reset, team    bool
		databaseURL    string
		password       string
		jwtSecret      string
		gormDB         *gorm.DB
		sqlDB          interface{ Close() error }
		hash           string
		totpEnc        string
		now            time.Time
		ctx            context.Context
		users          repository.IUserRepo
		sellerRegs     repository.ISellerRegRepo
		projects       repository.IProjectRepo
		connect        repository.IConnectRepo
		accs           []seedAccount
		sellerID       uuid.UUID
		adminID        uuid.UUID
		createdUsers   int
		updatedUsers   int
		skippedUsers   int
		createdProjects int
		skippedProjects int
		err            error
	)

	reset, team = parseDevFlags(flagArgs)

	databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@127.0.0.1:5432/crowdfund?sslmode=disable"
	}
	password = os.Getenv("TEST_USER_PASSWORD")
	if password == "" {
		password = "password123"
	}
	if len(password) < 8 {
		log.Fatal("TEST_USER_PASSWORD must be at least 8 characters")
	}
	jwtSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters")
	}

	if err = db.Up(databaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	gormDB, err = db.Open(databaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	sqlDB, err = gormDB.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()

	hash, err = auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	totpEnc, err = auth.EncryptSecret([]byte(jwtSecret), devTOTPSecret)
	if err != nil {
		log.Fatalf("totp encrypt: %v", err)
	}

	now = time.Now().UTC()
	ctx = context.Background()
	users = repository.NewUserRepo(gormDB)
	sellerRegs = repository.NewSellerRegRepo(gormDB)
	projects = repository.NewProjectRepo(gormDB)
	connect = repository.NewConnectRepo(gormDB)

	accs = []seedAccount{{
		email: "buyer@example.com",
		name:  "Demo Buyer",
		role:  model.UserRoleBuyer,
	}}
	if team {
		accs = append(accs,
			seedAccount{email: "buyer2@example.com", name: "Demo Buyer 2", role: model.UserRoleBuyer},
			seedAccount{
				email:   "seller@example.com",
				name:    "Demo Seller",
				role:    model.UserRoleSeller,
				with2FA: true,
			},
			seedAccount{
				email:     "admin@example.com",
				name:      "Demo Admin",
				role:      model.UserRoleAdmin,
				adminRole: adminRolePtr(model.AdminRoleReviewAdmin),
				with2FA:   true,
			},
		)
	}

	for _, acc := range accs {
		var (
			u      model.User
			action string
		)
		u, action, err = upsertUser(ctx, users, acc, hash, totpEnc, now, reset)
		if err != nil {
			log.Fatalf("user %s: %v", acc.email, err)
		}
		switch action {
		case "created":
			createdUsers++
		case "updated":
			updatedUsers++
		case "skipped":
			skippedUsers++
		}
		if acc.role == model.UserRoleSeller {
			sellerID = u.ID
		}
		if acc.role == model.UserRoleAdmin {
			adminID = u.ID
		}
	}

	if team && sellerID != uuid.Nil {
		if err = upsertSellerReg(ctx, sellerRegs, sellerID, adminID, now); err != nil {
			log.Fatalf("seller reg: %v", err)
		}
		if err = upsertConnect(ctx, connect, sellerID, now); err != nil {
			log.Fatalf("connect: %v", err)
		}
		for _, dp := range demoProjects() {
			var created bool
			created, err = upsertProject(ctx, projects, sellerID, dp, now, reset)
			if err != nil {
				log.Fatalf("project %q: %v", dp.title, err)
			}
			if created {
				createdProjects++
			} else {
				skippedProjects++
			}
		}
	}

	fmt.Println()
	fmt.Println("seed summary:")
	fmt.Printf("  users: created/updated=%d skipped=%d\n", createdUsers+updatedUsers, skippedUsers)
	if team {
		fmt.Printf("  projects: created=%d skipped=%d\n", createdProjects, skippedProjects)
	}
	fmt.Printf("  password (共通): %s\n", password)
	if team {
		fmt.Printf("  2FA (seller/admin): TOTP secret=%s  current code=%s\n", devTOTPSecret, currentTOTP())
		fmt.Println("  accounts:")
		fmt.Println("    buyer@example.com   (buyer)")
		fmt.Println("    buyer2@example.com  (buyer)")
		fmt.Println("    seller@example.com  (seller)")
		fmt.Println("    admin@example.com   (admin / review)")
	}
}

func adminRolePtr(r model.AdminRole) *model.AdminRole {
	return &r
}

func upsertUser(
	ctx context.Context,
	users repository.IUserRepo,
	acc seedAccount,
	hash, totpEnc string,
	now time.Time,
	reset bool,
) (model.User, string, error) {
	var (
		existing model.User
		name     *string
		u        model.User
		err      error
	)

	existing, err = users.GetEmail(ctx, acc.email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.User{}, "", err
	}
	if err == nil && !reset {
		fmt.Printf("skip (exists): %s\n", acc.email)
		return existing, "skipped", nil
	}

	n := acc.name
	name = &n
	u = model.User{
		ID:            uuid.New(),
		Email:         acc.email,
		Name:          name,
		PasswordHash:  hash,
		Role:          acc.role,
		AccountStatus: model.AccountStatusActive,
		SellerStatus:  model.SellerStatusNotApplicable,
		AdminRole:     acc.adminRole,
		TOTPEnabled:   acc.with2FA,
		TokenVersion:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if acc.role == model.UserRoleSeller {
		u.SellerStatus = model.SellerStatusActive
	}
	if acc.with2FA {
		u.TOTPSecret = &totpEnc
	}
	if err == nil {
		u.ID = existing.ID
		u.CreatedAt = existing.CreatedAt
		u.TokenVersion = existing.TokenVersion
		if err = users.Update(ctx, &u); err != nil {
			return model.User{}, "", err
		}
		fmt.Printf("reset: %s (%s)\n", acc.email, acc.role)
		return u, "updated", nil
	}
	if err = users.Create(ctx, &u); err != nil {
		return model.User{}, "", err
	}
	fmt.Printf("created: %s (%s)\n", acc.email, acc.role)
	return u, "created", nil
}

func upsertSellerReg(ctx context.Context, regs repository.ISellerRegRepo, sellerID, adminID uuid.UUID, now time.Time) error {
	var (
		reg   model.SellerReg
		front = "seed/kyc/front.jpg"
		back  = "seed/kyc/back.jpg"
		err   error
	)

	_, err = regs.GetUserID(ctx, sellerID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	reg = model.SellerReg{
		ID:           uuid.New(),
		UserID:       sellerID,
		FullName:     "Demo Seller",
		BirthDate:    time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC),
		Address:      "Tokyo, Japan",
		PhotoPresent: true,
		IDFrontKey:   &front,
		IDBackKey:    &back,
		Status:       model.RegApproved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if adminID != uuid.Nil {
		reg.ReviewedBy = &adminID
		reg.ReviewedAt = &now
	}
	if err = regs.Create(ctx, &reg); err != nil {
		return err
	}
	fmt.Println("created: seller registration (approved)")
	return nil
}

func upsertConnect(ctx context.Context, connect repository.IConnectRepo, sellerID uuid.UUID, now time.Time) error {
	var (
		row model.ConnectAccount
		err error
	)

	_, err = connect.GetUserID(ctx, sellerID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	row = model.ConnectAccount{
		ID:               uuid.New(),
		UserID:           sellerID,
		StripeAccountID:  "acct_seed_demo_seller",
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err = connect.Create(ctx, &row); err != nil {
		return err
	}
	fmt.Println("created: Stripe Connect account (demo)")
	return nil
}

func demoProjects() []demoProject {
	return []demoProject{
		{
			title:        "Indie Game Studio",
			description:  "Support an indie game team. Get dev logs, beta access, and exclusive art.",
			targetAmount: 500_000,
			plans: []demoPlan{
				{model.MembershipBronze, 500, "Monthly dev newsletter"},
				{model.MembershipSilver, 1500, "Beta access + Discord role"},
				{model.MembershipGold, 3000, "Producer credit + monthly call"},
			},
		},
		{
			title:        "Music Creator Membership",
			description:  "Join a music creator community with unreleased tracks and live sessions.",
			targetAmount: 300_000,
			plans: []demoPlan{
				{model.MembershipBronze, 300, "Early track previews"},
				{model.MembershipSilver, 900, "Monthly mini live"},
				{model.MembershipGold, 2000, "Request a cover song"},
			},
		},
		{
			title:        "Art & Illustration Club",
			description:  "Support an illustrator and receive sketches, wallpapers, and tutorials.",
			targetAmount: 200_000,
			plans: []demoPlan{
				{model.MembershipBronze, 500, "Wallpaper pack"},
				{model.MembershipSilver, 1200, "Process videos"},
				{model.MembershipGold, 2500, "Custom sketch lottery"},
			},
		},
	}
}

func upsertProject(
	ctx context.Context,
	repo repository.IProjectRepo,
	sellerID uuid.UUID,
	dp demoProject,
	now time.Time,
	reset bool,
) (bool, error) {
	var (
		rows  []model.Project
		total int64
		err   error
		i     int
		p     *model.Project
		plans []model.ProjectPlan
	)

	rows, total, err = repo.ListSeller(ctx, sellerID, 100, 0)
	if err != nil {
		return false, err
	}
	_ = total
	for i = range rows {
		if rows[i].Title == dp.title {
			if !reset {
				fmt.Printf("skip (exists): project %q\n", dp.title)
				return false, nil
			}
			rows[i].Description = dp.description
			rows[i].TargetAmount = dp.targetAmount
			rows[i].Status = model.ProjectStatusActive
			rows[i].UpdatedAt = now
			if err = repo.Update(ctx, &rows[i]); err != nil {
				return false, err
			}
			fmt.Printf("reset: project %q\n", dp.title)
			return false, nil
		}
	}

	p = &model.Project{
		ID:           uuid.New(),
		SellerID:     sellerID,
		Title:        dp.title,
		Description:  dp.description,
		TargetAmount: dp.targetAmount,
		DurationType: model.DurationUnlimited,
		StartDate:    now,
		Status:       model.ProjectStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	plans = make([]model.ProjectPlan, len(dp.plans))
	for i, pl := range dp.plans {
		plans[i] = model.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: pl.membership,
			Price:          pl.price,
			Benefits:       pl.benefits,
			IsActive:       true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	if err = repo.Create(ctx, p, plans, nil); err != nil {
		return false, err
	}
	fmt.Printf("created: project %q\n", dp.title)
	return true, nil
}

func currentTOTP() string {
	var (
		code string
		err  error
	)
	code, err = totp.GenerateCode(devTOTPSecret, time.Now())
	if err != nil {
		return "(generate failed)"
	}
	return code
}
