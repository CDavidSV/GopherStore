package server

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sync"
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
}

// Creates a new server instance.
func NewServer(logger *slog.Logger, hostName string) *Server {
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
		clients: make(map[*Client]struct{}),
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

	s.wg.Wait()
	return nil
}

func (s *Server) registerClient(client *Client) {
	s.logger.Info("new client connected", "remoteAddr", client.conn.RemoteAddr().String())
	s.clients[client] = struct{}{}
}

func (s *Server) deregisterClient(client *Client) {
	s.logger.Info("client disconnected", "remoteAddr", client.conn.RemoteAddr().String())
	delete(s.clients, client)
}

func (s *Server) handleMessage(msg Message) {
	switch cmd := msg.cmd.(type) {
	case SetCommand:
		s.logger.Info("handling SET command", "key", cmd.Key, "value", cmd.Value)
	case GetCommand:
		s.logger.Info("handling GET command", "key", cmd.Key)
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
		}
	}
}

// Accepts incomming connections and registers new clients.
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.logger.Error("failed to accept connection", "error", err)
			continue
		}

		// Connection accepted
		go s.handleNewClient(conn)
	}
}

// Handles registering a new client to the server and starts its reader loop.
func (s *Server) handleNewClient(conn net.Conn) {
	client := NewClient(conn, s.deregCh, s.msgCh)
	s.regCh <- client

	if err := client.reader(); err != nil {
		s.logger.Error("client read error", "error", err)
	}
}
