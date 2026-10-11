package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
    notes []Note
    nextID = 1
    mu sync.Mutex
    filename = "notes.json"
)

type Note struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

func loadNotes(filename string) ([]Note, error) {
    var notes []Note
	data, err := os.ReadFile(filename)
	if err != nil {
        if os.IsNotExist(err) {
            return []Note{}, nil
        }
		return nil, err
	}
   
    
    err = json.Unmarshal(data, &notes)
    if err != nil {
        return nil, err
    }
    return notes, nil
}

func saveNotes(filename string, notes []Note) error {
    data, err := json.MarshalIndent(notes, "", " ")
    if err != nil {
        return err
    }

    err = os.WriteFile(filename, data, 0644)
    if err != nil {
        return err
    }

    return nil 
}

func postHandler(w http.ResponseWriter, r *http.Request) {
    var note Note
    err := json.NewDecoder(r.Body).Decode(&note)
    if err != nil {
        http.Error(w, "errors decode json", http.StatusBadRequest)
        return
    }

    if note.Title == "" {
        http.Error(w, "errors: title name is empty", http.StatusBadRequest)
        return
    }  

    mu.Lock()
    defer mu.Unlock()

    note.ID = nextID
    nextID++
    note.CreatedAt = time.Now()
    notes = append(notes, note)

    err = saveNotes(filename, notes)
    if err != nil {
        http.Error(w, "save error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    errJs := json.NewEncoder(w).Encode(note)
    if errJs != nil {
        http.Error(w, "errors encode json", http.StatusInternalServerError)
        return
    }
}

func getHandler(w http.ResponseWriter, r *http.Request) {
    mu.Lock()
    defer mu.Unlock()
    tag := r.URL.Query().Get("tag")
    results := []Note{}

    if tag == "" {
        results = notes
    }else {
        for _, note := range notes {
            if slices.Contains(note.Tags, tag) {
                results = append(results, note)
            }
        }
    }

    w.Header().Set("Content-Type", "application/json")
    err := json.NewEncoder(w).Encode(results)
    if err != nil {
        http.Error(w, "errors get notes", http.StatusInternalServerError)
        return
    }
}

func getHand(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        http.Error(w, "errors convert id", http.StatusBadRequest)
        return
    }

    mu.Lock()
    defer mu.Unlock()

    found := false
    var result Note
    for _, note := range notes {
        if note.ID == id {
            found = true
            result = note
            break
        }
    }
    if !found {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    
    

    w.Header().Set("Content-Type", "application/json")
    errJs := json.NewEncoder(w).Encode(result)
    if errJs != nil {
        http.Error(w, "errors json encoding", http.StatusInternalServerError)
        return
    } 
}

func putHandler(w http.ResponseWriter, r *http.Request) {
    var updated Note
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    errRead := json.NewDecoder(r.Body).Decode(&updated)
    if errRead != nil {
        http.Error(w, "errors decode json", http.StatusBadRequest)
        return
    }

   if updated.Title == "" {
        http.Error(w, "title required", http.StatusBadRequest)
        return
    }

    mu.Lock()
    defer mu.Unlock()

    found := false
    for i := range notes {
        if notes[i].ID == id {
            notes[i].Title = updated.Title
            notes[i].Content = updated.Content
            notes[i].Tags = updated.Tags
            updated = notes[i]
            found = true
            break
        }
    }

    if !found {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    err = saveNotes(filename, notes)
    if err != nil {
        http.Error(w, "errors saving file", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    errJs := json.NewEncoder(w).Encode(updated)
    if errJs != nil {
        http.Error(w, "errors encode json", http.StatusInternalServerError)
        return
    }
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        http.Error(w, "error get id", http.StatusBadRequest)
        return
    }

    mu.Lock()
    defer mu.Unlock()
 
    found := false
    for i := 0; i < len(notes); i++ {
        if notes[i].ID == id {
            found = true
            notes = append(notes[:i], notes[i+1:]...)
            break
        }
    }

    if !found {
        http.Error(w, "not found id", http.StatusNotFound)
        return
    }

    err = saveNotes(filename, notes)
    if err != nil {
        http.Error(w, "save errors", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    errJs := json.NewEncoder(w).Encode(map[string]int{"deleted": id})
    if errJs != nil {
        http.Error(w, "json encode errors", http.StatusInternalServerError)
        return
    }
} 

func searchHandler(w http.ResponseWriter, r *http.Request) {
    qUrl := r.URL.Query().Get("q")
    if qUrl == "" {
        http.Error(w, "invalid", http.StatusBadRequest)
        return
    }
    mu.Lock()
    defer mu.Unlock()

    result :=[]Note{}
    for _, note := range notes {
        if strings.Contains(note.Title, qUrl) || strings.Contains(note.Content, qUrl) {
            result = append(result, note)
        }
    } 

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        fmt.Println(r.Method, r.URL.Path, "took", time.Since(start))
    })
}

func main() {
    var err error
    mux := http.NewServeMux()
    notes, err = loadNotes(filename)
    if err != nil {
        fmt.Println("error load notes", err)
        return
    }
    for _, n := range notes {
        if n.ID >= nextID {
            nextID = n.ID + 1
        }
    }

    mux.HandleFunc("POST /notes", postHandler)
    mux.HandleFunc("GET /notes", getHandler) 
    mux.HandleFunc("GET /notes/search", searchHandler)
    mux.HandleFunc("GET /notes/{id}", getHand)
    mux.HandleFunc("PUT /notes/{id}", putHandler)
    mux.HandleFunc("DELETE /notes/{id}", deleteHandler)
    
    err = http.ListenAndServe(":8080", logging(mux))
    if err != nil {
        fmt.Println("Error listen serve")
        return
    }
}