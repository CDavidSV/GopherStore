package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/CDavidSV/GopherStore/internal/resp"
	"github.com/CDavidSV/GopherStore/internal/util"
)

type Message struct {
	cmd    Command
	client *Client
}

type Server struct {
	logger  *slog.Logger
	host    *url.URL
	ln      net.Listener
	wg      sync.WaitGroup
	regCh   chan *Client
	deregCh chan *Client
	clients map[*Client]struct{}
	msgCh   chan Message
	quitCh  chan struct{}
	store   KVStore
}

// Creates a new server instance.
func NewServer(logger *slog.Logger, hostName string, store KVStore) *Server {
	urlVal := fmt.Sprintf("tcp://%s", hostName)
	parsedHost, err := url.Parse(urlVal)
	if err != nil {
		logger.Error("failed to parse host URL", "url", urlVal, "error", err)
		return nil
	}

	return &Server{
		logger:  logger,
		host:    parsedHost,
		regCh:   make(chan *Client),
		deregCh: make(chan *Client),
		msgCh:   make(chan Message),
		quitCh:  make(chan struct{}),
		clients: make(map[*Client]struct{}),
		store:   store,
	}
}

// Starts the server and begins listening for incoming connections.
func (s *Server) Start() error {
	listener, err := net.Listen(s.host.Scheme, s.host.Host)
	if err != nil {
		return err
	}
	s.ln = listener

	s.wg.Add(2)
	go s.serverLoop()
	go s.acceptLoop()

	s.logger.Info("server started", "host", s.host.String())

	// Wait for interrupt signal to stop the server.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	s.logger.Info("Shutting down server...")
	close(s.quitCh)
	s.wg.Wait()

	s.logger.Info("Server stopped")
	return nil
}

// Adds a new connected client to the server's client map.
func (s *Server) registerClient(client *Client) {
	s.logger.Info("new client connected", "remoteAddr", client.conn.RemoteAddr().String())
	s.clients[client] = struct{}{}
}

// Removes a client from the server's client map.
func (s *Server) deregisterClient(client *Client) {
	client.conn.Close()
	s.logger.Info("client disconnected", "remoteAddr", client.conn.RemoteAddr().String())
	delete(s.clients, client)
}

// Responds to a PING command from a client.
func (s *Server) handlePingCommand(cmd PingCommand, client *Client) {
	response := "PONG"
	if cmd.Value != "" {
		response = cmd.Value
	}
	if err := client.SendMessage(resp.EncodeSimpleString(response)); err != nil {
		s.logger.Error("failed to send PING response", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
	}
}

// Handles a SET command from a client.
func (s *Server) handleSetCommand(cmd SetCommand, client *Client) {
	value, err := s.store.GetValue(cmd.Key)
	if err != nil {
		s.logger.Error("failed to handle SET command", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		client.SendMessage(resp.EncodeError(err.Error()))
		return
	}

	if cmd.condition == ConditionNX && value != nil {
		// Key exists, do not set
		client.SendMessage(resp.EncodeBulkString(nil))
		return
	}

	if cmd.condition == ConditionXX && value == nil {
		// Key does not exist, do not set
		client.SendMessage(resp.EncodeSimpleString("OK"))
		return
	}

	var expiresAt int64 = -1
	if cmd.expiration != nil {
		expTime := time.Now().Add(*cmd.expiration)
		expiresAt = expTime.UnixNano()
	}

	if expiresAt != 0 {
		// Set the key-value pair
		s.store.Set(cmd.Key, cmd.Value, expiresAt)
	}

	// Reply with OK
	if err := client.SendMessage(resp.EncodeSimpleString("OK")); err != nil {
		s.logger.Error("failed to send SET response", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
	}
}

// Handles a GET command from a client.
func (s *Server) handleGetCommand(cmd GetCommand, client *Client) {
	value, err := s.store.GetValue(cmd.Key)
	if err != nil {
		s.logger.Error("failed to handle GET command", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		client.SendMessage(resp.EncodeError(err.Error()))
		return
	}

	if value == nil {
		// Reply with nil bulk string
		if err := client.SendMessage(resp.EncodeBulkString(nil)); err != nil {
			s.logger.Error("failed to send GET response", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		}
		return
	}

	// Send value as a bulk string to the client
	if err := client.SendMessage(resp.EncodeBulkString(value)); err != nil {
		s.logger.Error("failed to send GET response", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
	}
}

func (s *Server) handleDeleteCommand(cmd DeleteCommand, client *Client) {
	deleted := s.store.Delete(cmd.Keys)

	client.SendMessage(resp.EncodeInteger(deleted))
}

func (s *Server) handleExistsCommand(cmd ExistsCommand, client *Client) {
	existing := s.store.Exists(cmd.Keys)

	client.SendMessage(resp.EncodeInteger(existing))
}

func (s *Server) handleExpireCommand(cmd ExpireCommand, client *Client) {
	expiresAt := time.Now().Add(cmd.TTL).UnixNano()
	success := s.store.Expire(cmd.Key, expiresAt)

	// Reply with integer 1 if successful, 0 otherwise.
	if success {
		client.SendMessage(resp.EncodeInteger(1))
	} else {
		client.SendMessage(resp.EncodeInteger(0))
	}
}

func (s *Server) handlePushCommand(cmd PushCommand, client *Client) {
	newLen, err := s.store.Push(cmd.Key, cmd.Vals, cmd.pushAtFront)
	if err != nil {
		s.logger.Error("failed to handle PUSH command", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		client.SendMessage(resp.EncodeError(err.Error()))
		return
	}

	client.SendMessage(resp.EncodeInteger(int64(newLen)))
}

func (s *Server) handlePopCommand(cmd PopCommand, client *Client) {
	value, err := s.store.Pop(cmd.Key, cmd.popAtFront)
	if err != nil {
		s.logger.Error("failed to handle POP command", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		client.SendMessage(resp.EncodeError(err.Error()))
		return
	}

	if value == nil {
		client.SendMessage(resp.EncodeBulkString(nil))
	} else {
		client.SendMessage(resp.EncodeBulkString(value))
	}
}

func (s *Server) handleLLenCommand(cmd LLenCommand, client *Client) {
	list, err := s.store.GetList(cmd.Key)
	if err != nil {
		s.logger.Error("failed to handle LLEN command", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		client.SendMessage(resp.EncodeError(err.Error()))
		return
	}

	if list == nil {
		client.SendMessage(resp.EncodeInteger(0))
		return
	}

	client.SendMessage(resp.EncodeInteger(int64(len(list))))
}

func (s *Server) handleLRangeCommand(cmd LRangeCommand, client *Client) {
	list, err := s.store.GetList(cmd.Key)
	if err != nil {
		s.logger.Error("failed to handle LRANGE command", "error", err, "remoteAddr", client.conn.RemoteAddr().String())
		client.SendMessage(resp.EncodeError(err.Error()))
		return
	}

	if list == nil {
		client.SendMessage(resp.EncodeBulkStringArray(nil))
		return
	}

	// Slice list and send to client
	slicedList := util.SliceList(list, cmd.Start, cmd.End)
	client.SendMessage(resp.EncodeBulkStringArray(slicedList))
}

func (s *Server) handleMessage(msg Message) {
	switch cmd := msg.cmd.(type) {
	case PingCommand:
		s.handlePingCommand(cmd, msg.client)
	case SetCommand:
		s.handleSetCommand(cmd, msg.client)
	case GetCommand:
		s.handleGetCommand(cmd, msg.client)
	case DeleteCommand:
		s.handleDeleteCommand(cmd, msg.client)
	case ExistsCommand:
		s.handleExistsCommand(cmd, msg.client)
	case ExpireCommand:
		s.handleExpireCommand(cmd, msg.client)
	case PushCommand:
		s.handlePushCommand(cmd, msg.client)
	case PopCommand:
		s.handlePopCommand(cmd, msg.client)
	case LLenCommand:
		s.handleLLenCommand(cmd, msg.client)
	case LRangeCommand:
		s.handleLRangeCommand(cmd, msg.client)
	}
}

// Main server loop that handles clients and commands.
func (s *Server) serverLoop() {
	defer s.wg.Done()

	for {
		select {
		case client := <-s.regCh:
			s.registerClient(client)
		case client := <-s.deregCh:
			s.deregisterClient(client)
		case msg := <-s.msgCh:
			s.handleMessage(msg)
		case <-s.quitCh:
			// Shutdown the server
			s.store.Close()
			for client := range s.clients {
				s.deregisterClient(client)
			}
			s.ln.Close()
			return
		}
	}
}

// Accepts incomming connections and registers new clients.
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return // Listener closed, exit the loop
			}

			s.logger.Error("failed to accept connection", "error", err)
			continue
		}

		// Connection accepted
		go s.handleNewClient(conn)
	}
}

// Handles registering a new client to the server and starts its reader loop.
func (s *Server) handleNewClient(conn net.Conn) {
	client := NewClient(conn, s.deregCh, s.msgCh, s.logger)
	s.regCh <- client

	go client.write()
	if err := client.read(); err != nil {
		s.logger.Error("client read error", "error", err)
	}
}
