package main

import (
    "encoding/json"
    "log"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
)

type Item struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Price int    `json:"price"`
}

var items []Item
var nextID = 1

func main() {
    // Poblamos con un par de ítems iniciales
    items = append(items, Item{ID: nextID, Name: "Lapicera", Price: 10})
    nextID++
    items = append(items, Item{ID: nextID, Name: "Cuaderno", Price: 50})
    nextID++

    r := mux.NewRouter()

    // Rutas
    r.HandleFunc("/ping", pingHandler).Methods("GET")
    r.HandleFunc("/items", getItemsHandler).Methods("GET")
    r.HandleFunc("/items/{id}", getItemHandler).Methods("GET")
    r.HandleFunc("/items", createItemHandler).Methods("POST")
    r.HandleFunc("/items/{id}", updateItemHandler).Methods("PUT")
    r.HandleFunc("/items/{id}", deleteItemHandler).Methods("DELETE")

    log.Println("Servidor escuchando en puerto 8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("pong"))
}

func getItemsHandler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(items)
}

func getItemHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "ID inválido", http.StatusBadRequest)
        return
    }
    for _, item := range items {
        if item.ID == id {
            json.NewEncoder(w).Encode(item)
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
    newItem.ID = nextID
    nextID++
    items = append(items, newItem)
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(newItem)
}

func updateItemHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "ID inválido", http.StatusBadRequest)
        return
    }
    var updated Item
    if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
        http.Error(w, "JSON inválido", http.StatusBadRequest)
        return
    }
    for i, item := range items {
        if item.ID == id {
            updated.ID = id
            items[i] = updated
            json.NewEncoder(w).Encode(updated)
            return
        }
    }
    http.Error(w, "Ítem no encontrado", http.StatusNotFound)
}

func deleteItemHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "ID inválido", http.StatusBadRequest)
        return
    }
    for i, item := range items {
        if item.ID == id {
            items = append(items[:i], items[i+1:]...)
            w.WriteHeader(http.StatusNoContent)
            return
        }
    }
    http.Error(w, "Ítem no encontrado", http.StatusNotFound)
}
