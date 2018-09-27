/*
This is an example of a simple Hello World RES service written in Go.
* It exposes a single resource: "exampleService.myModel".
* It allows setting the resource's Message property through the "set" method.

Visit https://github.com/jirenius/resgate#client for the matching client.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/jirenius/go-res"
)

// Model example
type Model struct {
	Message string `json:"message"`
}

var myModel = &Model{Message: "Hello Go World"}

func main() {
	// Create a new RES Service
	s := res.NewService("exampleService")

	// Add handlers for "exampleService.myModel" resource
	s.Handle("myModel",
		res.Access(res.AccessGranted),
		res.GetModel(func(r *res.Request, w *res.GetModelResponse) {
			w.Model(myModel)
		}),
		res.Call("set", func(r *res.Request, w *res.CallResponse) {
			var p struct {
				Message *string `json:"message,omitempty"`
			}
			r.UnmarshalParams(&p)

			// Check if the message property was changed
			if p.Message != nil && *p.Message != myModel.Message {
				// Update the model
				myModel.Message = *p.Message
				// Send a change event with updated fields
				r.ChangeEvent(map[string]interface{}{"message": p.Message})
			}

			// Send success response
			w.OK(nil)
		}),
	)

	// Start service in separate goroutine
	stop := make(chan bool)
	go func() {
		defer close(stop)
		err := s.ListenAndServe("nats://localhost:4222")
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	}()

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	go func() { log.Fatal(http.ListenAndServe(":8081", http.FileServer(http.Dir(path)))) }()
	fmt.Println("Client at: http://localhost:8081/")

	// Wait for interrupt signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		// Graceful stop
		s.Stop()
	case <-stop:
	}
}