package test

import (
	"encoding/json"
	"testing"

	res "github.com/jirenius/go-res"
)

// Test that the service can be served without error
func TestStart(t *testing.T) {
	runTest(t, nil, nil)
}

// Test that service can be served without logger
func TestWithoutLogger(t *testing.T) {
	runTestWithLogger(t, nil, nil, nil)
}

// Test that Logger returns the logger set with SetLogger
func TestServiceLogger(t *testing.T) {
	l := newMemLogger(true, true)
	runTestWithLogger(t, l, func(s *Session) {
		if s.Logger() != l {
			t.Errorf("expected Logger to return the logger passed to SetLogger, but it didn't")
		}
	}, nil)
}

// Test that With returns an error if there is no registered pattern matching the resource
func TestServiceWithWithNoMatchingPattern(t *testing.T) {
	runTest(t, nil, func(s *Session) {
		err := s.With("test.model", func(r res.Resource) {})
		if err == nil {
			t.Errorf("expected With to return an error, but it didn't")
		}
	})
}

// Test that SetReset sets which resources are reset when calling Reset.
func TestServiceSetReset(t *testing.T) {
	resources := []string{"test.foo.>", "test.bar.>"}
	access := []string{"test.zoo.>", "test.baz.>"}

	var s *Session
	l := newMemLogger(true, true)
	c := NewTestConn()
	r := res.NewService("test")
	r.SetLogger(l)
	r.SetReset(resources, access)

	s = &Session{
		MockConn: c,
		Service:  r,
		cl:       make(chan struct{}),
	}

	go func() {
		defer close(s.cl)
		if err := r.Serve(c); err != nil {
			panic("test: failed to start service: " + err.Error())
		}
	}()
	s.GetMsg(t).
		AssertSubject(t, "system.reset").
		AssertPayload(t, map[string]interface{}{
			"resources": resources,
			"access":    access,
		})

	teardown(s)
}

// Test that TokenEvent sends a connection token event.
func TestServiceTokenEvent(t *testing.T) {
	token := `{"id":42,"user":"foo","role":"admin"}`
	runTest(t, nil, func(s *Session) {
		s.TokenEvent(defaultCID, json.RawMessage(token))
		s.GetMsg(t).AssertSubject(t, "conn."+defaultCID+".token").AssertPayload(t, json.RawMessage(`{"token":`+token+`}`))
	})
}

// Test that TokenEvent with nil sends a connection token event with a nil token.
func TestServiceNilTokenEvent(t *testing.T) {
	runTest(t, nil, func(s *Session) {
		s.TokenEvent(defaultCID, nil)
		s.GetMsg(t).AssertSubject(t, "conn."+defaultCID+".token").AssertPayload(t, json.RawMessage(`{"token":null}`))
	})
}

// Test that TokenEvent with an invalid cid causes panic.
func TestServiceTokenEventWithInvalidCID(t *testing.T) {
	runTest(t, nil, func(s *Session) {
		defer func() {
			v := recover()
			if v == nil {
				t.Fatalf("expected a panic, but nothing happened")
			}
		}()
		s.TokenEvent("invalid.*.cid", nil)
	})
}
