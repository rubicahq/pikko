package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/rbac"
	"github.com/rubicahq/pikko/services/api/internal/team"
)

type TeamHandlers struct {
	Team *team.Service
	Cfg  *config.Config
}

func (h *TeamHandlers) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	sameSite := http.SameSiteLaxMode
	switch h.Cfg.CookieSameSite {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.Cfg.CookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.Cfg.CookieDomain,
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   h.Cfg.CookieSecure,
		SameSite: sameSite,
	})
}

func (h *TeamHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	list, err := h.Team.ListMembers(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list team members.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"members":     list,
		"roles":       rbac.InviteRoles(),
		"permissions": rbac.PermissionsFor(MembershipRole(r)),
		"my_role":     MembershipRole(r),
	})
}

func (h *TeamHandlers) ListInvites(w http.ResponseWriter, r *http.Request) {
	list, err := h.Team.ListInvites(r.Context(), StoreID(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "list_failed", "Unable to list invites.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invites": list})
}

func (h *TeamHandlers) CreateInvite(w http.ResponseWriter, r *http.Request) {
	var in team.CreateInviteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	inv, err := h.Team.CreateInvite(r.Context(), StoreID(r), UserID(r), in)
	if err != nil {
		h.writeTeamErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invite": inv})
}

func (h *TeamHandlers) ResendInvite(w http.ResponseWriter, r *http.Request) {
	inviteID, err := uuid.Parse(r.PathValue("inviteID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Invite not found.")
		return
	}
	inv, err := h.Team.ResendInvite(r.Context(), StoreID(r), UserID(r), inviteID)
	if err != nil {
		h.writeTeamErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite": inv})
}

func (h *TeamHandlers) CancelInvite(w http.ResponseWriter, r *http.Request) {
	inviteID, err := uuid.Parse(r.PathValue("inviteID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Invite not found.")
		return
	}
	inv, err := h.Team.CancelInvite(r.Context(), StoreID(r), UserID(r), inviteID)
	if err != nil {
		h.writeTeamErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite": inv})
}

func (h *TeamHandlers) RevokeMember(w http.ResponseWriter, r *http.Request) {
	membershipID, err := uuid.Parse(r.PathValue("membershipID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Member not found.")
		return
	}
	if err := h.Team.RevokeMember(r.Context(), StoreID(r), UserID(r), membershipID); err != nil {
		h.writeTeamErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *TeamHandlers) PatchMember(w http.ResponseWriter, r *http.Request) {
	membershipID, err := uuid.Parse(r.PathValue("membershipID"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "Member not found.")
		return
	}
	var in team.PatchMemberInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	m, err := h.Team.PatchMember(r.Context(), StoreID(r), UserID(r), membershipID, in)
	if err != nil {
		h.writeTeamErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"member": m})
}

func (h *TeamHandlers) PeekInvite(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	peek, err := h.Team.PeekInvite(r.Context(), token)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "peek_failed", "Unable to check invite.")
		return
	}
	if !peek.Valid {
		writeError(w, r, http.StatusGone, "invalid_invite", "This invite has expired or is no longer valid.")
		return
	}
	writeJSON(w, http.StatusOK, peek)
}

func (h *TeamHandlers) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body.")
		return
	}
	res, err := h.Team.AcceptInvite(r.Context(), body.Token, ClientIP(r), r.UserAgent())
	if err != nil {
		h.writeTeamErr(w, r, err)
		return
	}
	h.setSessionCookie(w, res.SessionToken, time.Now().UTC().Add(h.Cfg.SessionTTL))
	writeJSON(w, http.StatusOK, map[string]any{
		"user":          res.User,
		"store_id":      res.StoreID,
		"role":          res.Role,
		"is_new_account": res.IsNewAccount,
		"redirect_path": res.RedirectPath,
	})
}

func (h *TeamHandlers) Me(w http.ResponseWriter, r *http.Request) {
	role := MembershipRole(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"role":        role,
		"permissions": rbac.PermissionsFor(role),
	})
}

func (h *TeamHandlers) writeTeamErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, team.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Not found.")
	case errors.Is(err, team.ErrForbidden):
		writeError(w, r, http.StatusForbidden, "forbidden", "You do not have permission for this action.")
	case errors.Is(err, team.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_input", "Check the invite details and try again.")
	case errors.Is(err, team.ErrAlreadyMember):
		writeError(w, r, http.StatusConflict, "already_member", "This person is already a store member.")
	case errors.Is(err, team.ErrInvitePending):
		writeError(w, r, http.StatusConflict, "invite_pending", "An invite is already pending for this email.")
	case errors.Is(err, team.ErrInviteInvalid):
		writeError(w, r, http.StatusGone, "invalid_invite", "This invite has expired or is no longer valid.")
	case errors.Is(err, team.ErrCannotRevoke):
		writeError(w, r, http.StatusConflict, "cannot_revoke", "This membership cannot be changed.")
	case errors.Is(err, team.ErrCooldown):
		writeError(w, r, http.StatusTooManyRequests, "cooldown", "Please wait before resending.")
	default:
		writeError(w, r, http.StatusInternalServerError, "team_failed", "Unable to complete team operation.")
	}
}
