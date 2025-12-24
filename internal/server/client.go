package server

import (
	"bufio"
	"io"
	"log/slog"
	"net"

	"github.com/CDavidSV/GopherStore/internal/resp"
)

type Client struct {
	conn    net.Conn
	deregCh chan *Client
	msgCh   chan Message
}

func NewClient(conn net.Conn, deregCh chan *Client, msgCh chan Message) *Client {
	return &Client{
		conn:    conn,
		deregCh: deregCh,
		msgCh:   msgCh,
	}
}

func (c *Client) reader() error {
	defer func() {
		c.conn.Close()
		c.deregCh <- c
	}()

	reader := bufio.NewReader(c.conn)

	for {
		v, err := resp.ReadRESP(reader)
		if err != nil {
			// error could be EOF or a RESP parsing error
			if err == io.EOF {
				return nil
			} else if respErr, ok := err.(*resp.RESPError); ok {
				slog.Debug("RESP error while reading from client", "error", respErr.Msg)
				continue
			}

			// If none of the above, we handle it as an unexpected error and deregister the client.
			slog.Error("unexpected error while reading from client", "error", err)
			c.deregCh <- c
			return err
		}

		// Depending on the type, we handle commands accordingly.
		cmd, ok := v.(resp.RespArray)
		if !ok || len(cmd.Elements) == 0 {
			slog.Debug("received non-array or empty command from client")
			continue
		}

		// Process the command
		parsedCmd, err := ParseCommand(cmd)
		if err != nil {
			slog.Warn("failed to parse command from client", "error", err)
			continue
		}

		c.msgCh <- Message{
			cmd:    parsedCmd,
			client: c,
		}
	}
}
