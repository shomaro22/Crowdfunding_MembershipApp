package middleware

import "time"

var (
	Policy2FAChallenge = RatePolicy{
		Name:   "2fa/challenge",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 5, Key: KeyIP}},
	}
	PolicyGoogleOAuth = RatePolicy{
		Name:   "google_oauth",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 20, Key: KeyIP}},
	}
	PolicyOAuthExchange = RatePolicy{
		Name:   "oauth_exchange",
		Window: time.Minute,
		Rules:  []RateRule{{Limit: 20, Key: KeyIP}},
	}
)
