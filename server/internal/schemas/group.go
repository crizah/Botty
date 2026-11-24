package schemas

import (
	"errors"
	"strings"
	"time"

	db "server/internal/db/sqlc"
)

// --- Requests ---

// DevCreateGroupRequest is only used by the dev-only "simulate a group" endpoint.
// TODO: change - delete with the endpoint once ingest discovers real groups.
type DevCreateGroupRequest struct {
	Name    string   `json:"name"`
	Members []string `json:"members"`
}

// --- Responses ---

type BotInfoResponse struct {
	Number  string `json:"number"`
	DevMode bool   `json:"dev_mode"`
}

type GroupsResponse struct {
	Groups []db.ListGroupsForOwnerRow `json:"groups"`
}

type GroupResponse struct {
	Group db.WaGroup `json:"group"`
}

type ProjectGroupsResponse struct {
	Groups []db.ListProjectGroupsRow `json:"groups"`
}

type MembersResponse struct {
	Members []db.WaGroupMember `json:"members"`
}

type AllowlistResponse struct {
	Members []db.ListAllowlistRow `json:"members"`
}

// ClaimResponse is a one-time code the user sends in a group as "@botty link <code>".
type ClaimResponse struct {
	Code      string    `json:"code"`
	Command   string    `json:"command"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DevRedeemRequest simulates the bot seeing "@botty link <code>" in a group.
// TODO: change - delete once the ingest worker handles real messages.
type DevRedeemRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (r *DevRedeemRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Code == "" {
		return errors.New("code is required")
	}
	if r.Name == "" {
		r.Name = "Test group"
	}
	return nil
}
