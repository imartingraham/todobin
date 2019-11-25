package route

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/imartingraham/todobin/internal/util"
)

// Message that will be sent to client
type Message struct {
	Event string `json:"event"`
	Data  struct {
		ListID string `json:"list_id"`
		TodoID string `json:"todo_id"`
		Done   bool   `json:"done"`
	} `json:"data"`
}

// RegisterMessage registers client
type RegisterMessage struct {
	Message Message
	Client  *websocket.Conn
}

type socketManager struct {
	clients   map[string][]*websocket.Conn
	register  chan RegisterMessage
	broadcast chan Message
	m         sync.Mutex
}

var manager = socketManager{
	clients:   make(map[string][]*websocket.Conn),
	register:  make(chan RegisterMessage), // registration channel
	broadcast: make(chan Message),         // broadcast channel
}

var upgrader = websocket.Upgrader{}

// HandleWs handles websocket connections
func HandleWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		util.Airbrake.Notify(err, r)
		log.Fatal(err)
	}
	defer ws.Close()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)

		if err != nil {
			util.Airbrake.Notify(err, r)
			break
		}

		if msg.Event == "register" {
			registerMsg := RegisterMessage{
				Message: msg,
				Client:  ws,
			}
			manager.register <- registerMsg
		} else {
			manager.broadcast <- msg
		}
	}
}

// ListenForWebsocketMessages processes WS messages and sends them to the client
func ListenForWebsocketMessages() {
	for {
		select {
		case msg := <-manager.broadcast:
			err := manager.handleMessage(&msg)
			if err != nil {
				util.Airbrake.Notify(fmt.Errorf("handleMessage failed: %w", err), nil)
			}
		case reg := <-manager.register:
			err := manager.registerClient(&reg)
			if err != nil {
				util.Airbrake.Notify(fmt.Errorf("registerClient failed: %w", err), nil)
			}
		}
	}
}

func filterConn(conns []*websocket.Conn, c *websocket.Conn) (int, *websocket.Conn) {
	for i, curconn := range conns {
		if curconn == c {
			return i, curconn
		}
	}
	return -1, nil
}
func (m *socketManager) removeClient(listID string, c *websocket.Conn) {
	listClients := m.clients[listID]
	m.m.Lock()
	defer m.m.Unlock()
	i, _ := filterConn(listClients, c)
	if c != nil {
		m.clients[listID] = append(m.clients[listID][:i], m.clients[listID][i+1:]...)
	}
}

func (m *socketManager) todoDone(client *websocket.Conn, msg *Message) error {
	if err := client.WriteJSON(msg); err != nil {
		defer client.Close()
		m.removeClient(msg.Data.ListID, client)
		return err
	}
	return nil
}
func (m *socketManager) todoDelete(client *websocket.Conn, msg *Message) error {
	if err := client.WriteJSON(msg); err != nil {
		defer client.Close()
		m.removeClient(msg.Data.ListID, client)
		return err
	}
	return nil
}

func (m *socketManager) handleMessage(msg *Message) error {
	listClients := m.clients[msg.Data.ListID]
	for _, client := range listClients {
		switch msg.Event {
		case "todo:done":
			if err := m.todoDone(client, msg); err != nil {
				return fmt.Errorf("todoDone failed: %w", err)
			}
			break
		case "todo:delete":
			if err := m.todoDelete(client, msg); err != nil {
				return fmt.Errorf("todoDelete failed: %w", err)
			}
			break
		}
		
	}
	return nil
}

func (m *socketManager) registerClient(registerMsg *RegisterMessage) error {
	msg := registerMsg.Message
	client := registerMsg.Client
	log.Printf("Registering client for list: %s\n", msg.Data.ListID)
	m.m.Lock()
	defer m.m.Unlock()

	m.clients[msg.Data.ListID] = append(m.clients[msg.Data.ListID], client)
	msg.Event = "register:success"
	if err := client.WriteJSON(msg); err != nil {
		return fmt.Errorf("registerClient failed: %w", err)
	}
	return nil
}
