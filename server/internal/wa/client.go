// Package wa is the WhatsApp connection (whatsmeow): pairing, reconnects, and fan-out of
// incoming group messages to a fixed set of sharded workers.
package wa

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

// Status of the bot's WhatsApp session.
type Status string

const (
	StatusConnecting   Status = "connecting"
	StatusWaitingForQR Status = "waiting_for_qr"
	StatusConnected    Status = "connected"
	StatusLoggedOut    Status = "logged_out"
)

type Client struct {
	sessionDSN string
	router     *Router
	keeper     *Keeper
	status     atomic.Value // Status
}

// New builds a client. sessionDB is a SQLite file for whatsmeow's own session keys (not the app DB).
func New(sessionDB string, handler Handler, keeper *Keeper, workers int) *Client {
	c := &Client{
		sessionDSN: fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", sessionDB),
		router:     NewRouter(handler, workers),
		keeper:     keeper,
	}
	c.status.Store(StatusConnecting)
	return c
}

func (c *Client) Status() Status { return c.status.Load().(Status) }

// Run connects and stays connected until ctx is cancelled. If WhatsApp logs the bot out
// it goes straight back to pairing (new QR) without restarting the process.
func (c *Client) Run(ctx context.Context) error {
	sdb, err := sql.Open("sqlite", c.sessionDSN)
	if err != nil {
		return err
	}
	container := sqlstore.NewWithDB(sdb, "sqlite", waLog.Stdout("WA-DB", "WARN", true))
	if err := container.Upgrade(ctx); err != nil {
		return fmt.Errorf("session store: %w", err)
	}
	defer container.Close()

	c.router.Start(ctx)

	for ctx.Err() == nil {
		device, err := container.GetFirstDevice(ctx) // a fresh, unpaired device when there is none
		if err != nil {
			return err
		}
		loggedOut := c.session(ctx, device)
		if !loggedOut {
			return nil // ctx cancelled
		}
		c.status.Store(StatusLoggedOut)
		log.Printf("[wa] logged out by WhatsApp - starting a new pairing")
	}
	return nil
}

// session runs one connected lifetime of a device. It returns true if the device was logged out.
func (c *Client) session(ctx context.Context, device *store.Device) bool {
	cli := whatsmeow.NewClient(device, waLog.Stdout("WA", "INFO", true))
	logout := make(chan struct{}, 1)

	cli.AddEventHandler(func(evt any) {
		switch e := evt.(type) {
		case *events.Message:
			c.router.Dispatch(cli, e)
		case *events.GroupInfo:
			chat := e.JID
			c.router.Submit(chat.String(), func(ctx context.Context) {
				c.keeper.OnGroupInfo(ctx, e, func(ctx context.Context) ([]Participant, error) { return participants(ctx, cli, chat) })
			})
		case *events.Connected:
			c.status.Store(StatusConnected)
			log.Printf("[wa] connected as %s", cli.Store.ID)
		case *events.Disconnected:
			c.status.Store(StatusConnecting)
		case *events.LoggedOut:
			select {
			case logout <- struct{}{}:
			default:
			}
		}
	})

	c.status.Store(StatusConnecting)
	if cli.Store.ID == nil {
		qrCh, _ := cli.GetQRChannel(ctx)
		if err := cli.Connect(); err != nil {
			log.Printf("[wa] connect: %v", err)
			return false
		}
		go func() {
			for item := range qrCh {
				if item.Event == "code" {
					c.status.Store(StatusWaitingForQR)
					log.Printf("[wa] scan this QR: WhatsApp > Linked devices > Link a device")
					qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, log.Writer())
					// TODO: change - also expose the QR on an admin-only page so production can re-pair without terminal access
				} else {
					log.Printf("[wa] pairing: %s", item.Event)
				}
			}
		}()
	} else if err := cli.Connect(); err != nil {
		log.Printf("[wa] connect: %v", err)
		return false
	}

	select {
	case <-ctx.Done():
		cli.Disconnect()
		return false
	case <-logout:
		cli.Disconnect()
		return true
	}
}
