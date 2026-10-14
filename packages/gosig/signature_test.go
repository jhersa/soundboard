package gosig

import (
	"testing"
	"time"
)

// Vector fijo: si la firma cambia, los jobs en vuelo entre una API y un processor de versiones
// distintas se rechazan, así que este test obliga a cambiarla a propósito.
const (
	vectorSecret    = "vector-secreto"
	vectorTimestamp = "1760000000"
	vectorBody      = `{"jobId":"j1","status":"running"}`
	vectorSignature = "a5dba8f052eabb323a2bf0f8c58569f9cb2ce0688300d6891e0922a2683361a5"
	vectorKey       = "uploads/abc.mp3"
	vectorKeySig    = "1152fc244bf74d179e3a058f422059db77bde15ceb23271b4cfc3397b4244970"
)

var vectorTime = time.Unix(1760000000, 0)

func TestVectorCompartido(t *testing.T) {
	headers := Headers(vectorSecret, []byte(vectorBody), vectorTime)
	if headers[HeaderTimestamp] != vectorTimestamp || headers[HeaderSignature] != vectorSignature {
		t.Fatalf("firma distinta a la esperada: %v", headers)
	}
	if !Verify(vectorSecret, []byte(vectorKey), vectorTimestamp, vectorKeySig, vectorTime) {
		t.Fatal("no verifica la firma de la key")
	}
}

func TestRechazos(t *testing.T) {
	body := []byte(vectorBody)
	cases := map[string]struct {
		timestamp, signature string
		now                  time.Time
	}{
		"sin headers":      {"", "", vectorTime},
		"firma alterada":   {vectorTimestamp, "0" + vectorSignature[1:], vectorTime},
		"expirada":         {vectorTimestamp, vectorSignature, vectorTime.Add(301 * time.Second)},
		"del futuro":       {vectorTimestamp, vectorSignature, vectorTime.Add(-301 * time.Second)},
		"timestamp basura": {"abc", vectorSignature, vectorTime},
	}
	for name, tc := range cases {
		if Verify(vectorSecret, body, tc.timestamp, tc.signature, tc.now) {
			t.Errorf("%s: debió rechazarse", name)
		}
	}
	if !Verify(vectorSecret, body, vectorTimestamp, vectorSignature, vectorTime.Add(299*time.Second)) {
		t.Error("dentro de la ventana debió aceptarse")
	}
}
