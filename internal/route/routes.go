package route

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/imartingraham/todobin/internal/model"
	"github.com/imartingraham/todobin/internal/util"
	"github.com/microcosm-cc/bluemonday"
)

var tpl = template.Must(template.ParseGlob("web/template/*.html"))

// HandleIndex is the route for "/"
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	var tplvars struct {
		CSRFToken string
		Name      string
		Todo      string
	}

	switch r.Method {
	case "GET":
		tplvars.CSRFToken = csrf.Token(r)

	case "POST":
		// Disallow all html tags
		p := bluemonday.StrictPolicy()

		if err := r.ParseForm(); err != nil {
			util.Airbrake.Notify(fmt.Errorf("ParseForm() err: %w", err), r)
			return
		}
		todoList := &model.TodoList{
			Name: p.Sanitize(r.FormValue("name")),
		}

		tplvars.Todo = strings.TrimSpace(r.FormValue("todolist"))
		rawTodos := strings.Split(tplvars.Todo, "\n")

		for _, t := range rawTodos {
			if strings.HasPrefix(t, "-") {
				t = strings.Replace(t, "-", "", 1)
			}
			var important bool
			if strings.HasPrefix(t, "!") {
				important = true
				t = strings.Replace(t, "!", "", 1)
			}

			// When I sanitize the string before splitting
			// it doesn't work, so for now I'm just sanitizing
			// each line
			todoList.Todos = append(todoList.Todos, &model.Todo{
				Todo:      p.Sanitize(strings.TrimSpace(t)),
				Important: important,
			})
		}

		err := todoList.Save()
		if err != nil {
			util.Airbrake.Notify(fmt.Errorf("Failed to save todo list: %w", err), r)
			panic(err)
		}

		u := "/todo/" + todoList.ID
		http.Redirect(w, r, u, 301)
		return
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
	err := tpl.ExecuteTemplate(w, "index.html", tplvars)
	if err != nil {
		util.Airbrake.Notify(fmt.Errorf("failed to execute index.html: %w", err), r)
		log.Fatalf("[error] failed to execute index.html: %v\n", err)
	}
}

// HandleTodos is the route for "/todo/[uuid]"
func HandleTodos(w http.ResponseWriter, r *http.Request) {

	var tplvars struct {
		CSRFToken string
		TodoList  *model.TodoList
	}

	tplvars.CSRFToken = csrf.Token(r)
	vars := mux.Vars(r)
	listID, ok := vars["listId"]
	if !ok {
		log.Println("[error] listId not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	list, err := model.TodoListByID(listID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	tplvars.TodoList = list
	err = tpl.ExecuteTemplate(w, "todo.html", tplvars)
	if err != nil {
		util.Airbrake.Notify(fmt.Errorf("Failed to execute todo.html: %w", err), nil)
		log.Fatalf("[error] Failed to execute todo.html: %v\n", err)
	}
}

// HandleTodoDone marks todo as done or undone
func HandleTodoDone(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.WriteHeader(http.StatusForbidden)
		util.Airbrake.Notify(errors.New("Route only handles PUT requests"), r)
		log.Fatalf("[error] Route only handles PUT requests")
		return
	}

	vars := mux.Vars(r)
	listID := vars["listId"]
	todoID := vars["todoId"]

	t, err := model.TodoByID(listID, todoID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		util.Airbrake.Notify(fmt.Errorf("Failed to fetch todo: %w", err), r)
		log.Fatalf("[error] failed to fetch todo: %v\n", err)
	}

	err = t.ToggleDone()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		util.Airbrake.Notify(fmt.Errorf("Failed to toggle done for todo: %w", err), r)
		log.Fatalf("[error] failed to toggle done for todo: %v\n", err)
	}

	jsonData, err := json.Marshal(t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		util.Airbrake.Notify(fmt.Errorf("Failed to json encode todo: %w", err), r)
		log.Fatalf("[error] could not json encode todo: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func HandleTodoDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		w.WriteHeader(http.StatusForbidden)
		util.Airbrake.Notify(errors.New("Route only handles PUT requests"), r)
		log.Fatalf("[error] Route only handles PUT requests")
		return
	}

	vars := mux.Vars(r)
	listID := vars["listId"]
	todoID := vars["todoId"]

	t, err := model.TodoByID(listID, todoID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		util.Airbrake.Notify(fmt.Errorf("Failed to fetch todo: %w", err), r)
		log.Fatalf("[error] failed to fetch todo: %v\n", err)
	}

	err = t.Delete()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		util.Airbrake.Notify(fmt.Errorf("Failed to toggle done for todo: %w", err), r)
		log.Fatalf("[error] failed to toggle done for todo: %v\n", err)
	}

	jsonData, err := json.Marshal(t)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		util.Airbrake.Notify(fmt.Errorf("Failed to json encode todo: %w", err), r)
		log.Fatalf("[error] could not json encode todo: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
