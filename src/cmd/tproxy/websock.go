//
// websocket
//

package main

import (
	"github.com/gorilla/websocket"
	"io"
	"net/http"
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
// Dead data from websocket in a byte-stream mode
//
func (ws *websock) Read(buf []byte) (l int, err error) {
	if len(buf) == 0 {
		return 0, nil
	}

	for l == 0 && err == nil {
		for ws.r == nil {
			var t int
			t, ws.r, err = ws.Conn.NextReader()
			if err != nil {
				return 0, err
			}
			if t != websocket.BinaryMessage {
				ws.r = nil
			}
		}
		return 0, nil

		l, err = ws.r.Read(buf)
	}

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
		return len(buf), nil
	}
}
