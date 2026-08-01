package panel

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/kejilion/kejilion-panel/internal/auth"
)

type recoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

func (s *Server) handleTOTPSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if r.Method == http.MethodDelete && !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		status, err := s.auth.TOTPStatus(session.User.ID)
		if err != nil {
			s.writeProblem(w, r, http.StatusInternalServerError, "totp_status_failed", "Unable to read two-factor authentication status", "")
			return
		}
		s.writeJSON(w, http.StatusOK, status)
		return
	}
	if !s.checkCSRF(w, r, session) {
		return
	}
	var input struct {
		CurrentPassword string `json:"currentPassword"`
		SecondFactor    string `json:"secondFactor"`
	}
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	const action = "auth.totp.disable"
	if err := s.audit(r, session.User.ID, action, "user", session.User.ID, "intent", nil); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	if err := s.auth.DisableTOTP(session.User.ID, input.CurrentPassword, input.SecondFactor); err != nil {
		_ = s.audit(r, session.User.ID, action, "user", session.User.ID, "failure", nil)
		s.writeTOTPProblem(w, r, err, "totp_disable_failed", "Unable to disable two-factor authentication")
		return
	}
	s.clearAuthCookies(w, r)
	_ = s.audit(r, session.User.ID, action, "user", session.User.ID, "success", nil)
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	if r.Method == http.MethodPost {
		var input struct {
			CurrentPassword string `json:"currentPassword"`
		}
		if err := s.decodeJSON(w, r, &input); err != nil {
			return
		}
		enrollment, err := s.auth.StartTOTPEnrollment(session.User.ID, input.CurrentPassword)
		if err != nil {
			s.writeTOTPProblem(w, r, err, "totp_enrollment_failed", "Unable to start two-factor enrollment")
			return
		}
		s.writeJSON(w, http.StatusCreated, enrollment)
		return
	}

	var input struct {
		EnrollmentID string `json:"enrollmentId"`
		Code         string `json:"code"`
	}
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	const action = "auth.totp.enable"
	if err := s.audit(r, session.User.ID, action, "user", session.User.ID, "intent", nil); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	codes, err := s.auth.ConfirmTOTPEnrollment(session.User.ID, input.EnrollmentID, input.Code)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "user", session.User.ID, "failure", nil)
		s.writeTOTPProblem(w, r, err, "totp_enable_failed", "Unable to enable two-factor authentication")
		return
	}
	s.clearAuthCookies(w, r)
	_ = s.audit(r, session.User.ID, action, "user", session.User.ID, "success", map[string]any{"recoveryCodeCount": len(codes)})
	s.writeJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

func (s *Server) handleTOTPRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "")
		return
	}
	if !s.checkOrigin(w, r) {
		return
	}
	_, session, ok := s.requireSession(w, r)
	if !ok || !s.checkCSRF(w, r, session) {
		return
	}
	var input struct {
		CurrentPassword string `json:"currentPassword"`
		SecondFactor    string `json:"secondFactor"`
	}
	if err := s.decodeJSON(w, r, &input); err != nil {
		return
	}
	const action = "auth.totp.recovery_codes.rotate"
	if err := s.audit(r, session.User.ID, action, "user", session.User.ID, "intent", nil); err != nil {
		s.writeProblem(w, r, http.StatusServiceUnavailable, "audit_unavailable", "Audit storage unavailable", "")
		return
	}
	codes, err := s.auth.RegenerateRecoveryCodes(session.User.ID, input.CurrentPassword, input.SecondFactor)
	if err != nil {
		_ = s.audit(r, session.User.ID, action, "user", session.User.ID, "failure", nil)
		s.writeTOTPProblem(w, r, err, "totp_recovery_codes_failed", "Unable to regenerate recovery codes")
		return
	}
	s.clearAuthCookies(w, r)
	_ = s.audit(r, session.User.ID, action, "user", session.User.ID, "success", map[string]any{"recoveryCodeCount": len(codes)})
	s.writeJSON(w, http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

func (s *Server) writeTOTPProblem(w http.ResponseWriter, r *http.Request, err error, fallbackCode, fallbackTitle string) {
	var rateError *auth.RateLimitError
	switch {
	case errors.As(err, &rateError):
		seconds := max(int(rateError.RetryAfter.Seconds()), 1)
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		s.writeProblem(w, r, http.StatusTooManyRequests, "authentication_rate_limited", "Authentication rate limited", "")
	case errors.Is(err, auth.ErrInvalidCurrentPassword):
		s.writeValidationProblem(w, r, "currentPassword", err.Error())
	case errors.Is(err, auth.ErrInvalidSecondFactor):
		s.writeValidationProblem(w, r, "secondFactor", err.Error())
	case errors.Is(err, auth.ErrTOTPAlreadyEnabled):
		s.writeProblem(w, r, http.StatusConflict, "totp_already_enabled", "Two-factor authentication is already enabled", "")
	case errors.Is(err, auth.ErrTOTPDisabled):
		s.writeProblem(w, r, http.StatusConflict, "totp_not_enabled", "Two-factor authentication is not enabled", "")
	case errors.Is(err, auth.ErrTOTPEnrollmentExpired):
		s.writeProblem(w, r, http.StatusGone, "totp_enrollment_expired", "Two-factor enrollment expired", "")
	case errors.Is(err, auth.ErrSecondFactorUnavailable):
		s.writeProblem(w, r, http.StatusServiceUnavailable, "second_factor_unavailable", "Two-factor authentication is temporarily unavailable", "Use a recovery code or restore the Panel TOTP encryption key.")
	default:
		s.writeProblem(w, r, http.StatusInternalServerError, fallbackCode, fallbackTitle, "")
	}
}
