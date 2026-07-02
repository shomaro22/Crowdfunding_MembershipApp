package main

import (
	"fmt"
	"strings"

	"crowdfund-app/model"
)

func formatDemoGenre(tpl string, genre string) string {
	if strings.Contains(tpl, "%") {
		return fmt.Sprintf(tpl, genre)
	}
	return tpl
}

var demoCategoryTemplates = []struct {
	titleFmt string
	descFmt  string
	target   int64
	bronze   int
	silver   int
	gold     int
}{
	{
		"月額ファンクラブ｜%sの限定コンテンツ",
		"%sの制作過程・限定配信・コミュニティ特典をお届けするメンバーシップです。",
		300_000, 500, 1_500, 3_000,
	},
	{
		"%sクリエイターの裏側メンバーシップ",
		"未公開素材・制作Vlog・月例ライブ。%sの活動の舞台裏をファンと一緒に楽しむコミュニティです。",
		250_000, 400, 1_200, 2_500,
	},
	{
		"%sプロジェクト｜会員制サポートコミュニティ",
		"目標達成に向けた進捗レポートと会員限定特典。%sを支えるファンのための月額プランです。",
		400_000, 600, 1_800, 3_500,
	},
	{
		"%sの制作室｜メンバー限定アーカイブ",
		"日々の創作活動・制作ノウハウ・限定コンテンツを会員向けに公開します。",
		200_000, 350, 900, 2_000,
	},
	{
		"%sファンクラブ｜限定配信＆交流",
		"通常配信に加え、メンバー限定の深掘りトーク・制作秘話・アフタートークをお届けします。",
		180_000, 300, 800, 1_800,
	},
}

var demoGenreNames = []string{
	"イラスト", "音楽", "ゲーム開発", "フィットネス", "写真", "ポッドキャスト", "ハンドメイド",
	"料理", "アニメーション", "ライティング", "VTuber", "陶芸", "園芸", "プログラミング",
	"ダンス", "映画レビュー", "ボードゲーム", "キャンプ", "ペット", "書道",
	"コスプレ", "3Dモデリング", "ASMR", "旅行Vlog", "漫画", "ラジオ",
	"ヨガ", "釣り", "茶道", "ワイン", "カフェ巡り", "DIY", "子育て",
	"教育", "科学実験", "歴史解説", "語学学習", "占い", "アウトドア料理",
	"クラシック音楽", "ストリートアート", "写真プリント", "レコード収集", "盆栽",
	"編み物", "レザークラフト", "香水作り", "天体観測", "マジック",
}

func extendedDemoProjects() []demoProject {
	var (
		out     []demoProject
		i       int
		j       int
		genre   string
		tpl     struct {
			titleFmt string
			descFmt  string
			target   int64
			bronze   int
			silver   int
			gold     int
		}
		favBase int
		curAmt  int64
		supB    int
		supS    int
		supG    int
		dp      demoProject
	)

	out = make([]demoProject, 0, 50)
	for i = 0; i < 50; i++ {
		genre = demoGenreNames[i%len(demoGenreNames)]
		tpl = demoCategoryTemplates[i%len(demoCategoryTemplates)]
		favBase = 20 + (i*17)%400
		curAmt = tpl.target * int64(40+(i*13)%120) / 100
		supB = 3 + (i % 15)
		supS = 1 + (i % 6)
		supG = 1 + (i % 3)
		if i%5 == 0 {
			curAmt = tpl.target * int64(105+(i%20)) / 100
		}

		dp = demoProject{
			title:         formatDemoGenre(tpl.titleFmt, genre),
			description:   formatDemoGenre(tpl.descFmt, genre),
			targetAmount:  tpl.target,
			currentAmount: curAmt,
			favoriteCount: favBase,
			imageURL:      fmt.Sprintf("https://picsum.photos/seed/crowdfund-demo-%d/800/600", i+100),
			plans: []demoPlan{
				{model.MembershipBronze, tpl.bronze, fmt.Sprintf("%s限定コンテンツ＋ニュースレター", genre), supB},
				{model.MembershipSilver, tpl.silver, fmt.Sprintf("%s制作過程・会員Discord", genre), supS},
				{model.MembershipGold, tpl.gold, fmt.Sprintf("%s月例ライブ＋リクエスト枠", genre), supG},
			},
		}
		_ = j
		out = append(out, dp)
	}
	return out
}
