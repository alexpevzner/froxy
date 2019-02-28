//
// websocket
//

package main

import (
	"io"
	"net/http"
	"tproxy/log"

	"github.com/gorilla/websocket"
)

//
// The websocket
//
type websock struct {
	*websocket.Conn
	r io.Reader
}

var _ = io.Reader(&websock{})
var _ = io.Writer(&websock{})

//
// Establish websocket connection in a client mode
//
func websockDial(url string, hdr http.Header) (*websock, *http.Response, error) {
	conn, resp, err := websocket.DefaultDialer.Dial(url, hdr)

	var ws *websock
	if conn != nil {
		ws = &websock{
			Conn: conn,
		}
	}

	return ws, resp, err
}

//
// Upgrate HTTP server connection to the websocket
//
func websockUpgrade(w http.ResponseWriter, r *http.Request) (*websock, error) {
	conn, err := websocket.Upgrade(w, r, nil, 65536, 16384)
	if err != nil {
		return nil, err
	}

	return &websock{Conn: conn}, nil
}

//
// Dead data from websocket in a byte-stream mode
//
func (ws *websock) Read(buf []byte) (l int, err error) {
	log.Debug("ws R len(buf) = %d", len(buf))
	if len(buf) == 0 {
		return 0, nil
	}

	for l == 0 && err == nil {
		for ws.r == nil {
			var t int
			t, ws.r, err = ws.Conn.NextReader()
			if err != nil {
				log.Debug("ws R %s", err)
				return 0, err
			}
			if t != websocket.BinaryMessage {
				log.Debug("ws R skip %d", t)
				ws.r = nil
			}
		}

		l, err = ws.r.Read(buf)
		if err == io.EOF {
			ws.r = nil
			err = nil
		}
	}

	log.Debug("ws R %d %s", l, err)
	return
}

//
// Write data to websocket in a byte-stream mode
//
func (ws *websock) Write(buf []byte) (int, error) {
	err := ws.Conn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		return 0, err
	} else {
		log.Debug("ws W %d", len(buf))

		return len(buf), nil
	}
}
