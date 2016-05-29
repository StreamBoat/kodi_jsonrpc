// Package kodijsonrpc provides an interface for communicating with a Kodi/XBMC
// server via the raw JSON-RPC socket
//
// Extracted from the kodi-callback-daemon.
//
// Released under the terms of the MIT License (see LICENSE).
package kodijsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
)

// Connection is the main type for interacting with Kodi
type Connection struct {
	conn             net.Conn
	write            chan interface{}
	Notifications    chan Notification
	enc              *json.Encoder
	dec              *json.Decoder
	responseLock     sync.Mutex
	connectedLock    sync.Mutex
	connectLock      sync.Mutex
	writeWait        sync.WaitGroup
	notificationWait sync.WaitGroup
	requestID        uint32
	responses        map[uint32]*chan *rpcResponse

	Connected bool
	Closed    bool

	address string
	timeout time.Duration
}

// Request is the RPC request type
type Request struct {
	Id      *uint32                 `json:"id,omitempty"`
	Method  string                  `json:"method"`
	Params  *map[string]interface{} `json:"params,omitempty"`
	JsonRPC string                  `json:"jsonrpc"`
}

type rpcError struct {
	Code    float64                 `json:"code"`
	Message string                  `json:"message"`
	Data    *map[string]interface{} `json:"data"`
}

// Response provides a reader for returning RPC responses
type Response struct {
	channel  *chan *rpcResponse
	Pending  bool // If Pending is false, Response is unwanted, or been consumed
	readLock sync.Mutex
}

type rpcResponse struct {
	Id      *float64                `json:"id"`
	JsonRPC string                  `json:"jsonrpc"`
	Method  *string                 `json:"method"`
	Params  *map[string]interface{} `json:"params"`
	Result  interface{}             `json:"result"`
	Error   *rpcError               `json:"error"`
}

// Notification stores Kodi server->client notifications.
type Notification struct {
	Method string `json:"method" mapstructure:"method"`
	Params struct {
		Data struct {
			Item *struct {
				Type string `json:"type" mapstructure:"type"`
			} `json:"item" mapstructure:"item"` // Optional
		} `json:"data" mapstructure:"data"`
	} `json:"params" mapstructure:"params"`
}

const (
	// VERSION hold the version number for this library
	VERSION = `2.0.5`

	// KODI_MIN_VERSION specifies the minimum Kodi/XBMC API version compatible
	// with this library
	KODI_MIN_VERSION = 6

	// LogDebugLevel and friends export log level constants, mapped to their
	// logrus equivalents
	LogDebugLevel = log.DebugLevel
	LogInfoLevel  = log.InfoLevel
	LogWarnLevel  = log.WarnLevel
	LogErrorLevel = log.ErrorLevel
	LogFatalLevel = log.FatalLevel
	LogPanicLevel = log.PanicLevel
)

func init() {
	// Initialize logger, default to level Info
	log.SetLevel(LogInfoLevel)
}

// New returns a Connection to the specified address.
// If timeout (seconds) is greater than zero, connection will fail if initial
// connection is not established within this time.
//
// User must ensure Close() is called on returned Connection when finished with
// it, to avoid leaks.
func New(address string, timeout time.Duration) (conn Connection, err error) {
	conn = Connection{}
	err = conn.init(address, timeout)

	return conn, err
}

// SetLogLevel adjusts the level of logger output, level must be one of:
//
// LogDebugLevel
// LogInfoLevel
// LogWarnLevel
// LogErrorLevel
// LogFatalLevel
// LogPanicLevel
func SetLogLevel(level log.Level) {
	log.SetLevel(level)
}

// Read returns the result and any errors from the response channel
// If timeout (seconds) is greater than zero, read will fail if not returned
// within this time.
func (rchan *Response) Read(timeout time.Duration) (result interface{}, err error) {
	rchan.readLock.Lock()
	defer close(*rchan.channel)
	defer func() {
		rchan.Pending = false
	}()
	defer rchan.readLock.Unlock()

	if rchan.Pending != true {
		return result, errors.New(`No pending responses!`)
	}
	if rchan.channel == nil {
		return result, errors.New(`Expected response channel, but got nil!`)
	}

	res := new(rpcResponse)
	if timeout > 0 {
		select {
		case res = <-*rchan.channel:
		case <-time.After(timeout * time.Second):
			return result, errors.New(`Timeout waiting on response channel`)
		}
	} else {
		res = <-*rchan.channel
	}
	if res == nil {
		return result, errors.New(`Empty result received`)
	}
	result, err = res.unpack()

	return result, err
}

// Unpack the result and any errors from the Response
func (res *rpcResponse) unpack() (result interface{}, err error) {
	if res.Error != nil {
		err = fmt.Errorf(`Kodi error (%v): %v`, res.Error.Code, res.Error.Message)
	} else if res.Result != nil {
		return res.Result, nil
	} else {
		log.WithField(`response`, res).Debug(`Received unknown response type from Kodi`)
		// XXX err nil?
	}
	return nil, err
}

// init brings up an instance of the Kodi Connection
func (c *Connection) init(address string, timeout time.Duration) (err error) {

	if c.address == `` {
		c.address = address
	}
	if c.timeout == 0 && timeout != 0 {
		c.timeout = timeout
	}

	if err = c.connect(); err != nil {
		return err
	}

	c.write = make(chan interface{}, 16)
	c.Notifications = make(chan Notification, 16)

	c.responses = make(map[uint32]*chan *rpcResponse)

	go c.reader()
	go c.writer()

	rchan, _ := c.Send(Request{Method: `JSONRPC.Version`}, true)
	if err != nil {
		log.WithField(`error`, err).Error(`Connection closed`)
		return err
	}

	res, err := rchan.Read(c.timeout)
	if err != nil {
		log.WithField(`error`, err).Error(`Kodi responded`)
		return err
	}
	resmap := res.(map[string]interface{})
	if version := resmap[`version`].(map[string]interface{}); version != nil {
		if version[`major`].(float64) < KODI_MIN_VERSION {
			return errors.New(`Kodi version too low, upgrade to Frodo or later`)
		}
	}

	return
}

// Send an RPC request to the Kodi server.
// Returns a Response, but does not attach a channel for it if wantResponse is
// false (for fire-and-forget commands that don't return any useful response).
// Returns error on closed connection
func (c *Connection) Send(req Request, wantResponse bool) (res Response, err error) {
	if c.Closed {
		return res, errors.New(`Cannot send on closed connection`)
	}
	req.JsonRPC = `2.0`
	res = Response{}

	c.writeWait.Add(1)
	if wantResponse == true {
		c.responseLock.Lock()
		id := c.requestID
		ch := make(chan *rpcResponse)
		c.responses[id] = &ch
		c.requestID++
		c.responseLock.Unlock()
		req.Id = &id

		log.WithField(`request`, req).Debug(`Sending Kodi Request (response desired)`)
		c.write <- req
		res.channel = &ch
		res.Pending = true
	} else {
		log.WithField(`request`, req).Debug(`Sending Kodi Request (response undesired)`)
		c.write <- req
		res.Pending = false
	}
	c.writeWait.Done()

	return
}

// connected sets whether we're currently connected or not
func (c *Connection) connected(status bool) {
	c.connectedLock.Lock()
	defer c.connectedLock.Unlock()
	c.Connected = status
}

// connect establishes a TCP connection
func (c *Connection) connect() (err error) {
	c.connected(false)
	c.connectLock.Lock()
	defer c.connectLock.Unlock()

	// If we blocked on the lock, and another routine connected in the mean
	// time, return early
	if c.Connected {
		return
	}

	if c.conn != nil {
		_ = c.conn.Close()
	}

	c.conn, err = net.Dial(`tcp`, c.address)
	if err != nil {
		success := make(chan bool, 1)
		done := make(chan bool, 1)
		go func() {
			for err != nil {
				log.WithField(`error`, err).Error(`Connecting to Kodi`)
				log.Info(`Attempting reconnect...`)
				time.Sleep(time.Second)
				c.conn, err = net.Dial(`tcp`, c.address)
				select {
				case <-done:
					break
				default:
				}
			}
			success <- true
		}()
		if c.timeout > 0 {
			select {
			case <-success:
			case <-time.After(c.timeout * time.Second):
				done <- true
				log.Error(`Timeout connecting to Kodi`)
				return err
			}
		} else {
			<-success
		}
	}

	c.enc = json.NewEncoder(c.conn)
	c.dec = json.NewDecoder(c.conn)

	log.Info(`Connected to Kodi`)
	c.connected(true)

	return
}

// writer loop processes outbound requests
func (c *Connection) writer() {
	for {
		var req interface{}
		req = <-c.write
		for err := c.enc.Encode(req); err != nil; {
			log.WithField(`error`, err).Warn(`Failed encoding request for Kodi`)
			if err = c.connect(); err != nil {
				continue
			}
			err = c.enc.Encode(req)
		}
	}
}

// reader loop processes inbound responses and notifications
func (c *Connection) reader() {
	for {
		res := new(rpcResponse)
		err := c.dec.Decode(res)
		if _, ok := err.(net.Error); err == io.EOF || ok {
			log.WithField(`error`, err).Error(`Reading from Kodi`)
			log.Error(`If this error persists, make sure you are using the JSON-RPC port, not the HTTP port!`)
			for err != nil {
				err = c.connect()
			}
		} else if err != nil {
			log.WithField(`error`, err).Error(`Decoding response from Kodi`)
			continue
		}
		if res.Id == nil && res.Method != nil {
			c.notificationWait.Add(1)
			// Process notifications in a separate routine so we don't delay the
			// processing of standard responses.  This does mean losing ordering
			// guarantees for notifications.
			go func() {
				if res.Params != nil {
					log.WithFields(log.Fields{
						`notification.Method`: *res.Method,
						`notification.Params`: *res.Params,
					}).Debug(`Received notification from Kodi`)
				} else {
					log.WithField(`notification.Method`, *res.Method).Debug(`Received notification from Kodi`)
				}
				n := Notification{}
				n.Method = *res.Method
				err := mapstructure.Decode(res.Params, &n.Params)
				if err != nil {
					log.WithField(`notification.Method`, *res.Method).Warn(`Decoding notifcation failed`)
					return
				}
				// Implement notification writes as a ring buffer.
				// In case the client is not processing notifications, we don't
				// want to block indefinitely here, instead drop the oldest
				// notification after 200ms, and log a warning
				select {
				case c.Notifications <- n:
				case <-time.After(200 * time.Millisecond):
					<-c.Notifications
					c.Notifications <- n
					log.Warn(`Dropped oldest notification, buffer full`)
				}
				c.notificationWait.Done()
			}()
		} else if res.Id != nil {
			if ch := c.responses[uint32(*res.Id)]; ch != nil {
				if res.Result != nil {
					log.WithField(`response.Result`, res.Result).Debug(`Received response from Kodi`)
				}
				*ch <- res
			} else {
				log.WithField(`response.Id`, *res.Id).Warn(`Received Kodi response for unknown request`)
				log.WithField(`connection.responses`, c.responses).Debug(`Current response channels`)
			}
		} else {
			if res.Error != nil {
				log.WithField(`response.Error`, *res.Error).Warn(`Received unparseable Kodi response`)
			} else {
				log.WithField(`response`, res).Warn(`Received unparseable Kodi response`)
			}
		}
	}
}

// Close closes the Kodi connection and associated channels
// Subsequent Sends will return an error for closed connections
func (c *Connection) Close() {
	if c.Closed {
		return
	}
	c.Closed = true

	if c.write != nil {
		c.writeWait.Wait()
		close(c.write)
	}
	if c.Notifications != nil {
		c.notificationWait.Wait()
		close(c.Notifications)
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}

	log.Info(`Disconnected from Kodi`)
}
