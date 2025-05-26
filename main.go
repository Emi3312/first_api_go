package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

var (
	items   []Item
	nextID  = 1
	mu      sync.Mutex // protege el slice
	clients = make(map[chan string]bool)
	clMu    sync.Mutex // protege el map de clients
)

// broadcaster envía msg a todos los clientes SSE
func broadcast(msg string) {
	clMu.Lock()
	for ch := range clients {
		select {
		case ch <- msg:
		default:
			// si el cliente está saturado, lo eliminamos
			delete(clients, ch)
			close(ch)
		}
	}
	clMu.Unlock()
}

func main() {
	// datos iniciales
	items = append(items, Item{ID: nextID, Name: "Lapicera", Price: 10})
	nextID++
	items = append(items, Item{ID: nextID, Name: "Cuaderno", Price: 50})
	nextID++

	r := mux.NewRouter()
	r.HandleFunc("/ping", pingHandler).Methods("GET")
	r.HandleFunc("/items", getItemsHandler).Methods("GET")
	r.HandleFunc("/items/{id}", getItemHandler).Methods("GET")
	r.HandleFunc("/items", createItemHandler).Methods("POST")
	r.HandleFunc("/items/{id}", updateItemHandler).Methods("PUT")
	r.HandleFunc("/items/{id}", deleteItemHandler).Methods("DELETE")
	r.HandleFunc("/events", eventsHandler).Methods("GET")

	log.Println("Servidor escuchando en puerto 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func getItemsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	json.NewEncoder(w).Encode(items)
}

func getItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	mu.Lock()
	defer mu.Unlock()
	for _, it := range items {
		if it.ID == id {
			json.NewEncoder(w).Encode(it)
			return
		}
	}
	http.Error(w, "Ítem no encontrado", http.StatusNotFound)
}

func createItemHandler(w http.ResponseWriter, r *http.Request) {
	var newItem Item
	if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	mu.Lock()
	newItem.ID = nextID
	nextID++
	items = append(items, newItem)
	mu.Unlock()

	// emitir evento SSE
	data, _ := json.Marshal(struct {
		Action string `json:"action"`
		Item   Item   `json:"item"`
	}{"create", newItem})
	broadcast(string(data))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newItem)
}

func updateItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var updated Item
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for i, it := range items {
		if it.ID == id {
			updated.ID = id
			items[i] = updated

			data, _ := json.Marshal(struct {
				Action string `json:"action"`
				Item   Item   `json:"item"`
			}{"update", updated})
			broadcast(string(data))

			json.NewEncoder(w).Encode(updated)
			return
		}
	}
	http.Error(w, "Ítem no encontrado", http.StatusNotFound)
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	mu.Lock()
	defer mu.Unlock()
	for i, it := range items {
		if it.ID == id {
			// quitar de slice
			items = append(items[:i], items[i+1:]...)

			data, _ := json.Marshal(struct {
				Action string `json:"action"`
				ID     int    `json:"id"`
			}{"delete", id})
			broadcast(string(data))

			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	http.Error(w, "Ítem no encontrado", http.StatusNotFound)
}

// eventsHandler abre un stream SSE por cada cliente
func eventsHandler(w http.ResponseWriter, r *http.Request) {
	// cabeceras SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no soportado", http.StatusInternalServerError)
		return
	}

	// canal para este cliente
	msgCh := make(chan string)
	clMu.Lock()
	clients[msgCh] = true
	clMu.Unlock()

	// enviar un evento “init” con el estado actual
	mu.Lock()
	initData, _ := json.Marshal(struct {
		Action string `json:"action"`
		Items  []Item `json:"items"`
	}{"init", items})
	mu.Unlock()
	fmt.Fprintf(w, "data: %s\n\n", initData)
	fl.Flush()

	// escuchar el canal y escribir en la respuesta
	notify := w.(http.CloseNotifier).CloseNotify()
	for {
		select {
		case <-notify:
			// cliente desconectado
			clMu.Lock()
			delete(clients, msgCh)
			clMu.Unlock()
			return
		case msg := <-msgCh:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			fl.Flush()
		}
	}
}
