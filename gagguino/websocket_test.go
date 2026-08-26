package gaggiuino

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestWsURL(t *testing.T) {
	cases := map[string]string{
		"http://gaggiuino.local":  "ws://gaggiuino.local/ws",
		"https://192.168.1.5":     "wss://192.168.1.5/ws",
		"http://gaggiuino.local/": "ws://gaggiuino.local/ws",
	}

	for in, want := range cases {
		got, err := wsURL(in)
		if err != nil {
			t.Fatalf("wsURL(%q) returned error: %v", in, err)
		}
		if got != want {
			t.Fatalf("wsURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func encodeEnvelope(t *testing.T, action string, data []byte) []byte {
	t.Helper()
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, action)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendBytes(b, data)
	return b
}

func TestDecodeEnvelope(t *testing.T) {
	var payload []byte
	payload = protowire.AppendTag(payload, 4, protowire.Fixed32Type)
	payload = protowire.AppendFixed32(payload, 0x42280000) // 42.0f

	raw := encodeEnvelope(t, "d_sensor_snap", payload)

	action, data, err := decodeEnvelope(raw)
	if err != nil {
		t.Fatalf("decodeEnvelope returned error: %v", err)
	}
	if action != "d_sensor_snap" {
		t.Fatalf("expected action %q, got %q", "d_sensor_snap", action)
	}

	fields := parseProtoFields(data)
	f, ok := fields[4]
	if !ok {
		t.Fatalf("expected field 4 to be present")
	}
	if got := f.asFloat32(); got != 42.0 {
		t.Fatalf("expected temperature 42.0, got %v", got)
	}
}

func TestWsClientApplySensorSnapshot(t *testing.T) {
	var payload []byte
	payload = protowire.AppendTag(payload, 4, protowire.Fixed32Type)
	payload = protowire.AppendFixed32(payload, 0x42280000) // 42.0f
	payload = protowire.AppendTag(payload, 10, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 7)
	payload = protowire.AppendTag(payload, 12, protowire.VarintType)
	payload = protowire.AppendVarint(payload, 1)

	c := newWSClient("http://gaggiuino.local")
	c.applySensorSnapshot(payload)

	st, ok := c.snapshot()
	if !ok {
		t.Fatalf("expected snapshot to be present")
	}
	if st.Temperature != 42.0 {
		t.Fatalf("expected temperature 42.0, got %v", st.Temperature)
	}
	if st.WaterLevel != 7 {
		t.Fatalf("expected water level 7, got %v", st.WaterLevel)
	}
	if !st.BrewSwitchState {
		t.Fatalf("expected brew switch state true")
	}
}
