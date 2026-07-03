import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";

import { checkEmailAvailable, registerSeller } from "@/api/auth";
import { ApiError } from "@/lib/api-client";
import { lookupJapaneseAddress } from "@/lib/postal-jp";
import { PasswordInput } from "@/components/common/PasswordInput";
import { digitsOnly } from "@/lib/validation";
import {
  isValidEmail,
  sanitizeDisplayText,
  sanitizeEmail,
  validatePassword,
} from "@/lib/utils";

const ALLOWED_IMAGE_TYPES = new Set(["image/jpeg", "image/png", "image/webp"]);
const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
const STEPS = ["アカウント", "本人情報", "本人確認書類", "確認"] as const;

const YEARS = Array.from({ length: 80 }, (_, i) =>
  String(new Date().getFullYear() - 18 - i),
);
const MONTHS = Array.from({ length: 12 }, (_, i) =>
  String(i + 1).padStart(2, "0"),
);
const DAYS = Array.from({ length: 31 }, (_, i) =>
  String(i + 1).padStart(2, "0"),
);

function validateImageFile(file: File | null, label: string): string | null {
  if (!file) {
    return `${label}を選択してください`;
  }
  if (!ALLOWED_IMAGE_TYPES.has(file.type)) {
    return `${label}は JPEG / PNG / WebP のみ対応しています`;
  }
  if (file.size > MAX_IMAGE_BYTES) {
    return `${label}は 5MB 以下にしてください`;
  }
  return null;
}

export function SellerSignUpForm() {
  const navigate = useNavigate();
  const [step, setStep] = useState(0);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [birthYear, setBirthYear] = useState("");
  const [birthMonth, setBirthMonth] = useState("");
  const [birthDay, setBirthDay] = useState("");
  const [postalCode, setPostalCode] = useState("");
  const [address, setAddress] = useState("");
  const [frontFile, setFrontFile] = useState<File | null>(null);
  const [backFile, setBackFile] = useState<File | null>(null);
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [emailTaken, setEmailTaken] = useState(false);
  const [postalLoading, setPostalLoading] = useState(false);

  const birthDate =
    birthYear && birthMonth && birthDay
      ? `${birthYear}-${birthMonth}-${birthDay}`
      : "";

  const validateStep = (targetStep: number): string | null => {
    if (targetStep === 0) {
      if (!isValidEmail(email)) {
        return "有効なメールアドレスを入力してください";
      }
      if (emailTaken) {
        return "このメールアドレスは既に登録されています";
      }
      const pwCheck = validatePassword(password);
      if (!pwCheck.valid) {
        return pwCheck.errors[0] ?? "パスワードが無効です";
      }
      if (password !== confirmPassword) {
        return "確認用パスワードが一致しません";
      }
    }
    if (targetStep === 1) {
      if (!sanitizeDisplayText(fullName, 200)) {
        return "氏名を入力してください";
      }
      if (!birthDate) {
        return "生年月日を選択してください";
      }
      if (!sanitizeDisplayText(address, 500)) {
        return "住所を入力してください";
      }
    }
    if (targetStep === 2) {
      const frontErr = validateImageFile(frontFile, "身分証（表面）");
      if (frontErr) {
        return frontErr;
      }
      const backErr = validateImageFile(backFile, "身分証（裏面）");
      if (backErr) {
        return backErr;
      }
    }
    return null;
  };

  const onEmailBlur = async () => {
    if (!isValidEmail(email)) {
      setEmailTaken(false);
      return;
    }
    try {
      const available = await checkEmailAvailable(sanitizeEmail(email));
      setEmailTaken(!available);
    } catch {
      setEmailTaken(false);
    }
  };

  const onPostalBlur = async () => {
    const digits = postalCode.replace(/\D/g, "");
    if (digits.length !== 7) {
      return;
    }
    setPostalLoading(true);
    try {
      const found = await lookupJapaneseAddress(digits);
      if (found) {
        setAddress(found);
      }
    } catch {
      /* 手入力で続行 */
    } finally {
      setPostalLoading(false);
    }
  };

  const goNext = () => {
    setError("");
    const validationMessage = validateStep(step);
    if (validationMessage !== null) {
      setError(validationMessage);
      return;
    }
    setStep((s) => Math.min(s + 1, STEPS.length - 1));
  };

  const goBack = () => {
    setError("");
    setStep((s) => Math.max(s - 1, 0));
  };

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    for (let i = 0; i < STEPS.length - 1; i++) {
      const validationMessage = validateStep(i);
      if (validationMessage !== null) {
        setError(validationMessage);
        setStep(i);
        return;
      }
    }
    if (!frontFile || !backFile) {
      setError("身分証画像が不足しています");
      setStep(2);
      return;
    }

    setIsSubmitting(true);
    try {
      await registerSeller({
        email: sanitizeEmail(email),
        password,
        full_name: sanitizeDisplayText(fullName, 200),
        birth_date: birthDate,
        address: sanitizeDisplayText(address, 500),
        identity_document_front: frontFile,
        identity_document_back: backFile,
      });
      void navigate("/signin", {
        replace: true,
        state: {
          message: "出品者登録を受け付けました。審査完了までお待ちください。",
        },
      });
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "BAD_SELLER_KYC") {
          setError(
            "本人情報が不足しています。氏名・生年月日・住所を確認してください。",
          );
        } else if (err.code === "CONFLICT") {
          setError("このメールアドレスは既に登録されています");
        } else {
          setError(err.message);
        }
      } else {
        setError("出品者登録に失敗しました");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="mx-auto grid max-w-4xl overflow-hidden rounded-2xl bg-white shadow-xl md:grid-cols-5">
      <div className="flex flex-col justify-center bg-gradient-to-br from-purple-700 via-purple-600 to-pink-600 p-8 text-white md:col-span-2 md:p-10">
        <p className="mb-2 text-sm font-semibold uppercase tracking-wider text-purple-200">
          FanFunding
        </p>
        <h2 className="mb-4 text-2xl font-extrabold md:text-3xl">
          クリエイター登録
        </h2>
        <p className="text-sm leading-relaxed text-purple-100">
          本人確認（KYC）後、プロジェクトの出品が可能になります。
        </p>
        <ol className="mt-8 space-y-3">
          {STEPS.map((label, i) => (
            <li
              key={label}
              className={`flex items-center gap-3 text-sm ${i === step ? "font-bold text-white" : i < step ? "text-purple-200" : "text-purple-300/70"}`}
            >
              <span
                className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-bold ${i <= step ? "bg-white text-purple-700" : "border border-purple-300/50"}`}
              >
                {i + 1}
              </span>
              {label}
            </li>
          ))}
        </ol>
      </div>

      <div className="p-8 md:col-span-3 md:p-10">
        <div className="mb-6">
          <h3 className="text-xl font-bold text-gray-800">
            ステップ {step + 1}: {STEPS[step]}
          </h3>
          <p className="text-sm text-gray-500">
            KYC 審査のため正確な情報を入力してください
          </p>
        </div>

        {error ? (
          <p className="mb-4 rounded-lg bg-red-50 px-4 py-2 text-sm text-red-600">
            {error}
          </p>
        ) : null}

        <form onSubmit={(e) => void onSubmit(e)} className="space-y-4">
          {step === 0 ? (
            <>
              <div>
                <label
                  htmlFor="email"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  メールアドレス
                </label>
                <input
                  id="email"
                  type="email"
                  autoComplete="email"
                  className="input-field"
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value);
                    setEmailTaken(false);
                  }}
                  onBlur={() => void onEmailBlur()}
                />
                {emailTaken ? (
                  <p className="mt-1 text-sm text-amber-700">
                    このメールアドレスは既に登録されています。
                    <Link
                      to="/signin"
                      className="ml-1 font-medium text-purple-700 underline"
                    >
                      ログインはこちら
                    </Link>
                  </p>
                ) : null}
              </div>
              <div>
                <label
                  htmlFor="password"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  パスワード
                </label>
                <PasswordInput
                  id="password"
                  value={password}
                  onChange={setPassword}
                  autoComplete="new-password"
                />
              </div>
              <div>
                <label
                  htmlFor="confirmPassword"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  パスワード（確認）
                </label>
                <PasswordInput
                  id="confirmPassword"
                  value={confirmPassword}
                  onChange={setConfirmPassword}
                  autoComplete="new-password"
                />
              </div>
            </>
          ) : null}

          {step === 1 ? (
            <>
              <div>
                <label
                  htmlFor="fullName"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  氏名
                </label>
                <input
                  id="fullName"
                  type="text"
                  className="input-field"
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  生年月日
                </label>
                <div className="grid grid-cols-3 gap-2">
                  <select
                    className="input-field"
                    value={birthYear}
                    onChange={(e) => setBirthYear(e.target.value)}
                  >
                    <option value="">年</option>
                    {YEARS.map((y) => (
                      <option key={y} value={y}>
                        {y}年
                      </option>
                    ))}
                  </select>
                  <select
                    className="input-field"
                    value={birthMonth}
                    onChange={(e) => setBirthMonth(e.target.value)}
                  >
                    <option value="">月</option>
                    {MONTHS.map((m) => (
                      <option key={m} value={m}>
                        {Number(m)}月
                      </option>
                    ))}
                  </select>
                  <select
                    className="input-field"
                    value={birthDay}
                    onChange={(e) => setBirthDay(e.target.value)}
                  >
                    <option value="">日</option>
                    {DAYS.map((d) => (
                      <option key={d} value={d}>
                        {Number(d)}日
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <div>
                <label
                  htmlFor="postalCode"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  郵便番号（ハイフンなし）
                </label>
                <input
                  id="postalCode"
                  inputMode="numeric"
                  className="input-field"
                  value={postalCode}
                  onChange={(e) => setPostalCode(digitsOnly(e.target.value, 7))}
                  onBlur={() => void onPostalBlur()}
                  placeholder="例: 1000001"
                />
                {postalLoading ? (
                  <p className="mt-1 text-xs text-gray-500">住所を検索中...</p>
                ) : null}
              </div>
              <div>
                <label
                  htmlFor="address"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  住所
                </label>
                <textarea
                  id="address"
                  className="input-field min-h-24"
                  value={address}
                  onChange={(e) => setAddress(e.target.value)}
                />
              </div>
            </>
          ) : null}

          {step === 2 ? (
            <>
              <div>
                <label
                  htmlFor="front"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  身分証（表面）
                </label>
                <input
                  id="front"
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  onChange={(e) => setFrontFile(e.target.files?.[0] ?? null)}
                />
                {frontFile ? (
                  <p className="mt-1 text-xs text-gray-500">
                    選択済: {frontFile.name}
                  </p>
                ) : null}
              </div>
              <div>
                <label
                  htmlFor="back"
                  className="mb-1 block text-sm font-medium text-gray-700"
                >
                  身分証（裏面）
                </label>
                <input
                  id="back"
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  onChange={(e) => setBackFile(e.target.files?.[0] ?? null)}
                />
                {backFile ? (
                  <p className="mt-1 text-xs text-gray-500">
                    選択済: {backFile.name}
                  </p>
                ) : null}
              </div>
            </>
          ) : null}

          {step === 3 ? (
            <div className="space-y-3 rounded-xl border border-purple-100 bg-purple-50/50 p-4 text-sm">
              <p className="text-base font-bold text-gray-800">
                入力内容の確認
              </p>
              <p className="text-xs text-gray-500">
                以下の内容で出品者登録を申請します。修正があれば「戻る」で編集してください。
              </p>
              <p>
                <span className="font-medium text-gray-700">メール:</span>{" "}
                {email}
              </p>
              <p>
                <span className="font-medium text-gray-700">氏名:</span>{" "}
                {fullName}
              </p>
              <p>
                <span className="font-medium text-gray-700">生年月日:</span>{" "}
                {birthDate}
              </p>
              <p>
                <span className="font-medium text-gray-700">住所:</span>{" "}
                {address}
              </p>
              <p>
                <span className="font-medium text-gray-700">身分証:</span>{" "}
                {frontFile?.name} / {backFile?.name}
              </p>
              <p className="text-xs text-gray-500">
                送信後、管理者による KYC 審査が行われます（通常 1〜3 営業日）。
              </p>
            </div>
          ) : null}

          <div className="flex gap-3 pt-2">
            {step > 0 ? (
              <button
                type="button"
                onClick={goBack}
                className="btn-secondary flex-1 rounded-full font-bold"
              >
                戻る
              </button>
            ) : null}
            {step < STEPS.length - 1 ? (
              <button
                type="button"
                onClick={goNext}
                className="btn-primary flex-1 rounded-full font-bold"
              >
                {step === STEPS.length - 2 ? "確認画面へ" : "次へ"}
              </button>
            ) : (
              <button
                type="submit"
                disabled={isSubmitting || emailTaken}
                className="btn-primary flex-1 rounded-full font-bold"
              >
                {isSubmitting ? "送信中..." : "この内容で登録申請する"}
              </button>
            )}
          </div>
        </form>

        <p className="mt-6 text-center text-sm text-gray-600">
          ファン（購入者）の方は{" "}
          <Link to="/signup" className="text-purple-700 hover:underline">
            こちら
          </Link>
        </p>
      </div>
    </div>
  );
}
