package gaggiuino

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protowire"
)

const (
	wsPath             = "/ws"
	wsHandshakeTimeout = 10 * time.Second
	wsMinBackoff       = time.Second
	wsMaxBackoff       = 30 * time.Second
)

// wsClient maintains a WebSocket connection to the Gaggiuino device (see
// https://gaggiuino.github.io/#/rest-api/websocket) and keeps the latest
// known machine state available for the exporter to publish. Whenever the
// connection can't be established or drops, callers should fall back to
// scraping the REST API instead.
type wsClient struct {
	baseURL string

	mu        sync.Mutex
	connected bool
	state     status
	haveState bool
}

func newWSClient(baseURL string) *wsClient {
	return &wsClient{baseURL: baseURL}
}

// wsURL derives the ws(s)://<host>/ws endpoint from the configured HTTP base URL.
func wsURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + wsPath
	return u.String(), nil
}

func (c *wsClient) isConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *wsClient) setConnected(v bool) {
	c.mu.Lock()
	c.connected = v
	c.mu.Unlock()
}

// snapshot returns the latest state received over the WebSocket, if any has arrived yet.
func (c *wsClient) snapshot() (status, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.haveState
}

func (c *wsClient) update(fn func(*status)) {
	c.mu.Lock()
	fn(&c.state)
	c.haveState = true
	state := c.state
	c.mu.Unlock()
	Logger.Debug("state updated", "state", state)
}

// run keeps trying to (re)connect to the WebSocket endpoint, with backoff
// between attempts. While no connection is established, isConnected reports
// false so the caller keeps using the REST API fallback.
func (c *wsClient) run(ctx context.Context) {
	target, err := wsURL(c.baseURL)
	if err != nil {
		Logger.Error("invalid websocket URL, disabling websocket", "baseURL", c.baseURL, "error", err)
		return
	}

	backoff := wsMinBackoff
	for {
		if ctx.Err() != nil {
			return
		}

		if err := c.runOnce(ctx, target); err != nil {
			Logger.Debug("websocket connection failed, falling back to HTTP polling", "target", target, "error", err)
		}
		c.setConnected(false)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > wsMaxBackoff {
			backoff = wsMaxBackoff
		}
	}
}

// runOnce dials the WebSocket endpoint and reads messages until the
// connection fails or ctx is cancelled. A non-nil error, including a failed
// upgrade (e.g. non-101 response), signals the caller to keep polling REST instead.
func (c *wsClient) runOnce(ctx context.Context, target string) error {
	Logger.Debug("dialing websocket", "target", target)
	dialer := websocket.Dialer{HandshakeTimeout: wsHandshakeTimeout}
	conn, resp, err := dialer.DialContext(ctx, target, nil)
	if err != nil {
		if resp != nil {
			Logger.Debug("websocket dial got non-101 response", "target", target, "status", resp.Status)
			resp.Body.Close()
		}
		return fmt.Errorf("dial/upgrade failed: %w", err)
	}
	defer conn.Close()

	Logger.Debug("websocket connection established", "target", target)
	c.setConnected(true)

	for {
		if ctx.Err() != nil {
			return nil
		}

		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		if msgType != websocket.BinaryMessage {
			Logger.Debug("ignoring non-binary websocket message", "type", msgType)
			continue
		}

		action, payload, err := decodeEnvelope(data)
		if err != nil {
			Logger.Debug("failed to decode websocket envelope", "bytes", len(data), "error", err)
			continue
		}

		Logger.Debug("received websocket message", "action", action, "bytes", len(payload))
		c.handleMessage(action, payload)
	}
}

func (c *wsClient) handleMessage(action string, payload []byte) {
	switch action {
	case "d_sensor_snap":
		c.applySensorSnapshot(payload)
	case "d_sys_state":
		c.applySystemState(payload)
	case "d_act_prof":
		c.applyActiveProfile(payload)
	default:
		Logger.Debug("unhandled websocket action", "action", action)
	}
}

// applySensorSnapshot updates the cached state from a SensorStateSnapshotDto payload.
func (c *wsClient) applySensorSnapshot(payload []byte) {
	fields := parseProtoFields(payload)
	c.update(func(s *status) {
		if f, ok := fields[2]; ok {
			s.SteamSwitchState = f.asBool()
		}
		if f, ok := fields[4]; ok {
			s.Temperature = f.asFloat32()
		}
		if f, ok := fields[6]; ok {
			s.Pressure = f.asFloat32()
		}
		if f, ok := fields[9]; ok {
			s.Weight = f.asFloat32()
		}
		if f, ok := fields[10]; ok {
			s.WaterLevel = f.asInt()
		}
		if f, ok := fields[12]; ok {
			s.BrewSwitchState = f.asBool()
		}
	})
}

// applySystemState updates the cached state from a SystemStateDto payload.
func (c *wsClient) applySystemState(payload []byte) {
	fields := parseProtoFields(payload)
	c.update(func(s *status) {
		if f, ok := fields[6]; ok {
			s.Uptime = f.asInt()
		}
	})
}

// applyActiveProfile updates the cached state from a ProfileDto payload (d_act_prof).
func (c *wsClient) applyActiveProfile(payload []byte) {
	fields := parseProtoFields(payload)
	c.update(func(s *status) {
		if f, ok := fields[1]; ok {
			s.ProfileName = f.asString()
		}
		if f, ok := fields[4]; ok {
			s.TargetTemperature = f.asFloat32()
		}
		if f, ok := fields[6]; ok {
			s.ProfileID = f.asInt()
		}
	})
}

// protoField holds the raw decoded value of a single top-level protobuf field,
// interpreted lazily depending on the wire type it was read with.
type protoField struct {
	typ     protowire.Type
	varint  uint64
	fixed32 uint32
	bytes   []byte
}

func (f protoField) asBool() bool { return f.varint != 0 }
func (f protoField) asInt() int   { return int(f.varint) }
func (f protoField) asFloat32() float64 {
	return float64(math.Float32frombits(f.fixed32))
}
func (f protoField) asString() string { return string(f.bytes) }

// parseProtoFields decodes the top-level fields of a protobuf message without
// requiring generated bindings for the .proto schema (see
// https://gaggiuino.github.io/#/rest-api/websocket for the message shapes).
func parseProtoFields(b []byte) map[protowire.Number]protoField {
	fields := make(map[protowire.Number]protoField)
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			break
		}
		b = b[n:]

		switch typ {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b)
			if n < 0 {
				return fields
			}
			fields[num] = protoField{typ: typ, varint: v}
			b = b[n:]
		case protowire.Fixed32Type:
			v, n := protowire.ConsumeFixed32(b)
			if n < 0 {
				return fields
			}
			fields[num] = protoField{typ: typ, fixed32: v}
			b = b[n:]
		case protowire.Fixed64Type:
			_, n := protowire.ConsumeFixed64(b)
			if n < 0 {
				return fields
			}
			b = b[n:]
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(b)
			if n < 0 {
				return fields
			}
			fields[num] = protoField{typ: typ, bytes: v}
			b = b[n:]
		default:
			n := protowire.ConsumeFieldValue(num, typ, b)
			if n < 0 {
				return fields
			}
			b = b[n:]
		}
	}
	return fields
}

// decodeEnvelope decodes the outer WebSocketMessageDto{action, data} envelope.
func decodeEnvelope(b []byte) (string, []byte, error) {
	fields := parseProtoFields(b)
	action, ok := fields[1]
	if !ok {
		return "", nil, fmt.Errorf("websocket envelope missing action field")
	}
	return action.asString(), fields[2].bytes, nil
}
