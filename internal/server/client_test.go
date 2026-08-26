package server

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/CDavidSV/GopherStore/internal/resp"
)

func newTestClient(t *testing.T) (*Client, net.Conn, chan *Client, chan Message) {
	t.Helper()

	clientConn, peerConn := net.Pipe()
	deregCh := make(chan *Client, 1)
	msgCh := make(chan Message, 1)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Cleanup(func() {
		clientConn.Close()
		peerConn.Close()
	})

	return NewClient(clientConn, deregCh, msgCh, logger), peerConn, deregCh, msgCh
}

func readClientResponse(t *testing.T, client *Client) resp.RespValue {
	t.Helper()

	select {
	case message := <-client.sendCh:
		value, err := resp.ReadRESP(bufio.NewReader(bytes.NewReader(message)))
		if err != nil {
			t.Fatalf("ReadRESP() unexpected error: %v", err)
		}
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for client response")
		return nil
	}
}
func encodeClientCommand(values ...string) []byte {
	elements := make([][]byte, len(values))
	for i, value := range values {
		elements[i] = []byte(value)
	}
	return resp.EncodeBulkStringArray(elements)
}

func runClientRead(t *testing.T, client *Client, peer net.Conn, input []byte) <-chan error {
	t.Helper()

	readDone := make(chan error, 1)
	go func() {
		readDone <- client.read()
	}()

	writeDone := make(chan error, 1)
	go func() {
		_, err := peer.Write(input)
		writeDone <- err
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("peer.Write() unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out writing client input")
	}

	return readDone
}

func TestClientReadForwardsCommand(t *testing.T) {
	client, peer, _, msgCh := newTestClient(t)
	readDone := runClientRead(t, client, peer, encodeClientCommand("PING", "hello"))

	select {
	case message := <-msgCh:
		command, ok := message.cmd.(PingCommand)
		if !ok {
			t.Fatalf("message command type = %T, want PingCommand", message.cmd)
		}
		if command.Value != "hello" {
			t.Fatalf("PING value = %q, want %q", command.Value, "hello")
		}
		if message.client != client {
			t.Fatal("message client does not reference the sending client")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for parsed command")
	}

	if err := peer.Close(); err != nil {
		t.Fatalf("peer.Close() unexpected error: %v", err)
	}
	if err := <-readDone; err != nil {
		t.Fatalf("client.read() unexpected error: %v", err)
	}
}

func TestClientReadRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{name: "invalid RESP prefix", input: []byte("invalid"), want: "unknown RESP type prefix: i"},
		{name: "non-array value", input: resp.EncodeSimpleString("OK"), want: "expected array of commands"},
		{name: "empty command array", input: []byte("*0\r\n"), want: "empty command array"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, peer, _, _ := newTestClient(t)
			readDone := runClientRead(t, client, peer, tt.input)

			if err := <-readDone; err != nil {
				t.Fatalf("client.read() unexpected error: %v", err)
			}
			got := readClientResponse(t, client)
			response, ok := got.(resp.RespErrorValue)
			if !ok {
				t.Fatalf("response type = %T, want RespErrorValue", got)
			}
			if response.Message != tt.want {
				t.Fatalf("response message = %q, want %q", response.Message, tt.want)
			}
		})
	}
}

func TestClientReadContinuesAfterCommandParseError(t *testing.T) {
	client, peer, _, msgCh := newTestClient(t)
	input := append(encodeClientCommand("GET"), encodeClientCommand("PING")...)
	readDone := runClientRead(t, client, peer, input)

	response := readClientResponse(t, client)
	if got, ok := response.(resp.RespErrorValue); !ok || got.Message != "GET command requires exactly 1 argument" {
		t.Fatalf("error response = %#v, want GET argument error", response)
	}

	select {
	case message := <-msgCh:
		if _, ok := message.cmd.(PingCommand); !ok {
			t.Fatalf("message command type = %T, want PingCommand", message.cmd)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for command after parse error")
	}

	if err := peer.Close(); err != nil {
		t.Fatalf("peer.Close() unexpected error: %v", err)
	}
	if err := <-readDone; err != nil {
		t.Fatalf("client.read() unexpected error: %v", err)
	}
}

func TestClientSendMessageWhenBufferIsFull(t *testing.T) {
	client, peer, _, _ := newTestClient(t)
	defer peer.Close()

	message := []byte("message")
	for i := 0; i < cap(client.sendCh); i++ {
		if err := client.SendMessage(message); err != nil {
			t.Fatalf("SendMessage() unexpected error at message %d: %v", i, err)
		}
	}

	if err := client.SendMessage(message); err == nil {
		t.Fatal("SendMessage() returned nil for a full send channel")
	}
}

func TestClientWriteFlushesAndDeregisters(t *testing.T) {
	client, peer, deregCh, _ := newTestClient(t)
	writeDone := make(chan struct{})
	go func() {
		client.write()
		close(writeDone)
	}()

	want := resp.EncodeSimpleString("PONG")
	if err := client.SendMessage(want); err != nil {
		t.Fatalf("SendMessage() unexpected error: %v", err)
	}

	if err := peer.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() unexpected error: %v", err)
	}
	got := make([]byte, len(want))
	if _, err := io.ReadFull(peer, got); err != nil {
		t.Fatalf("ReadFull() unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("written response = %q, want %q", got, want)
	}

	close(client.doneCh)
	select {
	case <-writeDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for client.write() to stop")
	}

	select {
	case gotClient := <-deregCh:
		if gotClient != client {
			t.Fatal("deregistered client does not match writer client")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for client deregistration")
	}
}
