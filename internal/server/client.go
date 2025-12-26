package server

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/CDavidSV/GopherStore/internal/resp"
)

type Client struct {
	conn    net.Conn
	deregCh chan *Client
	msgCh   chan Message
	sendCh  chan []byte
	doneCh  chan struct{}
	writer  *bufio.Writer
	logger  *slog.Logger
}

func NewClient(conn net.Conn, deregCh chan *Client, msgCh chan Message, logger *slog.Logger) *Client {
	return &Client{
		conn:    conn,
		deregCh: deregCh,
		msgCh:   msgCh,
		sendCh:  make(chan []byte, 1024),
		doneCh:  make(chan struct{}),
		writer:  bufio.NewWriter(conn),
		logger:  logger,
	}
}

func (c *Client) SendMessage(msg []byte) error {
	select {
	case c.sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("send channel full")
	}
}

func (c *Client) read() error {
	defer func() {
		// Close the send channel to signal write() to stop
		close(c.doneCh)
	}()

	reader := bufio.NewReader(c.conn)

	for {
		v, err := resp.ReadRESP(reader)
		if err != nil {
			// error could be EOF or a RESP parsing error
			if err == io.EOF {
				return nil
			} else if respErr, ok := err.(*resp.RESPError); ok {
				c.logger.Debug("RESP error while reading from client", "error", respErr.Msg)
				c.SendMessage(resp.EncodeError(respErr.Error()))
				return nil
			}

			// If none of the above, we handle it as an unexpected error and deregister the client.
			c.SendMessage(resp.EncodeError("Internal server error"))
			return err
		}

		// Depending on the type, we handle commands accordingly.
		cmd, ok := v.(resp.RespArray)
		if !ok {
			c.logger.Debug("received non-array from client")
			c.SendMessage(resp.EncodeError("expected array of commands"))
			return nil
		}

		if len(cmd.Elements) == 0 {
			c.logger.Debug("received empty command array from client")
			c.SendMessage(resp.EncodeError("empty command array"))
			return nil
		}

		// Process the command
		parsedCmd, err := ParseCommand(cmd)
		if err != nil {
			c.logger.Debug("failed to parse command from client", "error", err)
			c.SendMessage(resp.EncodeError(err.Error()))
			continue
		}

		c.msgCh <- Message{
			cmd:    parsedCmd,
			client: c,
		}
	}
}

func (c *Client) write() {
	defer func() {
		c.writer.Flush()
		c.deregCh <- c
	}()

	for {
		select {
		case msg := <-c.sendCh:
			if _, err := c.writer.Write(msg); err != nil {
				c.logger.Error("failed to write to client", "error", err)
				return
			}

			if err := c.writer.Flush(); err != nil {
				c.logger.Error("failed to flush writer to client", "error", err)
				return
			}
		case <-c.doneCh:
			return
		}
	}
}
