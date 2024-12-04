// Copyright 2017 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.
package run

import (
	"bytes"
	"net/http"
	"time"

	beeLogger "github.com/beego/bee/v2/logger"
	"github.com/gorilla/websocket"
)

// wsBroker maintains the set of active clients and broadcasts messages to the clients.
type wsBroker struct {
	clients    map[*wsClient]bool // Registered clients.
	broadcast  chan []byte        // Inbound messages from the clients.
	register   chan *wsClient     // Register requests from the clients.
	unregister chan *wsClient     // Unregister requests from clients.
}

func (br *wsBroker) run() {
	for {
		select {
		case client := <-br.register:
			br.clients[client] = true
		case client := <-br.unregister:
			if _, ok := br.clients[client]; ok {
				delete(br.clients, client)
				close(client.send)
			}
		case message := <-br.broadcast:
			for client := range br.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(br.clients, client)
				}
			}
		}
	}
}

// wsClient represents the end-client.
type wsClient struct {
	broker *wsBroker       // The broker.
	conn   *websocket.Conn // The websocket connection.
	send   chan []byte     // Buffered channel of outbound messages.
}

// readPump pumps messages from the websocket connection to the broker.
func (c *wsClient) readPump() {
	defer func() {
		c.broker.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				beeLogger.Log.Errorf("An error happened when reading from the Websocket client: %v", err)
			}
			break
		}
	}
}

// write writes a message with the given message type and payload.
func (c *wsClient) write(mt int, payload []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(mt, payload)
}

// writePump pumps messages from the broker to the websocket connection.
func (c *wsClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The broker closed the channel.
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("/n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

var (
	broker        *wsBroker  // The broker.
	reloadAddress = ":12450" // The port on which the reload server will listen to.

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

const (
	writeWait  = 10 * time.Second    // Time allowed to write a message to the peer.
	pongWait   = 60 * time.Second    // Time allowed to read the next pong message from the peer.
	pingPeriod = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
)

// 用于启动一个 WebSocket 服务器，用来处理文件的实时重新加载
func startReloadServer() {
	// wsBroker 是一个结构体，用来管理 WebSocket 客户端的连接以及广播消息
	broker = &wsBroker{
		broadcast:  make(chan []byte),        // 用于向所有连接的客户端发送数据
		register:   make(chan *wsClient),     // 用于注册新的客户端连接
		unregister: make(chan *wsClient),     // 用于注销客户端连接
		clients:    make(map[*wsClient]bool), // 存储所有活跃的 WebSocket 客户端
	}

	// run 方法应该是 wsBroker 结构体的一个方法，用来处理客户端的注册、注销和消息广播等逻辑
	go broker.run()
	// 使用 http.HandleFunc 设置 /reload 路径的 HTTP 请求处理函数
	http.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		// 将 WebSocket 请求处理逻辑封装起来，通常会进行 WebSocket 握手，建立 WebSocket 连接，并将连接信息传递给 broker
		handleWsRequest(broker, w, r)
	})

	go startServer() // 启动 HTTP 服务器
	beeLogger.Log.Infof("Reload server listening at %s", reloadAddress)
}

func startServer() {
	err := http.ListenAndServe(reloadAddress, nil)
	if err != nil {
		beeLogger.Log.Errorf("Failed to start up the Reload server: %v", err)
		return
	}
}

func sendReload(payload string) {
	message := bytes.TrimSpace([]byte(payload))
	broker.broadcast <- message
}

// handleWsRequest handles websocket requests from the peer.
// 它用于处理来自客户端的 WebSocket 请求，升级 HTTP 连接为 WebSocket 连接，并启动相应的读取和写入处理
func handleWsRequest(broker *wsBroker, w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 请求为 WebSocket 请求
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		beeLogger.Log.Errorf("error while upgrading server connection: %v", err)
		return
	}

	// 创建一个新的 wsClient 实例，表示一个 WebSocket 客户端
	client := &wsClient{
		broker: broker,                 // 客户端所在的 wsBroker，用于管理客户端和消息的广播
		conn:   conn,                   // WebSocket 连接，通过 conn 客户端与服务器进行通信
		send:   make(chan []byte, 256), // 一个缓冲通道，用于发送数据到客户端
	}
	// 将 client 注册到 broker 中，发送到 broker.register 通道
	client.broker.register <- client

	// 启动一个 goroutine 来处理客户端的消息发送。这通常意味着 writePump 方法会负责持续监听 send 通道，并将数据发送到客户端
	go client.writePump()
	// 执行 readPump 方法，通常用于读取客户端发送的数据并进行相应的处理。这个方法是阻塞的，直到 WebSocket 连接关闭
	client.readPump()
}
