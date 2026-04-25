package auth

import "crypto/subtle"

// RegistrationValidator abstracts how the API validates agent registration tokens.
// The v1 implementation checks a single pre-shared token from config.
// Future implementations can validate per-agent tokens, signed JWTs, etc.
type RegistrationValidator interface {
	Valid(token string) bool
}

// StaticTokenValidator accepts a single pre-shared token set via AOP_REGISTRATION_TOKEN.
type StaticTokenValidator struct {
	token string
}

func NewStaticTokenValidator(token string) *StaticTokenValidator {
	return &StaticTokenValidator{token: token}
}

func (v *StaticTokenValidator) Valid(token string) bool {
	return subtle.ConstantTimeCompare([]byte(token), []byte(v.token)) == 1
}
