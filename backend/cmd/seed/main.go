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
	"path/filepath"
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

// minimal 1x1 JPEG
var seedKYCJPEG = []byte{
	0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01,
	0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xff, 0xdb, 0x00, 0x43,
	0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
	0x09, 0x08, 0x0a, 0x0c, 0x14, 0x0d, 0x0c, 0x0b, 0x0b, 0x0c, 0x19, 0x12,
	0x13, 0x0f, 0x14, 0x1d, 0x1a, 0x1f, 0x1e, 0x1d, 0x1a, 0x1c, 0x1c, 0x20,
	0x24, 0x2e, 0x27, 0x20, 0x22, 0x2c, 0x23, 0x1c, 0x1c, 0x28, 0x37, 0x29,
	0x2c, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1f, 0x27, 0x39, 0x3d, 0x38, 0x32,
	0x3c, 0x2e, 0x33, 0x34, 0x32, 0xff, 0xc0, 0x00, 0x0b, 0x08, 0x00, 0x01,
	0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xff, 0xc4, 0x00, 0x1f, 0x00, 0x00,
	0x01, 0x05, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0xff, 0xc4, 0x00, 0xb5, 0x10, 0x00, 0x02, 0x01, 0x03,
	0x03, 0x02, 0x04, 0x03, 0x05, 0x05, 0x04, 0x04, 0x00, 0x00, 0x01, 0x7d,
	0x01, 0x02, 0x03, 0x00, 0x04, 0x11, 0x05, 0x12, 0x21, 0x31, 0x41, 0x06,
	0x13, 0x51, 0x61, 0x07, 0x22, 0x71, 0x14, 0x32, 0x81, 0x91, 0xa1, 0x08,
	0x23, 0x42, 0xb1, 0xc1, 0x15, 0x52, 0xd1, 0xf0, 0x24, 0x33, 0x62, 0x72,
	0x82, 0x09, 0x0a, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x25, 0x26, 0x27, 0x28,
	0x29, 0x2a, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x43, 0x44, 0x45,
	0x46, 0x47, 0x48, 0x49, 0x4a, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59,
	0x5a, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a, 0x73, 0x74, 0x75,
	0x76, 0x77, 0x78, 0x79, 0x7a, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89,
	0x8a, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9a, 0xa2, 0xa3,
	0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6,
	0xb7, 0xb8, 0xb9, 0xba, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8, 0xc9,
	0xca, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xe1, 0xe2,
	0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xf1, 0xf2, 0xf3, 0xf4,
	0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xfa, 0xff, 0xda, 0x00, 0x08, 0x01, 0x01,
	0x00, 0x00, 0x3f, 0x00, 0xfb, 0xd5, 0xdb, 0x20, 0xa8, 0xf1, 0x45, 0x00,
	0x28, 0xa2, 0x8a, 0x00, 0xff, 0xd9,
}

var demoSellerEmails = []string{
	"seller@example.com",
	"seller2@example.com",
	"seller3@example.com",
	"seller4@example.com",
}

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
	email         string
	name          string
	role          model.UserRole
	adminRole     *model.AdminRole
	with2FA       bool
	accountStatus *model.AccountStatus
	sellerStatus  *model.SellerStatus
}

type demoProject struct {
	title         string
	description   string
	targetAmount  int64
	currentAmount int64
	favoriteCount int
	imageURL      string
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
		privateRoot    string
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
		sellerIDs      []uuid.UUID
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
				email:         "buyer-pending@example.com",
				name:          "審査待ち購入者",
				role:          model.UserRoleBuyer,
				accountStatus: accountStatusPtr(model.AccountStatusPending),
			},
			seedAccount{
				email:   "seller@example.com",
				name:    "デモクリエイター",
				role:    model.UserRoleSeller,
				with2FA: true,
			},
			seedAccount{
				email:   "seller2@example.com",
				name:    "デモクリエイター2",
				role:    model.UserRoleSeller,
				with2FA: true,
			},
			seedAccount{
				email:   "seller3@example.com",
				name:    "デモクリエイター3",
				role:    model.UserRoleSeller,
				with2FA: true,
			},
			seedAccount{
				email:   "seller4@example.com",
				name:    "デモクリエイター4",
				role:    model.UserRoleSeller,
				with2FA: true,
			},
			seedAccount{
				email:     "admin@example.com",
				name:      "デモ管理者",
				role:      model.UserRoleAdmin,
				adminRole: adminRolePtr(model.AdminRoleSuperAdmin),
				with2FA:   true,
			},
		)
	}

	privateRoot = strings.TrimSpace(os.Getenv("PRIVATE_ROOT"))
	if privateRoot == "" {
		privateRoot = "data/private"
	}
	if team && reset {
		if err = purgeTeamSeedData(ctx, gormDB, users); err != nil {
			log.Fatalf("purge: %v", err)
		}
	}
	if err = ensureKYCSeedFiles(privateRoot); err != nil {
		log.Fatalf("kyc files: %v", err)
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
			sellerIDs = append(sellerIDs, u.ID)
		}
		if acc.role == model.UserRoleBuyer && acc.email == "buyer@example.com" {
			buyerID = u.ID
		}
		if acc.role == model.UserRoleAdmin {
			adminID = u.ID
		}
	}

	if team && len(sellerIDs) > 0 {
		for i, sellerID := range sellerIDs {
			if err = upsertSellerReg(ctx, sellerRegs, sellerID, adminID, demoSellerEmails[i], now); err != nil {
				log.Fatalf("seller reg: %v", err)
			}
			if err = upsertConnect(ctx, connect, sellerID, fmt.Sprintf("acct_seed_demo_seller_%d", i+1), now); err != nil {
				log.Fatalf("connect: %v", err)
			}
		}
		allDemo := append(demoProjects(), extendedDemoProjects()...)
		for i, dp := range allDemo {
			sellerID := sellerIDs[i%len(sellerIDs)]
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
		if err = seedReviewProjects(ctx, gormDB, projects, sellerIDs[0], adminID, now, reset); err != nil {
			log.Fatalf("review projects: %v", err)
		}
		for _, sellerID := range sellerIDs {
			if err = repairCorruptedDemoText(ctx, gormDB, sellerID, allDemo, now); err != nil {
				log.Fatalf("repair demo text: %v", err)
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
		if buyerID != uuid.Nil && len(sellerIDs) > 0 {
			if err = seedDemoMessages(ctx, gormDB, buyerID, sellerIDs[0], reset); err != nil {
				log.Fatalf("demo messages: %v", err)
			}
		}
		if err = seedSyntheticSupporters(ctx, gormDB, users, hash, now, reset); err != nil {
			log.Fatalf("synthetic supporters: %v", err)
		}
		if err = seedSyntheticFavorites(ctx, gormDB, users, hash, now, reset); err != nil {
			log.Fatalf("synthetic favorites: %v", err)
		}
		if err = seedDemoReports(ctx, gormDB, buyerID, adminID, users, hash, now, reset); err != nil {
			log.Fatalf("demo reports: %v", err)
		}
		if err = seedAdminPurchases(ctx, gormDB, buyerID, reset); err != nil {
			log.Fatalf("admin purchases: %v", err)
		}
	}

	allDemoCatalog := append(demoProjects(), extendedDemoProjects()...)
	if err = repairAllCorruptedDemoText(ctx, gormDB, allDemoCatalog, now); err != nil {
		log.Fatalf("repair all demo text: %v", err)
	}

	fmt.Println()
	fmt.Println("シード完了:")
	fmt.Printf("  ユーザー: 作成/更新=%d スキップ=%d\n", createdUsers+updatedUsers, skippedUsers)
	if team {
		fmt.Printf("  プロジェクト: 作成=%d スキップ=%d\n", createdProjects, skippedProjects)
	}
	fmt.Printf("  共通パスワード: %s\n", password)
	if team {
		fmt.Printf("  2FA（出品者/管理者）: TOTP secret=%s  現在のコード=%s\n", devTOTPSecret, currentTOTP())
		fmt.Println("  デモアカウント:")
		fmt.Println("    buyer@example.com        （購入者）")
		fmt.Println("    buyer2@example.com       （購入者）")
		fmt.Println("    buyer-pending@example.com（購入者・審査待ち）")
		fmt.Println("    seller@example.com       （出品者・デモクリエイター）")
		fmt.Println("    seller2@example.com      （出品者・デモクリエイター2）")
		fmt.Println("    seller3@example.com      （出品者・デモクリエイター3）")
		fmt.Println("    seller4@example.com      （出品者・デモクリエイター4）")
		fmt.Println("    admin@example.com        （管理者・SUPER_ADMIN）")
	}
}

func adminRolePtr(r model.AdminRole) *model.AdminRole {
	return &r
}

func accountStatusPtr(s model.AccountStatus) *model.AccountStatus {
	return &s
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
	if acc.accountStatus != nil {
		u.AccountStatus = *acc.accountStatus
	}
	if acc.sellerStatus != nil {
		u.SellerStatus = *acc.sellerStatus
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

func upsertSellerReg(ctx context.Context, regs repository.ISellerRegRepo, sellerID, adminID uuid.UUID, fullName string, now time.Time) error {
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
		FullName:     fullName,
		BirthDate:    time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC),
		Address:      "東京都",
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
	fmt.Println("作成: 出品者登録（承認済）")
	return nil
}

func upsertConnect(ctx context.Context, connect repository.IConnectRepo, sellerID uuid.UUID, stripeAccountID string, now time.Time) error {
	var (
		row model.ConnectAccount
		err error
	)

	row, err = connect.GetUserID(ctx, sellerID)
	if err == nil {
		row.ChargesEnabled = true
		row.PayoutsEnabled = true
		row.DetailsSubmitted = true
		row.UpdatedAt = now
		if err = connect.Update(ctx, &row); err != nil {
			return err
		}
		fmt.Println("更新: Stripe Connect（デモ）")
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	row = model.ConnectAccount{
		ID:               uuid.New(),
		UserID:           sellerID,
		StripeAccountID:  stripeAccountID,
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err = connect.Create(ctx, &row); err != nil {
		return err
	}
	fmt.Println("作成: Stripe Connect（デモ）")
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
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-1/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 500, "限定壁紙＋制作メモ", 12},
				{model.MembershipSilver, 1_500, "ラフ画配布＋Discord参加", 5},
				{model.MembershipGold, 3_000, "月1ライブ＋リクエスト枠", 2},
			},
		},
		{
			title:         "音楽クリエイターの裏側メンバーシップ",
			description:   "未公開デモ音源・制作Vlog・月例ミニライブ。音楽活動の舞台裏をファンと一緒に楽しむコミュニティです。",
			targetAmount:  300_000,
			currentAmount: 180_000,
			favoriteCount: 156,
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-2/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 300, "未公開音源プレビュー", 8},
				{model.MembershipSilver, 900, "月例ミニライブ参加", 3},
				{model.MembershipGold, 2_000, "カバー曲リクエスト枠", 1},
			},
		},
		{
			title:         "インディーゲーム開発ログ＆先行体験",
			description:   "開発中タイトルの進捗レポート・ベータ版先行プレイ・制作秘話。目標を超える支援で拡張コンテンツも制作中。",
			targetAmount:  500_000,
			currentAmount: 520_000,
			favoriteCount: 412,
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-3/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 500, "開発ニュースレター", 20},
				{model.MembershipSilver, 1_500, "ベータ版アクセス", 8},
				{model.MembershipGold, 3_000, "クレジット掲載＋月例通話", 3},
			},
		},
		{
			title:         "ライフスタイル創作｜フィットネス＆Vlog",
			description:   "日々のトレーニング・食事・創作Vlogをメンバー限定で配信。健康的なライフスタイルを一緒に楽しむファンクラブです。",
			targetAmount:  200_000,
			currentAmount: 85_000,
			favoriteCount: 89,
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-4/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 500, "限定Vlog＋レシピ", 5},
				{model.MembershipSilver, 1_200, "週次ライブアーカイブ", 2},
				{model.MembershipGold, 2_500, "パーソナルプラン相談枠", 1},
			},
		},
		{
			title:         "写真・映像クリエイターの会員制ギャラリー",
			description:   "高解像度作品・撮影裏話・機材レビューを会員限定公開。創作活動を支える月額コミュニティです。",
			targetAmount:  250_000,
			currentAmount: 310_000,
			favoriteCount: 245,
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-5/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 400, "限定ギャラリー閲覧", 10},
				{model.MembershipSilver, 1_000, "撮影メイキング動画", 4},
				{model.MembershipGold, 2_200, "作品リクエスト抽選", 2},
			},
		},
		{
			title:         "ポッドキャスト＆トーク番組｜メンバー限定アーカイブ",
			description:   "通常配信に加え、メンバー限定の深掘りトーク・ゲスト回・アフタートークをお届けします。",
			targetAmount:  150_000,
			currentAmount: 42_000,
			favoriteCount: 67,
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-6/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 300, "限定エピソード", 4},
				{model.MembershipSilver, 800, "アフタートーク付き", 2},
				{model.MembershipGold, 1_800, "リスナー参加企画", 1},
			},
		},
		{
			title:         "ハンドメイド作家の制作室メンバーシップ",
			description:   "作品制作の様子・素材選び・ショップ先行販売。ものづくりの温かさを届けるファンクラブです。",
			targetAmount:  180_000,
			currentAmount: 195_000,
			favoriteCount: 178,
			imageURL:      "https://picsum.photos/seed/crowdfund-demo-7/800/600",
			plans: []demoPlan{
				{model.MembershipBronze, 600, "制作過程ストーリー", 7},
				{model.MembershipSilver, 1_400, "先行販売アクセス", 3},
				{model.MembershipGold, 2_800, "オーダーメイド抽選", 1},
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
		images []model.ProjectImage
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
				if projectTextCorrupted(rows[i]) {
					rows[i].Description = dp.description
					rows[i].UpdatedAt = now
					if err = repo.Update(ctx, &rows[i]); err != nil {
						return false, err
					}
					if err = syncDemoPlanCounts(ctx, gormDB, rows[i].ID, dp.plans, now); err != nil {
						return false, err
					}
					fmt.Printf("repaired: project %q\n", dp.title)
				} else {
					fmt.Printf("skip (exists): project %q\n", dp.title)
				}
				return false, nil
			}
			rows[i].Description = dp.description
			rows[i].TargetAmount = dp.targetAmount
			rows[i].CurrentAmount = dp.currentAmount
			rows[i].SupporterCount = totalSupporters
			rows[i].Status = model.ProjectStatusActive
			rows[i].UpdatedAt = now
			if dp.imageURL != "" {
				img := dp.imageURL
				rows[i].MainImageURL = &img
			}
			if err = repo.Update(ctx, &rows[i]); err != nil {
				return false, err
			}
			if err = syncDemoPlanCounts(ctx, gormDB, rows[i].ID, dp.plans, now); err != nil {
				return false, err
			}
			if err = syncDemoImages(ctx, repo, rows[i].ID, dp.imageURL, now); err != nil {
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
	if dp.imageURL != "" {
		img := dp.imageURL
		p.MainImageURL = &img
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
	if dp.imageURL != "" {
		images = []model.ProjectImage{{
			ID:        uuid.New(),
			ImageURL:  dp.imageURL,
			IsMain:    true,
			SortOrder: 0,
			CreatedAt: now,
		}}
	}
	if err = repo.Create(ctx, p, plans, images); err != nil {
		return false, err
	}
	fmt.Printf("created: project %q\n", dp.title)
	return true, nil
}

func syncDemoImages(ctx context.Context, repo repository.IProjectRepo, projectID uuid.UUID, imageURL string, now time.Time) error {
	var (
		images []model.ProjectImage
		project model.Project
		err    error
	)
	if imageURL == "" {
		return nil
	}
	images = []model.ProjectImage{{
		ID:        uuid.New(),
		ImageURL:  imageURL,
		IsMain:    true,
		SortOrder: 0,
		CreatedAt: now,
	}}
	err = repo.ReplaceImages(ctx, projectID, images)
	if err != nil {
		return err
	}
	project, _, _, err = repo.GetID(ctx, projectID)
	if err != nil {
		return err
	}
	img := imageURL
	project.MainImageURL = &img
	project.UpdatedAt = now
	return repo.Update(ctx, &project)
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
				plans[i].Benefits = demoPlans[j].benefits
				plans[i].Price = demoPlans[j].price
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

func projectTextCorrupted(p model.Project) bool {
	if strings.Contains(p.Description, "%!(EXTRA") {
		return true
	}
	return false
}

func planBenefitsCorrupted(plans []model.ProjectPlan) bool {
	var i int
	for i = range plans {
		if strings.Contains(plans[i].Benefits, "%!(EXTRA") {
			return true
		}
	}
	return false
}

func repairCorruptedDemoText(ctx context.Context, gormDB *gorm.DB, sellerID uuid.UUID, allProjects []demoProject, now time.Time) error {
	var (
		rows []model.Project
		err  error
		i    int
		j    int
		dp   demoProject
		found bool
	)
	err = gormDB.WithContext(ctx).Where("seller_id = ?", sellerID).Find(&rows).Error
	if err != nil {
		return err
	}
	for i = range rows {
		if !projectTextCorrupted(rows[i]) {
			var plans []model.ProjectPlan
			err = gormDB.WithContext(ctx).Where("project_id = ?", rows[i].ID).Find(&plans).Error
			if err != nil {
				return err
			}
			if !planBenefitsCorrupted(plans) {
				continue
			}
		}
		found = false
		for j = range allProjects {
			if allProjects[j].title == rows[i].Title {
				dp = allProjects[j]
				found = true
				break
			}
		}
		if !found {
			continue
		}
		rows[i].Description = dp.description
		rows[i].UpdatedAt = now
		if err = gormDB.WithContext(ctx).Save(&rows[i]).Error; err != nil {
			return err
		}
		if err = syncDemoPlanCounts(ctx, gormDB, rows[i].ID, dp.plans, now); err != nil {
			return err
		}
	}
	return nil
}

func repairAllCorruptedDemoText(ctx context.Context, gormDB *gorm.DB, allProjects []demoProject, now time.Time) error {
	var (
		rows  []model.Project
		err   error
		i     int
		j     int
		dp    demoProject
		found bool
		plans []model.ProjectPlan
	)
	err = gormDB.WithContext(ctx).Find(&rows).Error
	if err != nil {
		return err
	}
	for i = range rows {
		err = gormDB.WithContext(ctx).Where("project_id = ?", rows[i].ID).Find(&plans).Error
		if err != nil {
			return err
		}
		if !projectTextCorrupted(rows[i]) && !planBenefitsCorrupted(plans) {
			continue
		}
		found = false
		for j = range allProjects {
			if allProjects[j].title == rows[i].Title {
				dp = allProjects[j]
				found = true
				break
			}
		}
		if !found {
			continue
		}
		rows[i].Description = dp.description
		rows[i].UpdatedAt = now
		if err = gormDB.WithContext(ctx).Save(&rows[i]).Error; err != nil {
			return err
		}
		if err = syncDemoPlanCounts(ctx, gormDB, rows[i].ID, dp.plans, now); err != nil {
			return err
		}
		fmt.Printf("repaired: project %q\n", rows[i].Title)
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
		_ = gormDB.WithContext(ctx).Where("sender_id = ? OR receiver_id = ?", buyerID, buyerID).Delete(&model.Message{}).Error
	}
	err = gormDB.WithContext(ctx).
		Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").
		Limit(1).
		Find(&projects).Error
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i = range projects {
		var (
			plans         []model.ProjectPlan
			plan          model.ProjectPlan
			planIdx       int
			sup           model.ProjectSupporter
			pur           model.Purchase
			key           = marker + "-" + projects[i].ID.String()
			fav           model.Favorite
			supStatus     = model.SupporterStatusActive
			periodStart   time.Time
			periodEnd     time.Time
		)
		err = gormDB.WithContext(ctx).Where("project_id = ?", projects[i].ID).
			Order("membership_type ASC").Find(&plans).Error
		if err != nil || len(plans) == 0 {
			continue
		}
		planIdx = 0
		plan = plans[planIdx]
		periodStart = now
		periodEnd = now.AddDate(0, 1, 0)
		sup = model.ProjectSupporter{
			ID:                 uuid.New(),
			ProjectID:          projects[i].ID,
			UserID:             buyerID,
			PlanID:             plan.ID,
			Status:             supStatus,
			BillingMode:        model.BillingModeAuto,
			CurrentPeriodStart: periodStart,
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
			PeriodStart:    &periodStart,
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
	fmt.Println("作成: デモ購入履歴・お気に入り（1件のみ応援中・他プロジェクトはチェックアウト可能）")
	return nil
}

func seedDemoMessages(ctx context.Context, gormDB *gorm.DB, buyerID, sellerID uuid.UUID, reset bool) error {
	var (
		projects []model.Project
		err      error
		count    int64
	)
	if !reset {
		err = gormDB.WithContext(ctx).Model(&model.Message{}).
			Where(
				"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
				buyerID, sellerID, sellerID, buyerID,
			).
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	} else {
		_ = gormDB.WithContext(ctx).
			Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)", buyerID, sellerID, sellerID, buyerID).
			Delete(&model.Message{}).Error
	}
	err = gormDB.WithContext(ctx).
		Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").
		Limit(1).
		Find(&projects).Error
	if err != nil || len(projects) == 0 {
		return err
	}
	now := time.Now().UTC()
	projectID := projects[0].ID
	contents := []struct {
		senderID   uuid.UUID
		receiverID uuid.UUID
		content    string
	}{
		{buyerID, sellerID, "こんにちは！応援しています。"},
		{sellerID, buyerID, "ありがとうございます！限定コンテンツをお楽しみください。"},
		{buyerID, sellerID, "楽しみにしています！"},
	}
	for i, c := range contents {
		msg := model.Message{
			ID:         uuid.New(),
			ProjectID:  projectID,
			SenderID:   c.senderID,
			ReceiverID: c.receiverID,
			Content:    c.content,
			IsRead:     i < 2,
			CreatedAt:  now.Add(time.Duration(i) * time.Minute),
		}
		if err = gormDB.WithContext(ctx).Create(&msg).Error; err != nil {
			return err
		}
	}
	fmt.Println("作成: デモメッセージ（購入者↔出品者）")
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
	for i = 0; i < 250; i++ {
		email := fmt.Sprintf("fav-bot-%03d@example.com", i+1)
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
		start := (i * 37) % len(buyerIDs)
		for j = 0; j < target; j++ {
			fav := model.Favorite{
				ID:        uuid.New(),
				UserID:    buyerIDs[(start+j)%len(buyerIDs)],
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

func ensureKYCSeedFiles(privateRoot string) error {
	var (
		dir   string
		front string
		back  string
		err   error
	)
	dir = filepath.Join(privateRoot, "seed", "kyc")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	front = filepath.Join(dir, "front.jpg")
	back = filepath.Join(dir, "back.jpg")
	if _, err = os.Stat(front); os.IsNotExist(err) {
		if err = os.WriteFile(front, seedKYCJPEG, 0o600); err != nil {
			return err
		}
	}
	if _, err = os.Stat(back); os.IsNotExist(err) {
		if err = os.WriteFile(back, seedKYCJPEG, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func purgeTeamSeedData(ctx context.Context, gormDB *gorm.DB, users repository.IUserRepo) error {
	var (
		demoEmails []string
		userIDs    []uuid.UUID
		u          model.User
		err        error
		i          int
	)
	// docker compose dev: 過去の integration / 手動テストデータを含め全コンテンツを消す
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.Message{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.Report{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.Favorite{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.Purchase{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.ProjectSupporter{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.ProjectImage{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.ProjectPlan{}).Error
	_ = gormDB.WithContext(ctx).Where("1 = 1").Delete(&model.Project{}).Error

	demoEmails = []string{
		"buyer@example.com", "buyer2@example.com", "buyer-pending@example.com",
		"admin@example.com",
	}
	demoEmails = append(demoEmails, demoSellerEmails...)
	for i = 0; i < 250; i++ {
		demoEmails = append(demoEmails, fmt.Sprintf("fav-bot-%03d@example.com", i+1))
	}
	for i = 0; i < 120; i++ {
		demoEmails = append(demoEmails, fmt.Sprintf("sup-bot-%03d@example.com", i+1))
	}
	demoEmails = append(demoEmails,
		"kyc-pending1@example.com", "kyc-pending2@example.com", "kyc-pending3@example.com",
	)
	for _, email := range demoEmails {
		u, err = users.GetEmail(ctx, email)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		userIDs = append(userIDs, u.ID)
	}
	if len(userIDs) > 0 {
		_ = gormDB.WithContext(ctx).Where("user_id IN ?", userIDs).Delete(&model.SellerReg{}).Error
		_ = gormDB.WithContext(ctx).Where("user_id IN ?", userIDs).Delete(&model.ConnectAccount{}).Error
	}
	fmt.Println("purge: 全プロジェクト・エンゲージメントデータを削除しました")
	return nil
}

func seedReviewProjects(
	ctx context.Context,
	gormDB *gorm.DB,
	repo repository.IProjectRepo,
	sellerID, adminID uuid.UUID,
	now time.Time,
	reset bool,
) error {
	type reviewProject struct {
		title  string
		status model.ProjectStatus
	}
	var (
		items = []reviewProject{
			{title: "【審査中】新規ファンクラブ開設のお知らせ", status: model.ProjectStatusPending},
			{title: "【審査中】写真集メンバーシップ", status: model.ProjectStatusPending},
			{title: "【編集申請】タイトル変更の審査", status: model.ProjectStatusPending},
			{title: "【削除申請】終了予定プロジェクト", status: model.ProjectStatusDeleteRequested},
			{title: "【棄却】規約違反コンテンツのプロジェクト", status: model.ProjectStatusCancelled},
		}
		dp    demoProject
		err   error
		i     int
	)
	if !reset {
		var count int64
		err = gormDB.WithContext(ctx).Model(&model.Project{}).
			Where("seller_id = ? AND title LIKE ?", sellerID, "【審査中】%").
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	for i = range items {
		dp = demoProject{
			title:         items[i].title,
			description:   "管理者審査用のデモプロジェクトです。",
			targetAmount:  100_000,
			currentAmount: 0,
			favoriteCount: 0,
			plans: []demoPlan{
				{model.MembershipBronze, 500, "基本特典", 0},
				{model.MembershipSilver, 1_000, "中級特典", 0},
				{model.MembershipGold, 2_000, "上級特典", 0},
			},
		}
		_, err = upsertReviewProject(ctx, gormDB, repo, sellerID, dp, items[i].status, now)
		if err != nil {
			return err
		}
	}
	_ = adminID
	fmt.Println("created: 審査用プロジェクト（審査中・棄却）")
	return nil
}

func upsertReviewProject(
	ctx context.Context,
	gormDB *gorm.DB,
	repo repository.IProjectRepo,
	sellerID uuid.UUID,
	dp demoProject,
	status model.ProjectStatus,
	now time.Time,
) (bool, error) {
	var (
		rows []model.Project
		err  error
		i    int
		j    int
		p    *model.Project
	)
	rows, _, err = repo.ListSeller(ctx, sellerID, 100, 0)
	if err != nil {
		return false, err
	}
	for i = range rows {
		if rows[i].Title == dp.title {
			rows[i].Status = status
			rows[i].DeleteRequested = status == model.ProjectStatusDeleteRequested
			rows[i].UpdatedAt = now
			if err = repo.Update(ctx, &rows[i]); err != nil {
				return false, err
			}
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
		SupporterCount: 0,
		DurationType:   model.DurationUnlimited,
		StartDate:      now,
		Status:         status,
		DeleteRequested: status == model.ProjectStatusDeleteRequested,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	plans := make([]model.ProjectPlan, len(dp.plans))
	for i, pl := range dp.plans {
		plans[i] = model.ProjectPlan{
			ID:             uuid.New(),
			MembershipType: pl.membership,
			Price:          pl.price,
			Benefits:       pl.benefits,
			IsActive:       true,
			SupporterCount: 0,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	_ = j
	if err = repo.Create(ctx, p, plans, nil); err != nil {
		return false, err
	}
	return true, nil
}

func seedSyntheticSupporters(
	ctx context.Context,
	gormDB *gorm.DB,
	users repository.IUserRepo,
	hash string,
	now time.Time,
	reset bool,
) error {
	var (
		botIDs   []uuid.UUID
		projects []model.Project
		err      error
		i        int
		k        int
		u        model.User
		marker   = "seed-synthetic-supporters-v1"
		j        int
	)
	if !reset {
		var count int64
		err = gormDB.WithContext(ctx).Model(&model.Purchase{}).
			Where("idempotency_key LIKE ?", marker+"%").
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	for i = 0; i < 120; i++ {
		email := fmt.Sprintf("sup-bot-%03d@example.com", i+1)
		u, _, err = upsertUser(ctx, users, seedAccount{
			email: email,
			name:  fmt.Sprintf("支援Bot %d", i+1),
			role:  model.UserRoleBuyer,
		}, hash, "", now, reset)
		if err != nil {
			return err
		}
		botIDs = append(botIDs, u.ID)
	}
	err = gormDB.WithContext(ctx).
		Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").
		Find(&projects).Error
	if err != nil {
		return err
	}
	botIdx := 0
	for i, project := range projects {
		var plans []model.ProjectPlan
		dp := demoProjects()[i%len(demoProjects())]
		err = gormDB.WithContext(ctx).Where("project_id = ?", project.ID).
			Order("membership_type ASC").Find(&plans).Error
		if err != nil {
			return err
		}
		for j = 0; j < len(plans); j++ {
			plan := plans[j]
			var target int
			if j < len(dp.plans) {
				target = dp.plans[j].supporters
			}
			for k = 0; k < target; k++ {
				if botIdx >= len(botIDs) {
					botIdx = 0
				}
				botID := botIDs[botIdx]
				botIdx++
				periodStart := now.AddDate(0, 0, -7)
				periodEnd := now.AddDate(0, 1, 0)
				sup := model.ProjectSupporter{
					ID:                 uuid.New(),
					ProjectID:          project.ID,
					UserID:             botID,
					PlanID:             plan.ID,
					Status:             model.SupporterStatusActive,
					BillingMode:        model.BillingModeAuto,
					CurrentPeriodStart: periodStart,
					CurrentPeriodEnd:   periodEnd,
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				if err = gormDB.WithContext(ctx).Create(&sup).Error; err != nil {
					return err
				}
				key := fmt.Sprintf("%s-%s", marker, sup.ID.String())
				pur := model.Purchase{
					ID:             uuid.New(),
					ProjectID:      project.ID,
					SupporterID:    &sup.ID,
					BuyerID:        botID,
					Quantity:       1,
					TotalPrice:     plan.Price,
					Status:         model.PurchaseStatusPaid,
					PaymentMethod:  model.PaymentMethodCard,
					PeriodStart:    &periodStart,
					PeriodEnd:      &periodEnd,
					IdempotencyKey: &key,
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				if err = gormDB.WithContext(ctx).Create(&pur).Error; err != nil {
					return err
				}
			}
			plan.SupporterCount = target
			plan.UpdatedAt = now
			if err = gormDB.WithContext(ctx).Save(&plan).Error; err != nil {
				return err
			}
		}
		var totalSupporters int
		for _, pl := range plans {
			totalSupporters += pl.SupporterCount
		}
		project.SupporterCount = totalSupporters
		project.UpdatedAt = now
		if err = gormDB.WithContext(ctx).Save(&project).Error; err != nil {
			return err
		}
	}
	fmt.Println("created: synthetic supporters (BRONZE/SILVER/GOLD)")
	return nil
}

func seedDemoReports(
	ctx context.Context,
	gormDB *gorm.DB,
	buyerID, adminID uuid.UUID,
	users repository.IUserRepo,
	hash string,
	now time.Time,
	reset bool,
) error {
	var (
		projects []model.Project
		messages []model.Message
		buyer2   model.User
		err      error
		i        int
		marker   = "seed-demo-reports-v1"
	)
	if buyerID == uuid.Nil {
		return nil
	}
	buyer2, _, err = upsertUser(ctx, users, seedAccount{
		email: "buyer2@example.com",
		name:  "デモ購入者2",
		role:  model.UserRoleBuyer,
	}, hash, "", now, reset)
	if err != nil {
		return err
	}
	if !reset {
		var count int64
		err = gormDB.WithContext(ctx).Model(&model.Report{}).
			Where("reason LIKE ?", marker+"%").
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	err = gormDB.WithContext(ctx).Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").Limit(4).Find(&projects).Error
	if err != nil {
		return err
	}
	err = gormDB.WithContext(ctx).Where("sender_id = ? OR receiver_id = ?", buyerID, buyerID).
		Order("created_at ASC").Limit(1).Find(&messages).Error
	if err != nil {
		return err
	}
	type reportSeed struct {
		reporterID uuid.UUID
		targetType model.ReportTargetType
		targetID   uuid.UUID
		status     model.ReportStatus
		reason     string
	}
	seeds := make([]reportSeed, 0, 6)
	if len(projects) > 0 {
		seeds = append(seeds,
			reportSeed{buyerID, model.ReportTargetProject, projects[0].ID, model.ReportStatusPending, marker + " 不適切なコンテンツ（未解決）"},
		)
	}
	if len(projects) > 1 {
		seeds = append(seeds,
			reportSeed{buyer2.ID, model.ReportTargetProject, projects[1].ID, model.ReportStatusInReview, marker + " 著作権侵害の疑い（対応中）"},
			reportSeed{buyerID, model.ReportTargetProject, projects[1].ID, model.ReportStatusResolved, marker + " 対応済みの通報"},
		)
	}
	if len(projects) > 2 {
		seeds = append(seeds,
			reportSeed{buyer2.ID, model.ReportTargetProject, projects[2].ID, model.ReportStatusDismissed, marker + " 棄却された通報"},
		)
	}
	if len(messages) > 0 {
		seeds = append(seeds,
			reportSeed{buyer2.ID, model.ReportTargetMessage, messages[0].ID, model.ReportStatusPending, marker + " 迷惑メッセージ（未解決）"},
		)
	}
	for i = range seeds {
		var (
			s   reportSeed
			row model.Report
			res *string
		)
		s = seeds[i]
		row = model.Report{
			ID:         uuid.New(),
			ReporterID: s.reporterID,
			TargetType: s.targetType,
			TargetID:   s.targetID,
			Reason:     s.reason,
			Status:     s.status,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if s.status == model.ReportStatusResolved || s.status == model.ReportStatusDismissed {
			row.ReviewedBy = &adminID
			row.ReviewedAt = &now
			resStr := "デモ審査完了"
			res = &resStr
			row.Resolution = res
		}
		if err = gormDB.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
	}
	fmt.Println("created: demo reports (pending/in_review/resolved/dismissed)")
	return nil
}

func seedAdminPurchases(ctx context.Context, gormDB *gorm.DB, buyerID uuid.UUID, reset bool) error {
	var (
		projects []model.Project
		plans    []model.ProjectPlan
		err      error
		marker   = "seed-admin-purchases-v1"
	)
	if buyerID == uuid.Nil {
		return nil
	}
	if !reset {
		var count int64
		err = gormDB.WithContext(ctx).Model(&model.Purchase{}).
			Where("idempotency_key LIKE ?", marker+"%").
			Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	}
	err = gormDB.WithContext(ctx).Where("status = ?", model.ProjectStatusActive).
		Order("created_at ASC").Offset(3).Limit(2).Find(&projects).Error
	if err != nil || len(projects) == 0 {
		return err
	}
	now := time.Now().UTC()
	type purchaseSeed struct {
		status model.PurchaseStatus
		suffix string
	}
	statuses := []purchaseSeed{
		{model.PurchaseStatusPending, "pending"},
		{model.PurchaseStatusCancelled, "cancelled"},
		{model.PurchaseStatusRefunded, "refunded"},
	}
	for i, project := range projects {
		err = gormDB.WithContext(ctx).Where("project_id = ?", project.ID).
			Order("membership_type ASC").Limit(1).Find(&plans).Error
		if err != nil || len(plans) == 0 {
			continue
		}
		plan := plans[0]
		ps := statuses[i%len(statuses)]
		key := marker + "-" + ps.suffix + "-" + project.ID.String()
		pur := model.Purchase{
			ID:             uuid.New(),
			ProjectID:      project.ID,
			BuyerID:        buyerID,
			Quantity:       1,
			TotalPrice:     plan.Price,
			Status:         ps.status,
			PaymentMethod:  model.PaymentMethodCard,
			IdempotencyKey: &key,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err = gormDB.WithContext(ctx).Create(&pur).Error; err != nil {
			return err
		}
	}
	fmt.Println("created: admin review purchases (pending/cancelled/refunded)")
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
