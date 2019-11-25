package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/imartingraham/todobin/internal/route"
	"github.com/imartingraham/todobin/internal/util"
)

func main() {
	go route.ListenForWebsocketMessages()
	defer util.Airbrake.Close()
	defer util.Airbrake.NotifyOnPanic()

	r := mux.NewRouter()

	fs := http.FileServer(http.Dir("./web/public"))
	r.PathPrefix("/scripts/").Handler(fs)
	r.PathPrefix("/styles/").Handler(fs)
	r.PathPrefix("/images/").Handler(fs)
	r.HandleFunc("/todo/{listId}", route.HandleTodos)
	r.HandleFunc("/todo/{listId}/done/{todoId}", route.HandleTodoDone)
	r.HandleFunc("/todo/{listId}/delete/{todoId}", route.HandleTodoDelete)
	r.HandleFunc("/ws", route.HandleWs)
	r.HandleFunc("/", route.HandleIndex)

	http.Handle("/", r)

	port := util.GetEnvOr("PORT", "3000")
	fmt.Println("Ready and listening on " + port)
	p := csrf.Protect([]byte(os.Getenv("CSRF_TOKEN")))
	err := http.ListenAndServe(":"+port, p(r))
	if err != nil {
		util.Airbrake.Notify(fmt.Errorf("Failed to listen on port: %s", port), nil)
		panic(err)
	}
}
