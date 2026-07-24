package svc

import (
	"bytes"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	l "tmaxsrv/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize      = 51200
	CLIENT_RECV_CH_SIZE = 256
	CLIENT_SEND_CH_SIZE = 256
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// var upgrader = websocket.Upgrader{
// 	ReadBufferSize:  1024000,
// 	WriteBufferSize: 1024000,
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true
// 	},
// }

type User struct {
	ID      string
	Addr    string
	EnterAt time.Time
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	srvMgr *SrvMgr
	// The websocket connection.
	conn *websocket.Conn
	// Buffered channel of outbound messages.
	sendCh chan []byte
	recvCh chan []byte
	// scale ID
	scaleId  int64
	wgSndCh  sync.WaitGroup
	wgRecvCh sync.WaitGroup
	isQuit   bool

	User
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	c.wgRecvCh.Add(1)
	defer func() {
		c.wgRecvCh.Done()
		c.srvMgr.unregister <- c
		// c.sendCh = nil
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	c.conn.SetPingHandler(func(message string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(writeWait))
		return nil
	})
	for {
		if c.isQuit {
			break
		}
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				l.Log.Errorf("error: %v", err)
			}
			break
		}
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		l.Log.Debugf("%v\n", string(message))
		data := map[string][]byte{
			"message": message,
			"id":      []byte(c.ID),
			"scaleId": big.NewInt(c.scaleId).Bytes(),
		}
		userMessage, _ := json.Marshal(data)
		if c.scaleId == 0 || c.scaleId > 999999900 { // for common wssocket to srvMgr
			select {
			case c.srvMgr.recvWsClientMsg <- userMessage:
			default:
				l.Log.Warnf("recvWsClientMsg full, dropped message")
			}
		} else { // for scale message
			select {
			case c.recvCh <- userMessage:
			default:
				l.Log.Warnf("recvCh full, dropped message")
			}
		}
		// log.Log.Debugf("got user message: %v", userMessage)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	c.isQuit = false
	c.wgSndCh.Add(1)
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.wgSndCh.Done()
	}()
	for {
		if c.isQuit {
			break
		}
		select {
		case message, ok := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				l.Log.Errorf("To client error: %v", err.Error())
				return
			}
			l.Log.Debugf("To client: %v", string(message))
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			// n := len(c.send)
			// for i := 0; i < n; i++ {
			// 	w.Write(newline)
			// 	w.Write(<-c.send)
			// }

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) Close() error {
	c.isQuit = true
	if c.sendCh != nil {
		c.wgSndCh.Wait()
		close(c.sendCh)
		c.sendCh = nil
	}
	if c.recvCh != nil {
		c.wgRecvCh.Wait()
		close(c.recvCh)
		c.recvCh = nil
	}

	return nil
}

func GenUserId() string {
	uid := uuid.NewString()
	return uid
}
