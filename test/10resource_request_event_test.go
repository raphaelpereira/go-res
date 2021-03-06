package test

import (
	"encoding/json"
	"testing"

	res "github.com/jirenius/go-res"
)

var eventTestTbl = []struct {
	Event   string
	Payload json.RawMessage
}{
	{"foo", json.RawMessage(`{"bar":42}`)},
	{"foo", json.RawMessage(`{"bar":42,"baz":null}`)},
	{"foo", json.RawMessage(`["bar",42]`)},
	{"foo", json.RawMessage(`"bar"`)},
	{"foo", json.RawMessage(`null`)},
	{"foo", nil},
	{"_foo_", json.RawMessage(`{"bar":42}`)},
	{"12", json.RawMessage(`{"bar":42}`)},
	{"<_!", json.RawMessage(`{"bar":42}`)},
}

var changeEventTestTbl = []struct {
	Payload map[string]interface{}
}{
	{map[string]interface{}{"foo": 42}},
	{map[string]interface{}{"foo": "bar"}},
	{map[string]interface{}{"foo": nil}},
	{map[string]interface{}{"foo": 12, "bar": true}},
	{map[string]interface{}{"foo": "bar", "deleted": res.DeleteAction}},
	{map[string]interface{}{"foo": res.Ref("test.model.bar")}},
}

var addEventTestTbl = []struct {
	Value    interface{}
	Idx      int
	Expected json.RawMessage
}{
	{42, 0, json.RawMessage(`{"value":42,"idx":0}`)},
	{"bar", 1, json.RawMessage(`{"value":"bar","idx":1}`)},
	{nil, 2, json.RawMessage(`{"value":null,"idx":2}`)},
	{true, 3, json.RawMessage(`{"value":true,"idx":3}`)},
	{res.Ref("test.model.bar"), 4, json.RawMessage(`{"value":{"rid":"test.model.bar"},"idx":4}`)},
}

var removeEventTestTbl = []struct {
	Idx      int
	Expected json.RawMessage
}{
	{0, json.RawMessage(`{"idx":0}`)},
	{1, json.RawMessage(`{"idx":1}`)},
	{2, json.RawMessage(`{"idx":2}`)},
}

// Test method Event sends a custom event on the resource.
func TestEvent(t *testing.T) {
	for _, l := range eventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("model", res.Call("method", func(r res.CallRequest) {
				r.Event(l.Event, l.Payload)
				r.OK(nil)
			}))
		}, func(s *Session) {
			inb := s.Request("call.test.model.method", nil)
			s.GetMsg(t).AssertSubject(t, "event.test.model."+l.Event).AssertPayload(t, l.Payload)
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test method Event sends a custom event on the resource, using With.
func TestEventUsingWith(t *testing.T) {
	for _, l := range eventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("model")
		}, func(s *Session) {
			AssertNoError(t, s.With("test.model", func(r res.Resource) {
				r.Event(l.Event, l.Payload)
			}))
			s.GetMsg(t).AssertSubject(t, "event.test.model."+l.Event).AssertPayload(t, l.Payload)
		})
	}
}

// Test method Event panic if the event is one of the pre-defined or reserved events,
// "change", "delete", "add", "remove", "patch", "reaccess", or "unsubscribe". Or if the event name is invalid.
func TestEventPanicsOnInvalid(t *testing.T) {
	tbl := []struct {
		Event string
	}{
		{"change"},
		{"delete"},
		{"add"},
		{"remove"},
		{"patch"},
		{"reaccess"},
		{"unsubscribe"},
		{"foo.bar"},
		{"foo.>"},
		{"*"},
		{"*.bar"},
		{"?foo"},
		{"foo?"},
		{">.baz"},
	}

	for _, l := range tbl {
		runTestAsync(t, func(s *Session) {
			s.Handle("model")
		}, func(s *Session, done func()) {
			AssertNoError(t, s.With("test.model", func(r res.Resource) {
				defer func() {
					v := recover()
					if v == nil {
						t.Errorf("expected event %#v to panic, but nothing happened", l.Event)
					}
					done()
				}()
				r.Event(l.Event, nil)
			}))
		})
	}
}

// Test ChangeEvents sends a change event with properties that has been changed
// and their new values.
func TestChangeEvent(t *testing.T) {
	for _, l := range changeEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.ChangeEvent(l.Payload)
					r.OK(nil)
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.model.method", nil)
			s.GetMsg(t).AssertPayload(t, l.Payload).AssertSubject(t, "event.test.model.change")
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test ChangeEvents does not sends a change event when no properties has been changed.
func TestEmptyChangeEvent(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}),
			res.Call("method", func(r res.CallRequest) {
				r.ChangeEvent(nil)
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ChangeEvents sends a change event with properties that has been changed
// and their new values, using With
func TestChangeEventUsingWith(t *testing.T) {
	for _, l := range changeEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("model", res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}))
		}, func(s *Session) {
			AssertNoError(t, s.With("test.model", func(r res.Resource) {
				r.ChangeEvent(l.Payload)
			}))
			s.GetMsg(t).AssertPayload(t, l.Payload).AssertSubject(t, "event.test.model.change")
		})
	}
}

// Test ChangeEvent panics if the resource is an untyped resource.
func TestChangeEventPanicsOnUntypedResource(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("bar",
			res.Call("method", func(r res.CallRequest) {
				panicked := true
				defer func() {
					if !panicked {
						t.Errorf("expected ChangeEvent to panic, but nothing happened")
					}
				}()
				r.ChangeEvent(map[string]interface{}{"foo": 42})
				panicked = false
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.bar.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ChangeEvent panics if the resource is an untyped resource, using With
func TestChangeEventPanicsOnUntypedResourceUsingWith(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("bar")
	}, func(s *Session, done func()) {
		s.With("test.bar", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected ChangeEvent to panic, but nothing happened")
				}
				done()
			}()
			r.ChangeEvent(map[string]interface{}{"foo": 42})
		})
	})
}

// Test ChangeEvent panics if the resource is a collection
func TestChangeEventPanicsOnCollection(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("collection",
			res.GetCollection(func(r res.CollectionRequest) {
				r.NotFound()
			}),
			res.Call("method", func(r res.CallRequest) {
				panicked := true
				defer func() {
					if !panicked {
						t.Errorf("expected ChangeEvent to panic, but nothing happened")
					}
				}()
				r.ChangeEvent(map[string]interface{}{"foo": 42})
				panicked = false
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.collection.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ChangeEvent panics if the resource is a collection, using With
func TestChangeEventPanicsOnCollectionUsingWith(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		s.With("test.collection", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected ChangeEvent to panic, but nothing happened")
				}
				done()
			}()
			r.ChangeEvent(map[string]interface{}{"foo": 42})
		})
	})
}

// Test AddEvent sends an add event with idx and the added value.
func TestAddEvent(t *testing.T) {
	for _, l := range addEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.AddEvent(l.Value, l.Idx)
					r.OK(nil)
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.collection.method", nil)
			s.GetMsg(t).AssertPayload(t, l.Expected).AssertSubject(t, "event.test.collection.add")
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test AddEvent sends an add event with idx and the added value, using With
func TestAddEventUsingWith(t *testing.T) {
	for _, l := range addEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
				r.NotFound()
			}))
		}, func(s *Session) {
			AssertNoError(t, s.With("test.collection", func(r res.Resource) {
				r.AddEvent(l.Value, l.Idx)
			}))
			s.GetMsg(t).AssertPayload(t, l.Expected).AssertSubject(t, "event.test.collection.add")
		})
	}
}

// Test AddEvent panics if the resource is an untyped resource.
func TestAddEventPanicsOnUntypedResource(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("bar",
			res.Call("method", func(r res.CallRequest) {
				panicked := true
				defer func() {
					if !panicked {
						t.Errorf("expected AddEvent to panic, but nothing happened")
					}
				}()
				r.AddEvent("foo", 0)
				panicked = false
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.bar.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test AddEvent panics if the resource is an untyped resource, using With
func TestAddEventPanicsOnUntypedResourceUsingWith(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("bar")
	}, func(s *Session, done func()) {
		s.With("test.bar", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected AddEvent to panic, but nothing happened")
				}
				done()
			}()
			r.AddEvent("foo", 0)
		})
	})
}

// Test AddEvent panics if the resource is a model
func TestAddEventPanicsOnModel(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}),
			res.Call("method", func(r res.CallRequest) {
				panicked := true
				defer func() {
					if !panicked {
						t.Errorf("expected AddEvent to panic, but nothing happened")
					}
				}()
				r.AddEvent("foo", 0)
				panicked = false
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test AddEvent panics if the resource is a model, using With
func TestAddEventPanicsOnModelUsingWith(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		s.With("test.model", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected AddEvent to panic, but nothing happened")
				}
				done()
			}()
			r.AddEvent("foo", 0)
		})
	})
}

// Test AddEvent panics if idx is less than zero
func TestAddEventPanicsOnIdxLessThanZero(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		s.With("test.collection", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected AddEvent to panic, but nothing happened")
				}
				done()
			}()
			r.AddEvent("foo", -1)
		})
	})
}

// Test RemoveEvent sends a remove event with idx.
func TestRemoveEvent(t *testing.T) {
	for _, l := range removeEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("collection",
				res.GetCollection(func(r res.CollectionRequest) {
					r.NotFound()
				}),
				res.Call("method", func(r res.CallRequest) {
					r.RemoveEvent(l.Idx)
					r.OK(nil)
				}),
			)
		}, func(s *Session) {
			inb := s.Request("call.test.collection.method", nil)
			s.GetMsg(t).AssertPayload(t, l.Expected).AssertSubject(t, "event.test.collection.remove")
			s.GetMsg(t).AssertSubject(t, inb)
		})
	}
}

// Test RemoveEvent sends an remove event with idx, using With
func TestRemoveEventUsingWith(t *testing.T) {
	for _, l := range removeEventTestTbl {
		runTest(t, func(s *Session) {
			s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
				r.NotFound()
			}))
		}, func(s *Session) {
			AssertNoError(t, s.With("test.collection", func(r res.Resource) {
				r.RemoveEvent(l.Idx)
			}))
			s.GetMsg(t).AssertPayload(t, l.Expected).AssertSubject(t, "event.test.collection.remove")
		})
	}
}

// Test RemoveEvent panics if the resource is an untyped resource.
func TestRemoveEventPanicsOnUntypedResource(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("bar",
			res.Call("method", func(r res.CallRequest) {
				panicked := true
				defer func() {
					if !panicked {
						t.Errorf("expected RemoveEvent to panic, but nothing happened")
					}
				}()
				r.RemoveEvent(0)
				panicked = false
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.bar.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test RemoveEvent panics if the resource is an untyped resource, using With
func TestRemoveEventPanicsOnUntypedResourceUsingWith(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("bar")
	}, func(s *Session, done func()) {
		s.With("test.bar", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected RemoveEvent to panic, but nothing happened")
				}
				done()
			}()
			r.RemoveEvent(0)
		})
	})
}

// Test RemoveEvent panics if the resource is a model
func TestRemoveEventPanicsOnModel(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.NotFound()
			}),
			res.Call("method", func(r res.CallRequest) {
				panicked := true
				defer func() {
					if !panicked {
						t.Errorf("expected RemoveEvent to panic, but nothing happened")
					}
				}()
				r.RemoveEvent(0)
				panicked = false
				r.OK(nil)
			}),
		)
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test RemoveEvent panics if the resource is a model, using With
func TestRemoveEventPanicsOnModelUsingWith(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("model", res.GetModel(func(r res.ModelRequest) {
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		s.With("test.model", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected RemoveEvent to panic, but nothing happened")
				}
				done()
			}()
			r.RemoveEvent(0)
		})
	})
}

// Test RemoveEvent panics if idx is less than zero
func TestRemoveEventPanicsOnIdxLessThanZero(t *testing.T) {
	runTestAsync(t, func(s *Session) {
		s.Handle("collection", res.GetCollection(func(r res.CollectionRequest) {
			r.NotFound()
		}))
	}, func(s *Session, done func()) {
		s.With("test.collection", func(r res.Resource) {
			defer func() {
				v := recover()
				if v == nil {
					t.Errorf("expected RemoveEvent to panic, but nothing happened")
				}
				done()
			}()
			r.RemoveEvent(-1)
		})
	})
}

// Test ReaccessEvent sends a reaccess event.
func TestReaccessEvent(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model", res.Call("method", func(r res.CallRequest) {
			r.ReaccessEvent()
			r.OK(nil)
		}))
	}, func(s *Session) {
		inb := s.Request("call.test.model.method", nil)
		s.GetMsg(t).AssertSubject(t, "event.test.model.reaccess").AssertPayload(t, nil)
		s.GetMsg(t).AssertSubject(t, inb)
	})
}

// Test ReaccessEvent sends a reaccess event, using With.
func TestReaccessEventUsingWith(t *testing.T) {
	runTest(t, func(s *Session) {
		s.Handle("model")
	}, func(s *Session) {
		AssertNoError(t, s.With("test.model", func(r res.Resource) {
			r.ReaccessEvent()
		}))
		s.GetMsg(t).AssertSubject(t, "event.test.model.reaccess").AssertPayload(t, nil)
	})
}
