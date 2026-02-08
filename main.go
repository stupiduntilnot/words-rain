package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed web/*
var webFS embed.FS

type server struct {
	wordbooksDir string
	staticFS     fs.FS
}

type wordbookListResponse struct {
	Wordbooks []string `json:"wordbooks"`
}

type wordbookWordsResponse struct {
	Name  string   `json:"name"`
	Words []string `json:"words"`
}

func main() {
	var wordbooksDir string
	var host string
	var port int

	flag.StringVar(&wordbooksDir, "wordbooks-dir", "", "Directory containing .txt wordbook files")
	flag.StringVar(&host, "host", "127.0.0.1", "HTTP host")
	flag.IntVar(&port, "port", 8080, "HTTP port")
	flag.Parse()

	if strings.TrimSpace(wordbooksDir) == "" {
		log.Fatal("--wordbooks-dir is required")
	}

	if err := ensureDirExists(wordbooksDir); err != nil {
		log.Fatalf("invalid --wordbooks-dir: %v", err)
	}

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("failed to load static files: %v", err)
	}

	s := &server{
		wordbooksDir: wordbooksDir,
		staticFS:     staticFS,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/wordbooks", s.handleWordbooks)
	mux.HandleFunc("/api/wordbooks/", s.handleWordbookWords)
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("serving on http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func ensureDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func (s *server) handleWordbooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	books, err := listWordbooks(s.wordbooksDir)
	if err != nil {
		http.Error(w, "failed to list wordbooks", http.StatusInternalServerError)
		return
	}

	writeJSON(w, wordbookListResponse{Wordbooks: books})
}

func (s *server) handleWordbookWords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawName := strings.TrimPrefix(r.URL.Path, "/api/wordbooks/")
	name, err := url.PathUnescape(rawName)
	if err != nil {
		http.Error(w, "invalid wordbook name", http.StatusBadRequest)
		return
	}
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		http.Error(w, "invalid wordbook name", http.StatusBadRequest)
		return
	}

	words, err := readWordbook(filepath.Join(s.wordbooksDir, name+".txt"))
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "wordbook not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to read wordbook", http.StatusInternalServerError)
		return
	}

	writeJSON(w, wordbookWordsResponse{Name: name, Words: words})
}

func listWordbooks(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	books := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".txt") {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		base = strings.TrimSpace(base)
		if base == "" {
			continue
		}
		books = append(books, base)
	}

	sort.Strings(books)
	return books, nil
}

func readWordbook(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		w := strings.TrimSpace(strings.ToLower(line))
		if w == "" {
			continue
		}
		words = append(words, w)
	}
	return words, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode json", http.StatusInternalServerError)
	}
}
