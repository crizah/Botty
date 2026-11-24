// Package mail sends transactional email through Resend.
package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

// Resend sends through https://resend.com.
type Resend struct {
	Key  string
	From string // e.g. "Botty <noreply@crizah.space>"
	HTTP *http.Client
}

func NewResend(key, from string) *Resend {
	return &Resend{Key: key, From: from, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (r *Resend) Send(ctx context.Context, m Message) error {
	body, _ := json.Marshal(map[string]any{
		"from": r.From, "to": []string{m.To}, "subject": m.Subject, "text": m.Text, "html": m.HTML,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.Key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend: %d %s", resp.StatusCode, raw)
	}
	return nil
}

// InviteEmail builds the "you've been invited" email.
func InviteEmail(to, inviter, project, roleLabel, link string) Message {
	text := fmt.Sprintf(`Hi,

%s invited you to join "%s" on Botty as %s.

Accept the invite:
%s

This link works once and expires in 48 hours.

- Botty`, inviter, project, roleLabel, link)

	htmlBody := fmt.Sprintf(`<div style="font-family:system-ui,sans-serif;max-width:480px;margin:0 auto;padding:24px;color:#000">
<p>Hi,</p>
<p><b>%s</b> invited you to join <b>%s</b> on Botty as <b>%s</b>.</p>
<p><a href="%s" style="display:inline-block;background:#000;color:#fff;padding:10px 18px;border-radius:8px;text-decoration:none">Accept invite</a></p>
<p style="color:#555;font-size:13px">This link works once and expires in 48 hours.</p>
</div>`, html.EscapeString(inviter), html.EscapeString(project), html.EscapeString(roleLabel), html.EscapeString(link))

	return Message{To: to, Subject: fmt.Sprintf("%s invited you to %s on Botty", inviter, project), Text: text, HTML: htmlBody}
}
