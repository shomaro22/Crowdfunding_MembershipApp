package middleware

import "time"

func Policy2FAChallenge(secret []byte) RatePolicy {
	return RatePolicy{
		Name:   "2fa/challenge",
		Period: time.Minute,
		Rules: []RateRule{
			{Limit: 5, Key: KeyIP},
			{Limit: 5, Key: KeyPending2FAUser(secret)},
		},
	}
}

var (
	PolicyGoogleOAuth = RatePolicy{
		Name:   "google_oauth",
		Period: time.Minute,
		Rules:  []RateRule{{Limit: 20, Key: KeyIP}},
	}
	PolicyOAuthExchange = RatePolicy{
		Name:   "oauth_exchange",
		Period: time.Minute,
		Rules:  []RateRule{{Limit: 20, Key: KeyIP}},
	}
	PolicyPayment = RatePolicy{
		Name:   "payment",
		Period: time.Minute,
		Rules:  []RateRule{{Limit: 10, Key: KeyIP}},
	}
)
