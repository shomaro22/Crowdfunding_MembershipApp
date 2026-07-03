# Git 履歴から秘密値を除去する手順

`docker-compose.yml` 等に誤ってコミットした **Google Client Secret** などは、ファイルを直しただけでは履歴に残ります。  
クライアントシークレットは **Google Cloud Console でローテーション済み** であれば漏洩リスクは下がりますが、履歴掃除は別途実施してください。

## 前提

- 作業前に **全員へ共有**（force-push 後は各自 `git fetch` / 再 clone が必要）
- **バックアップ**を取る（`git clone --mirror` 等）
- ローテーション済みの新シークレットは **`.env` のみ** に置く（`.env` は `.gitignore` 対象）

## リポジトリ URL の確認

`YOUR_ORG` は **プレースホルダ** です。そのまま実行しないでください。

```bash
cd /path/to/Crowdfunding_MembershipApp
git remote get-url origin
```

このプロジェクトの例:

| 方式 | URL |
|------|-----|
| HTTPS | `https://github.com/shomaro22/Crowdfunding_MembershipApp.git` |
| SSH | `git@github.com:shomaro22/Crowdfunding_MembershipApp.git` |

`ERROR: Repository not found` の主な原因:

- `YOUR_ORG` を置き換えていない
- SSH 鍵が GitHub に未登録（HTTPS なら `gh auth login` または PAT）
- リポジトリ名・オーナーの typo

## 方法 A: git filter-repo（推奨）

[git-filter-repo](https://github.com/newren/git-filter-repo) をインストールしたうえで:

```bash
# 0. 作業用ディレクトリ（既存の作業 clone とは別にする）
cd ~/Desktop   # 例

# 1. ミラー clone（origin の URL をそのまま使う）
git clone --mirror https://github.com/shomaro22/Crowdfunding_MembershipApp.git crowdfund-mirror
# SSH を使う場合:
# git clone --mirror git@github.com:shomaro22/Crowdfunding_MembershipApp.git crowdfund-mirror

cd crowdfund-mirror

# 2. 履歴から秘密文字列を置換
# まず履歴に残っている実値を特定する（プレースホルダでは置換されない）
git grep 'GOCSPX-' $(git rev-list --all) | head
git grep 'apps.googleusercontent.com' $(git rev-list --all) | head

# 上で見つかった文字列を replacements.txt に書く（例）
cat > /tmp/replacements.txt <<'EOF'
GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx==>***REMOVED***
123456789012-xxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com==>***REMOVED_CLIENT_ID***
EOF
git filter-repo --replace-text /tmp/replacements.txt
rm /tmp/replacements.txt

# filter-repo は誤 push 防止のため origin を削除する。push 前に戻す。
git remote add origin https://github.com/shomaro22/Crowdfunding_MembershipApp.git
# SSH の場合:
# git remote add origin git@github.com:shomaro22/Crowdfunding_MembershipApp.git

# 3. 強制 push（main / develop など対象ブランチすべて）
git push --force --all origin
git push --force --tags origin
```

`docker-compose.yml` 全体を過去版から書き換えたい場合:

```bash
git filter-repo --path docker-compose.yml --invert-paths   # 履歴から一旦削除
# その後正しい docker-compose.yml を含むコミットを再度 push する、など方針を決める
```

## 方法 B: BFG Repo-Cleaner

```bash
brew install bfg   # または jar を直接利用

git clone --mirror https://github.com/shomaro22/Crowdfunding_MembershipApp.git Crowdfunding_MembershipApp.git
cd Crowdfunding_MembershipApp.git

# 秘密文字列を ***REMOVED*** に置換
bfg --replace-text secrets.txt

git reflog expire --expire=now --all
git gc --prune=now --aggressive
git push --force
```

`secrets.txt` の例:

```
GOCSPX-xxxxxxxx==>***REMOVED***
```

## 作業後の確認

`-S 'GOOGLE_CLIENT_SECRET'` は **環境変数名** の追加・削除も拾うため、Fix コミットがヒットしても正常です。  
**漏洩した実際の値**（`GOCSPX-...` など）が残っていないかを確認してください。

```bash
# 実際の秘密値が履歴に残っていないか（何も出なければ OK）
git grep 'GOCSPX-' $(git rev-list --all)
git grep 'apps.googleusercontent.com' $(git rev-list --all) | rg -v '\$\{GOOGLE_CLIENT_ID'

# 参考: 環境変数名の変更履歴（Fix コミットが出るのは OK）
git log --oneline --all -S 'GOOGLE_CLIENT_SECRET' -- docker-compose.yml

# 現在の main が安全か
git show origin/main:docker-compose.yml | rg 'GOCSPX|GOOGLE_CLIENT'
```

`git log -p` の末尾が `:` で止まるのは **less ページャ** です。`q` で終了、`Space` で続きを表示。

## チーム向けリカバリ

履歴書き換え後、各開発者:

```bash
git fetch origin
git reset --hard origin/main   # 作業中ブランチは rebase し直し
```

未 push のローカルブランチは、古い履歴を参照している可能性があるため **作業ツリーを退避してから** 取り込み直してください。

## 再発防止

1. 秘密値は **`.env` のみ**（`.env.example` はプレースホルダのみ）
2. `docker-compose.yml` にデフォルト秘密値を書かない（`${VAR:-}` の空デフォルト）
3. pre-commit で `GOCSPX-` / `sk_live_` 等のパターン検出（任意）
4. GitHub **Secret scanning** を有効化
