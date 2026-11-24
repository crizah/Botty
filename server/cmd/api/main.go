package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"server/internal/config"
	"server/internal/handlers"
	"server/internal/ingest"
	"server/internal/mail"
	"server/internal/middleware"
	"server/internal/sarvam"
	"server/internal/services"
	"server/internal/wa"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Fatal(err)
	}
	// foreign keys on, WAL for concurrent reads, immediate write transactions to avoid SQLITE_BUSY upgrades
	dsn := "file:" + cfg.DBPath + "?_txlock=immediate&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Ping(); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.WAEnabled {
		keeper := wa.NewKeeper(services.NewGroupService(conn, cfg.BotNumber, cfg.DevMode))
		chain := wa.Chain{wa.NewLinkHandler(services.NewClaimService(conn), keeper)}
		if cfg.SarvamKey != "" {
			chain = append(chain, &wa.TagHandler{
				Store:     services.NewTagService(conn),
				AI:        sarvam.New(cfg.SarvamKey, cfg.SarvamURL, cfg.NamingModel),
				Threshold: cfg.MatchThreshold,
				MaxTasks:  cfg.TagMaxTasks,
			})
		} else {
			log.Println("[wa] SARVAM_API_KEY not set - tagging the bot to make/close tasks is disabled")
		}
		chain = append(chain, keeper)
		waClient := wa.New(cfg.WASessionDB, chain, keeper, cfg.WAWorkers)
		go func() {
			if err := waClient.Run(ctx); err != nil {
				log.Printf("[wa] stopped: %v", err)
			}
		}()
	} else {
		// stub: WhatsApp group discovery + identity mapping (used when WA_ENABLED is off)
		go (&ingest.Worker{Resolver: ingest.StubResolver{}}).Run(ctx)
	}

	handler := middleware.Logging(middleware.CORS(cfg.AllowedOrigin)(routes(conn, cfg)))
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("botty api listening on :%s (allowed origins: %v)", cfg.Port, cfg.AllowedOrigin)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// routes builds the services and handlers and wires every endpoint.
func routes(conn *sql.DB, cfg config.Config) http.Handler {
	// services
	authSvc := services.NewAuthService(conn, cfg.JWTSecret)
	projectSvc := services.NewProjectService(conn)
	groupSvc := services.NewGroupService(conn, cfg.BotNumber, cfg.DevMode)
	claimSvc := services.NewClaimService(conn)
	taskSvc := services.NewTaskService(conn)

	// handlers
	authH := handlers.NewAuthHandler(authSvc, cfg.CookieSecure)
	projectH := handlers.NewProjectHandler(projectSvc)
	groupH := handlers.NewGroupHandler(groupSvc)
	mailer := mail.NewResend(cfg.ResendKey, cfg.MailFrom)
	memberH := handlers.NewMemberHandler(services.NewMemberService(conn, cfg.JWTSecret, mailer, cfg.FrontendURL))
	claimH := handlers.NewClaimHandler(claimSvc, cfg.DevMode)
	taskH := handlers.NewTaskHandler(taskSvc)

	mux := http.NewServeMux()
	requireAuth := middleware.RequireAuth(cfg.JWTSecret)
	protected := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, requireAuth(h)) }

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// auth
	mux.HandleFunc("POST /api/v1/auth/signup", authH.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", authH.Login)
	mux.HandleFunc("POST /api/v1/auth/logout", authH.Logout)
	protected("GET /api/v1/auth/me", authH.Me)

	// bot + the groups it has been added to
	protected("GET /api/v1/bot", groupH.BotInfo)
	protected("GET /api/v1/groups", groupH.ListMine)
	protected("POST /api/v1/groups/claims", claimH.Create)
	if cfg.DevMode {
		// TODO: change - dev only, remove once the ingest worker discovers real groups
		protected("POST /api/v1/dev/groups", groupH.DevCreate)
		protected("POST /api/v1/dev/groups/link", claimH.DevRedeem)
	}

	// projects
	protected("GET /api/v1/projects", projectH.List)
	protected("POST /api/v1/projects", projectH.Create)
	protected("GET /api/v1/projects/{id}", projectH.Get)
	protected("PATCH /api/v1/projects/{id}", projectH.Update)
	protected("GET /api/v1/projects/{id}/audit", projectH.Audit)

	// project users + email invites (owner-only changes)
	protected("GET /api/v1/projects/{id}/users", memberH.List)
	protected("PATCH /api/v1/projects/{id}/users/{uid}", memberH.SetRole)
	protected("DELETE /api/v1/projects/{id}/users/{uid}", memberH.Remove)
	protected("POST /api/v1/projects/{id}/invites", memberH.Invite)
	protected("POST /api/v1/projects/{id}/invites/{iid}/resend", memberH.Resend)
	protected("DELETE /api/v1/projects/{id}/invites/{iid}", memberH.Revoke)
	mux.HandleFunc("GET /api/v1/invites/preview", memberH.Preview) // public: the emailed token is the credential
	protected("POST /api/v1/invites/accept", memberH.Accept)

	// groups inside a project + tag allowlist
	protected("GET /api/v1/projects/{id}/groups", groupH.ListForProject)
	protected("POST /api/v1/projects/{id}/groups", groupH.Attach)
	protected("DELETE /api/v1/projects/{id}/groups/{gid}", groupH.Detach)
	protected("GET /api/v1/projects/{id}/groups/{gid}/members", groupH.Members)
	protected("GET /api/v1/projects/{id}/groups/{gid}/allowlist", groupH.Allowlist)
	protected("POST /api/v1/projects/{id}/groups/{gid}/allowlist", groupH.AllowlistAdd)
	protected("DELETE /api/v1/projects/{id}/groups/{gid}/allowlist/{mid}", groupH.AllowlistRemove)

	// tasks, reports, comments
	protected("GET /api/v1/projects/{id}/tasks", taskH.List)
	protected("POST /api/v1/projects/{id}/tasks", taskH.Create)
	protected("GET /api/v1/tasks/{id}", taskH.Get)
	protected("PATCH /api/v1/tasks/{id}", taskH.Update)
	protected("PATCH /api/v1/tasks/{id}/status", taskH.SetStatus)
	protected("POST /api/v1/tasks/{id}/reports", taskH.CreateReport)
	protected("POST /api/v1/tasks/{id}/comments", taskH.CommentOnTask)
	protected("PATCH /api/v1/reports/{id}/status", taskH.SetReportStatus)
	protected("POST /api/v1/reports/{id}/comments", taskH.CommentOnReport)

	return mux
}
