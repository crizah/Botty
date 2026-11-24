package handlers

import (
	"fmt"
	"net/http"

	"server/internal/schemas"
	"server/internal/services"
)

type ClaimHandler struct {
	svc     *services.ClaimService
	devMode bool
}

func NewClaimHandler(svc *services.ClaimService, devMode bool) *ClaimHandler {
	return &ClaimHandler{svc, devMode}
}

// Create generates a one-time "@botty link" code for the signed-in user.
func (h *ClaimHandler) Create(w http.ResponseWriter, r *http.Request) {
	out, err := h.svc.Create(r.Context(), uid(r))
	respond(w, http.StatusCreated, out, err)
}

// DevRedeem pretends the bot saw "@botty link <code>" in a new group.
// TODO: change - dev only; the ingest worker calls ClaimService.Redeem with the real group.
func (h *ClaimHandler) DevRedeem(w http.ResponseWriter, r *http.Request) {
	var req schemas.DevRedeemRequest
	if !decode(w, r, &req) {
		return
	}
	g, err := h.svc.Redeem(r.Context(), req.Code, fmt.Sprintf("dev-%s@g.us", req.Code), req.Name)
	respond(w, http.StatusCreated, schemas.GroupResponse{Group: g}, err)
}
