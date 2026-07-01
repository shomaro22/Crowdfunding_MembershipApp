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

	
	"crowdfund-app/db"
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/usecase"

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
	title         string
	description   string
	targetAmount  int64
	currentAmount int64
	favoriteCount int
	plans         []demoPlan
}

type demoPlan struct {
	membership model.MembershipType
	price      int
	benefits   string
	supporters int
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
		buyerID        uuid.UUID
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

	hash, err = usecase.HashPassword(password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	totpEnc, err = usecase.EncryptSecret([]byte(jwtSecret), devTOTPSecret)
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
		name:  "デモ購入者",
		role:  model.UserRoleBuyer,
	}}
	if team {
		accs = append(accs,
			seedAccount{email: "buyer2@example.com", name: "デモ購入者2", role: model.UserRoleBuyer},
			seedAccount{
				email:   "seller@example.com",
				name:    "デモクリエイター",
				role:    model.UserRoleSeller,
				with2FA: true,
			},
			seedAccount{
				email:     "admin@example.com",
				name:      "デモ管理者",
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
		if acc.role == model.UserRoleBuyer && acc.email == "buyer@example.com" {
			buyerID = u.ID
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
			created, err = upsertProject(ctx, gormDB, projects, sellerID, dp, now, reset)
			if err != nil {
				log.Fatalf("project %q: %v", dp.title, err)
			}
			if created {
				createdProjects++
			} else {
				skippedProjects++
			}
		}
		if err = seedPendingSellerRegs(ctx, gormDB, users, sellerRegs, hash, now, reset); err != nil {
			log.Fatalf("pending kyc: %v", err)
		}
		if buyerID != uuid.Nil {
			if err = seedDemoEngagement(ctx, gormDB, buyerID, reset); err != nil {
				log.Fatalf("demo engagement: %v", err)
			}
		}
		if err = seedSyntheticFavorites(ctx, gormDB, users, hash, now, reset); err != nil {
			log.Fatalf("synthetic favorites: %v", err)
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
		FullName:     "デモクリエイター",
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
			title:         "月額ファンクラブ｜限定イラストと制作講座",
			description:   "イラストレーターの制作過程を限定公開。壁紙配布・ラフスケッチ・月1回のライブ配信付きメンバーシップです。",
			targetAmount:  300_000,
			currentAmount: 450_000,
			favoriteCount: 328,
			plans: []demoPlan{
				{model.MembershipBronze, 500, "限定壁紙＋制作メモ", 120},
				{model.MembershipSilver, 1_500, "ラフ画配布＋Discord参加", 45},
				{model.MembershipGold, 3_000, "月1ライブ＋リクエスト枠", 15},
			},
		},
		{
			title:         "音楽クリエイターの裏側メンバーシップ",
			description:   "未公開デモ音源・制作Vlog・月例ミニライブ。音楽活動の舞台裏をファンと一緒に楽しむコミュニティです。",
			targetAmount:  300_000,
			currentAmount: 180_000,
			favoriteCount: 156,
			plans: []demoPlan{
				{model.MembershipBronze, 300, "未公開音源プレビュー", 80},
				{model.MembershipSilver, 900, "月例ミニライブ参加", 28},
				{model.MembershipGold, 2_000, "カバー曲リクエスト枠", 8},
			},
		},
		{
			title:         "インディーゲーム開発ログ＆先行体験",
			description:   "開発中タイトルの進捗レポート・ベータ版先行プレイ・制作秘話。目標を超える支援で拡張コンテンツも制作中。",
			targetAmount:  500_000,
			currentAmount: 520_000,
			favoriteCount: 412,
			plans: []demoPlan{
				{model.MembershipBronze, 500, "開発ニュースレター", 200},
				{model.MembershipSilver, 1_500, "ベータ版アクセス", 55},
				{model.MembershipGold, 3_000, "クレジット掲載＋月例通話", 12},
			},
		},
		{
			title:         "ライフスタイル創作｜フィットネス＆Vlog",
			description:   "日々のトレーニング・食事・創作Vlogをメンバー限定で配信。健康的なライフスタイルを一緒に楽しむファンクラブです。",
			targetAmount:  200_000,
			currentAmount: 85_000,
			favoriteCount: 89,
			plans: []demoPlan{
				{model.MembershipBronze, 500, "限定Vlog＋レシピ", 42},
				{model.MembershipSilver, 1_200, "週次ライブアーカイブ", 15},
				{model.MembershipGold, 2_500, "パーソナルプラン相談枠", 3},
			},
		},
		{
			title:         "写真・映像クリエイターの会員制ギャラリー",
			description:   "高解像度作品・撮影裏話・機材レビューを会員限定公開。創作活動を支える月額コミュニティです。",
			targetAmount:  250_000,
			currentAmount: 310_000,
			favoriteCount: 245,
			plans: []demoPlan{
				{model.MembershipBronze, 400, "限定ギャラリー閲覧", 95},
				{model.MembershipSilver, 1_000, "撮影メイキング動画", 30},
				{model.MembershipGold, 2_200, "作品リクエスト抽選", 10},
			},
		},
		{
			title:         "Podcast＆トーク番組｜メンバー限定アーカイブ",
			description:   "通常配信に加え、メンバー限定の深掘りトーク・ゲスト回・アフタートークをお届けします。",
			targetAmount:  150_000,
			currentAmount: 42_000,
			favoriteCount: 67,
			plans: []demoPlan{
				{model.MembershipBronze, 300, "限定エピソード", 35},
				{model.MembershipSilver, 800, "アフタートーク付き", 12},
				{model.MembershipGold, 1_800, "リスナー参加企画", 4},
			},
		},
		{
			title:         "ハンドメイド作家の制作室メンバーシップ",
			description:   "作品制作の様子・素材選び・ショップ先行販売。ものづくりの温かさを届けるファンクラブです。",
			targetAmount:  180_000,
			currentAmount: 195_000,
			favoriteCount: 178,
			plans: []demoPlan{
				{model.MembershipBronze, 600, "制作過程ストーリー", 48},
				{model.MembershipSilver, 1_400, "先行販売アクセス", 18},
				{model.MembershipGold, 2_800, "オーダーメイド抽選", 6},
			},
		},
	}
}

func demoProjectSupporterTotal(dp demoProject) int {
	var total int
	for _, pl := range dp.plans {
		total += pl.supporters
	}
	return total
}

func upsertProject(
	ctx context.Context,
	gormDB *gorm.DB,
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
		j     int
		p     *model.Project
		plans []model.ProjectPlan
		totalSupporters int
	)

	totalSupporters = demoProjectSupporterTotal(dp)
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
			rows[i].CurrentAmount = dp.currentAmount
			rows[i].SupporterCount = totalSupporters
			rows[i].Status = model.ProjectStatusActive
			rows[i].UpdatedAt = now
			if err = repo.Update(ctx, &rows[i]); err != nil {
				return false, err
			}
			if err = syncDemoPlanCounts(ctx, gormDB, rows[i].ID, dp.plans, now); err != nil {
				return false, err
			}
			fmt.Printf("reset: project %q\n", dp.title)
			return false, nil
		}
	}

	p = &model.Project{
		ID:             uuid.New(),
		SellerID:       sellerID,
		Title:          dp.title,
		Description:    dp.description,
		TargetAmount:   dp.targetAmount,
		CurrentAmount:  dp.currentAmount,
		SupporterCount: totalSupporters,
		DurationType:   model.DurationUnlimited,
		StartDate:      now,
		Status:         model.ProjectStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	plans = make([]model.ProjectPlan, len(dp.plans))
	for i, pl := range dp.plans {
		plans[i] = model.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: pl.membership,
			Price:          pl.price,
			Benefits:       pl.benefits,
			IsActive:       true,
			SupporterCount: pl.supporters,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	_ = j
	if err = repo.Create(ctx, p, plans, nil); err != nil {
		return false, err
	}
	fmt.Printf("created: project %q\n", dp.title)
	return true, nil
}

func syncDemoPlanCounts(ctx context.Context, gormDB *gorm.DB, projectID uuid.UUID, demoPlans []demoPlan, now time.Time) error {
	var (
		plans []model.ProjectPlan
		err   error
		i     int
		j     int
	)
	err = gormDB.WithContext(ctx).Where("project_id = ?", projectID).Find(&plans).Error
	if err != nil {
		return err
	}
	for i = range plans {
		for j = range demoPlans {
			if plans[i].MembershipType == demoPlans[j].membership {
				plans[i].SupporterCount = demoPlans[j].supporters
				plans[i].UpdatedAt = now
				if err = gormDB.WithContext(ctx).Save(&plans[i]).Error; err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func seedPendingSellerRegs(
	ctx context.Context,
	gormDB *gorm.DB,
	users repository.IUserRepo,
	regs repository.ISellerRegRepo,
	hash string,
	now time.Time,
	reset bool,
) error {
	type pendingSeller struct {
		email string
		name  string
	}
	var (
		candidates = []pendingSeller{
			{email: "kyc-pending1@example.com", name: "審査待ちクリエイター1"},
			{email: "kyc-pending2@example.com", name: "審査待ちクリエイター2"},
			{email: "kyc-pending3@example.com", name: "審査待ちクリエイター3"},
		}
		acc  seedAccount
		u    model.User
		front = "seed/kyc/front.jpg"
		back  = "seed/kyc/back.jpg"
		reg  model.SellerReg
		err  error
	)
	for _, c := range candidates {
		acc = seedAccount{email: c.email, name: c.name, role: model.UserRoleSeller}
		u, _, err = upsertUser(ctx, users, acc, hash, "", now, reset)
		if err != nil {
			return err
		}
		u.SellerStatus = model.SellerStatusPending
		if err = users.Update(ctx, &u); err != nil {
			return err
		}
		_, err = regs.GetUserID(ctx, u.ID)
		if err == nil {
			continue
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		reg = model.SellerReg{
			ID:           uuid.New(),
			UserID:       u.ID,
			FullName:     c.name,
			BirthDate:    time.Date(1992, 6, 1, 0, 0, 0, 0, time.UTC),
			Address:      "東京都渋谷区",
			PhotoPresent: true,
			IDFrontKey:   &front,
			IDBackKey:    &back,
			Status:       model.RegPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err = regs.Create(ctx, &reg); err != nil {
			return err
		}
		fmt.Printf("created: pending KYC %s\n", c.email)
	}
	return nil
}

func seedDemoEngagement(ctx context.Context, gormDB *gorm.DB, buyerID uuid.UUID, reset bool) error {
	var (
		projects []model.Project
		err      error
		i        int
		marker   = "seed-demo-engagement-v1"
		count    int64
	)
	if !reset {
		err = gormDB.WithContext(ctx).Model(&model.Purchase{}).
			Where("buyer_id = ? AND idempotency_key = ?", buyerID, marker).
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	} else {
		_ = gormDB.WithContext(ctx).Where("buyer_id = ?", buyerID).Delete(&model.Favorite{}).Error
		_ = gormDB.WithContext(ctx).Where("user_id = ?", buyerID).Delete(&model.ProjectSupporter{}).Error
		_ = gormDB.WithContext(ctx).Where("buyer_id = ?", buyerID).Delete(&model.Purchase{}).Error
	}
	err = gormDB.WithContext(ctx).
		Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").
		Limit(3).
		Find(&projects).Error
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	periodEnd := now.AddDate(0, 1, 0)
	for i = range projects {
		var (
			plans []model.ProjectPlan
			plan  model.ProjectPlan
			sup   model.ProjectSupporter
			pur   model.Purchase
			key   = marker + "-" + projects[i].ID.String()
			fav   model.Favorite
		)
		err = gormDB.WithContext(ctx).Where("project_id = ?", projects[i].ID).
			Order("membership_type ASC").Find(&plans).Error
		if err != nil || len(plans) == 0 {
			continue
		}
		plan = plans[0]
		sup = model.ProjectSupporter{
			ID:                 uuid.New(),
			ProjectID:          projects[i].ID,
			UserID:             buyerID,
			PlanID:             plan.ID,
			Status:             model.SupporterStatusActive,
			BillingMode:        model.BillingModeAuto,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   periodEnd,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err = gormDB.WithContext(ctx).Create(&sup).Error; err != nil {
			return err
		}
		pur = model.Purchase{
			ID:             uuid.New(),
			ProjectID:      projects[i].ID,
			SupporterID:    &sup.ID,
			BuyerID:        buyerID,
			Quantity:       1,
			TotalPrice:     plan.Price,
			Status:         model.PurchaseStatusPaid,
			PaymentMethod:  model.PaymentMethodCard,
			PeriodStart:    &now,
			PeriodEnd:      &periodEnd,
			IdempotencyKey: &key,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err = gormDB.WithContext(ctx).Create(&pur).Error; err != nil {
			return err
		}
		fav = model.Favorite{
			ID:        uuid.New(),
			UserID:    buyerID,
			ProjectID: projects[i].ID,
			CreatedAt: now,
		}
		_ = gormDB.WithContext(ctx).Create(&fav).Error
	}
	fmt.Println("created: demo purchases, supporters, favorites for buyer")
	return nil
}

func seedSyntheticFavorites(
	ctx context.Context,
	gormDB *gorm.DB,
	users repository.IUserRepo,
	hash string,
	now time.Time,
	reset bool,
) error {
	var (
		buyerIDs []uuid.UUID
		projects []model.Project
		err      error
		i        int
		j        int
		u        model.User
	)
	for i = 0; i < 40; i++ {
		email := fmt.Sprintf("fav-bot-%02d@example.com", i+1)
		u, _, err = upsertUser(ctx, users, seedAccount{
			email: email,
			name:  fmt.Sprintf("いいねBot %d", i+1),
			role:  model.UserRoleBuyer,
		}, hash, "", now, reset)
		if err != nil {
			return err
		}
		buyerIDs = append(buyerIDs, u.ID)
	}
	if reset {
		for _, id := range buyerIDs {
			_ = gormDB.WithContext(ctx).Where("user_id = ?", id).Delete(&model.Favorite{}).Error
		}
	}
	err = gormDB.WithContext(ctx).
		Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").
		Find(&projects).Error
	if err != nil {
		return err
	}
	for i, p := range projects {
		target := demoProjects()[i%len(demoProjects())].favoriteCount
		if target <= 0 {
			target = 10
		}
		if target > len(buyerIDs) {
			target = len(buyerIDs)
		}
		for j = 0; j < target; j++ {
			fav := model.Favorite{
				ID:        uuid.New(),
				UserID:    buyerIDs[j%len(buyerIDs)],
				ProjectID: p.ID,
				CreatedAt: now,
			}
			err = gormDB.WithContext(ctx).Create(&fav).Error
			if err != nil {
				if repository.IsFavoriteDuplicate(err) {
					continue
				}
				return err
			}
		}
	}
	fmt.Println("created: synthetic favorite counts")
	return nil
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
