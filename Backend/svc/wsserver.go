package svc

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"tmaxsrv/log"
	"tmaxsrv/prnfmt"
)

const (
	// Time allowed to write a message to the peer.
	WRITE_WAIT = 10 * time.Second
	// Maximum message size allowed from peer.
	MAX_MSG_SZ = 8192
	// Time allowed to read the next pong message from the peer.
	PONG_WAIT = 60 * time.Second
)

type WsServer struct {
	listener net.Listener
	addr     string
	upgrade  *websocket.Upgrader
}

func NewWsServer() *WsServer {
	ws := new(WsServer)
	ws.addr = "0.0.0.0:7878"
	ws.upgrade = &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if r.Method != "GET" {
				log.Log.Debug("method is not GET")
				return false
			}
			if r.URL.Path != "/tmax" {
				log.Log.Debug("path error")
				return false
			}
			return true
		},
	}
	return ws
}

func (s *WsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/print/online" {
		prnfmt.HandlePrintOnline(w, r)
		return
	}

	if r.URL.Path != "/tmax" {
		httpCode := http.StatusInternalServerError
		reasePhrase := http.StatusText(httpCode)
		log.Log.Warnf("path error %v", reasePhrase)
		log.Log.Warnf("path error,url=%v", r.URL.Path)
		http.Error(w, reasePhrase, httpCode)
		return
	}

	var rqScaleId int64
	var err error
	if len(r.URL.RawQuery) == 0 { // client registers for information retrieving
		log.Log.Warn("bad query\n")
		return
	} else {
		qs := strings.Split(r.URL.RawQuery, "=")
		if len(qs) != 2 || qs[0] != "scaleid" {
			log.Log.Warn("bad query\n")
			return
		} else {
			rqScaleId, err = strconv.ParseInt(strings.TrimSpace(qs[1]), 10, 64)
			if err != nil {
				log.Log.Error(fmt.Sprintf("convert http request scale id string to int64 failed: %v", err))
			}
			log.Log.Infof("ScaleId:%v\n", rqScaleId)
		}
	}

	wsconn, err := s.upgrade.Upgrade(w, r, nil)
	if err != nil {
		log.Log.Warnf("websocket error: %v", err)
		return
	}

	log.Log.Infof("client connect : %v", wsconn.RemoteAddr())
	c := &Client{
		srvMgr: mSrvMgr, conn: wsconn, sendCh: make(chan []byte, CLIENT_SEND_CH_SIZE),
		recvCh: make(chan []byte, CLIENT_RECV_CH_SIZE), scaleId: rqScaleId,
	}
	c.ID = GenUserId()
	c.Addr = wsconn.RemoteAddr().String()
	c.EnterAt = time.Now()

	mSrvMgr.register <- c
	// defer func() { mHub.unregister <- c }()
	go c.writePump()
	go c.readPump()
}

var mSrvMgr *SrvMgr

func (s *WsServer) Start(srvMgr *SrvMgr) (err error) {
	mSrvMgr = srvMgr
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		log.Log.Error("net listen error:", err)
		return
	}

	err = http.Serve(s.listener, s)
	if err != nil {
		log.Log.Error("http serve error:", err)
		return
	}

	return
}
