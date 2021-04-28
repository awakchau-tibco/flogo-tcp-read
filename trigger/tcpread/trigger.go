package tcpread

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/trigger"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{}, &Reply{})
var logger log.Logger

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

// Factory is a kafka trigger factory
type Factory struct {
}

// Metadata implements trigger.Factory.Metadata
func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// New implements trigger.Factory.New
func (*Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	s := &Settings{}
	err := metadata.MapToStruct(config.Settings, s, true)
	if err != nil {
		return nil, err
	}

	return &Trigger{settings: s}, nil
}

// Trigger ...
type Trigger struct {
	settings    *Settings
	handlers    []trigger.Handler
	listener    net.Listener
	logger      log.Logger
	delimiter   byte
	connections []net.Conn
}

// Initialize initializes the trigger
func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	host := t.settings.Host
	port := t.settings.Port
	t.handlers = ctx.GetHandlers()
	logger = ctx.Logger()
	err := t.setDelimiter()
	if err != nil {
		return err
	}
	logger.Debugf("Delimiter is set to bytes [%+v]", t.delimiter)
	if port == "" {
		return errors.New("Valid port must be set")
	}
	listener, err := net.Listen(t.settings.Network, host+":"+port)
	if err != nil {
		return err
	}
	t.listener = listener
	return err
}

func (t *Trigger) setDelimiter() error {
	if len(t.settings.CustomDelimiter) > 0 {
		// Refer this for delimiter hex codes: http://www.columbia.edu/kermit/ascii.html
		delimBytes, err := hex.DecodeString(t.settings.CustomDelimiter)
		if err != nil {
			return err
		}
		r, _ := utf8.DecodeRune(delimBytes)
		t.delimiter = byte(r)
		logger.Debugf("Custom delimiter is set to: Decimal [%[1]v] or Hex [%[1]x]", t.delimiter)
		return nil
	}
	switch t.settings.Delimiter {
	case "Carriage Return (CR)":
		t.delimiter = '\r'
	case "Line Feed (LF)":
		t.delimiter = '\n'
	case "Form Feed (FF)":
		t.delimiter = '\f'
	}
	logger.Debugf("Delimiter is set to: Decimal [%[1]v] or Hex [%[1]x]", t.delimiter)
	return nil
}

// Start starts the kafka trigger
func (t *Trigger) Start() error {
	go t.waitForConnection()
	logger.Infof("Started listener on Port: %s, Network: %s", t.settings.Port, t.settings.Network)
	return nil
}

func (t *Trigger) waitForConnection() {
	for {
		// Listen for an incoming connection.
		conn, err := t.listener.Accept()
		if err != nil {
			errString := err.Error()
			if !strings.Contains(errString, "use of closed network connection") {
				logger.Error("Error accepting connection: ", err.Error())
			}
			return
		}
		logger.Infof("Handling new connection from client: %s", conn.RemoteAddr().String())
		// Handle connections in a new goroutine.
		go t.handleNewConnection(conn)
	}
}

func (t *Trigger) handleNewConnection(conn net.Conn) {
	//Gather connection list for later cleanup
	t.connections = append(t.connections, conn)
	for {
		if t.settings.TimeoutMs > 0 {
			logger.Debug("Setting timeout: ", t.settings.TimeoutMs)
			conn.SetDeadline(time.Now().Add(time.Duration(t.settings.TimeoutMs) * time.Millisecond))
		}
		output := &Output{}
		if t.delimiter != 0 {
			data, err := bufio.NewReader(conn).ReadBytes(t.delimiter)
			if err != nil {
				errString := err.Error()
				if !strings.Contains(errString, "use of closed network connection") {
					logger.Error("Error reading data from connection: ", err.Error())
				} else {
					logger.Info("Connection is closed.")
				}
				if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
					// Return if not timeout error
					return
				}

			} else {
				output.Data = string(data[:len(data)-1])
			}
		} else {
			var buf bytes.Buffer
			_, err := io.Copy(&buf, conn)
			if err != nil {
				errString := err.Error()
				if !strings.Contains(errString, "use of closed network connection") {
					logger.Error("Error reading data from connection: ", err.Error())
				} else {
					logger.Info("Connection is closed.")
				}
				if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
					// Return if not timeout error
					return
				}
			} else {
				output.Data = string(buf.Bytes())
			}
		}

		if output.Data != "" {
			var replyData []string
			for i := 0; i < len(t.handlers); i++ {
				results, err := t.handlers[i].Handle(context.Background(), output)
				if err != nil {
					logger.Error("Error invoking action : ", err.Error())
					continue
				}

				reply := &Reply{}
				err = reply.FromMap(results)
				if err != nil {
					logger.Error("Failed to convert flow output : ", err.Error())
					continue
				}
				if reply.Reply != "" {
					replyData = append(replyData, reply.Reply)
				}
			}

			if len(replyData) > 0 {
				replyToSend := strings.Join(replyData, string(t.delimiter))
				// Send a response back to client contacting us.
				_, err := conn.Write([]byte(replyToSend + "\n"))
				if err != nil {
					logger.Error("Failed to write to connection : ", err.Error())
				}
			}
		}
	}
}

// Stop implements ext.Trigger.Stop
func (t *Trigger) Stop() error {
	for i := 0; i < len(t.connections); i++ {
		t.connections[i].Close()
	}
	t.connections = nil
	if t.listener != nil {
		t.listener.Close()
	}
	logger.Info("Stopped listener")
	return nil
}
