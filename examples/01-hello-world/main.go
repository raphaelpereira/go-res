package main

import res "github.com/jirenius/go-res"

func main() {
	s := res.NewService("example")
	s.Handle("model",
		res.Access(res.AccessGranted),
		res.GetModel(func(r res.ModelRequest) {
			r.Model(map[string]string{
				"message": "Hello, World!",
			})
		}),
	)
	s.ListenAndServe("nats://localhost:4222")
}
