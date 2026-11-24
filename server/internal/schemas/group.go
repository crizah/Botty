package schemas

import db "server/internal/db/sqlc"

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
