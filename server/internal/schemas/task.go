package schemas

import (
	"errors"
	"strings"

	db "server/internal/db/sqlc"
)

// --- Requests ---

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (r *CreateTaskRequest) Validate() error {
	r.Title, r.Description = strings.TrimSpace(r.Title), strings.TrimSpace(r.Description)
	if r.Title == "" || len(r.Title) > 200 {
		return errors.New("task title is required (max 200 characters)")
	}
	if len(r.Description) > 5000 {
		return errors.New("description is too long")
	}
	return nil
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func (r *UpdateTaskRequest) Validate() error {
	if r.Title != nil {
		s := strings.TrimSpace(*r.Title)
		if s == "" || len(s) > 200 {
			return errors.New("task title is required (max 200 characters)")
		}
		r.Title = &s
	}
	return nil
}

// StatusRequest is used for both tasks and reports.
type StatusRequest struct {
	Status string `json:"status"`
}

func (r *StatusRequest) Validate() error {
	if r.Status != "todo" && r.Status != "done" {
		return errors.New("status must be todo or done")
	}
	return nil
}

type CreateReportRequest struct {
	GroupID            string `json:"group_id"`
	ReporterMemberID   string `json:"reporter_member_id"`
	AssignedByMemberID string `json:"assigned_by_member_id"`
	MessageID          string `json:"message_id"`
	MessageText        string `json:"message_text"`
}

func (r *CreateReportRequest) Validate() error {
	if (r.ReporterMemberID != "" || r.AssignedByMemberID != "") && r.GroupID == "" {
		return errors.New("group_id is required when setting reporter or assigned_by")
	}
	if len(r.MessageText) > 5000 {
		return errors.New("message_text is too long")
	}
	return nil
}

type CommentRequest struct {
	Body string `json:"body"`
}

func (r *CommentRequest) Validate() error {
	r.Body = strings.TrimSpace(r.Body)
	if r.Body == "" || len(r.Body) > 5000 {
		return errors.New("comment must be 1-5000 characters")
	}
	return nil
}

// --- Responses ---

type TasksResponse struct {
	Tasks []db.ListTasksByProjectRow `json:"tasks"`
}

type TaskResponse struct {
	Task db.Task `json:"task"`
}

type TaskDetailResponse struct {
	Task    db.Task                   `json:"task"`
	Reports []db.ListReportsByTaskRow `json:"reports"`
	// Duplicates are repeats of a report's message by the same person; group by report_id.
	Duplicates []db.ReportDuplicate       `json:"duplicates"`
	Comments   []db.ListCommentsByTaskRow `json:"comments"`
	Audit      []db.ListAuditByTaskRow    `json:"audit"`
}

type ReportResponse struct {
	Report db.Report `json:"report"`
}

type ReportStatusResponse struct {
	Report     db.Report `json:"report"`
	TaskStatus string    `json:"task_status"`
}

type CommentResponse struct {
	Comment db.Comment `json:"comment"`
}
