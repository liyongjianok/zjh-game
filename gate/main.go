// gate/main.go
package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 定义高并发下的读写超时与缓冲区大小
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 // 游戏指令通常很小，限制包体防内存耗尽
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 生产环境需严格校验 Origin
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 封装单一玩家的 WebSocket 连接
type Client struct {
	id   string
	conn *websocket.Conn
	send chan []byte
}

// ConnectionManager 管理所有网关层的长连接
type ConnectionManager struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 启动连接管理器的事件循环
func (manager *ConnectionManager) Run() {
	for {
		select {
		case client := <-manager.register:
			manager.mutex.Lock()
			manager.clients[client] = true
			manager.mutex.Unlock()
			log.Printf("Client %s connected. Total clients: %d", client.id, len(manager.clients))

		case client := <-manager.unregister:
			manager.mutex.Lock()
			if _, ok := manager.clients[client]; ok {
				delete(manager.clients, client)
				close(client.send)
			}
			manager.mutex.Unlock()
			log.Printf("Client %s disconnected.", client.id)
		}
	}
}

// readPump 处理客户端上行消息
func (c *Client) readPump(manager *ConnectionManager) {
	defer func() {
		manager.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		// TODO: 在此处解析 Protobuf 消息，并转发至 Connector/Game 服务
		log.Printf("Received from %s: %d bytes", c.id, len(message))
	}
}

// writePump 处理下发给客户端的消息（包含心跳）
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			w.Write(message)
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

// serveWs 处理外部的 WebSocket 升级请求
func serveWs(manager *ConnectionManager, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// 实际业务中应从 JWT 或 Token 提取真实 UserID
	mockUserID := "user_" + time.Now().Format("150405")

	client := &Client{id: mockUserID, conn: conn, send: make(chan []byte, 256)}
	manager.register <- client

	// 开启双向协程，遵循 Go 并发通信最佳实践
	go client.writePump()
	go client.readPump(manager)
}

func main() {
	manager := NewConnectionManager()
	go manager.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(manager, w, r)
	})

	log.Println("Gate Service starting on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
