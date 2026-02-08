package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*
var webFS embed.FS

type server struct {
	wordbooksDir string
	staticFS     fs.FS
	configPath   string
}

type wordbookListResponse struct {
	Wordbooks []string `json:"wordbooks"`
}

type wordbookWordsResponse struct {
	Name  string   `json:"name"`
	Words []string `json:"words"`
}

type settingsResponse struct {
	Accent   string `json:"accent"`
	Wordbook string `json:"wordbook"`
}

type settingsAccentRequest struct {
	Accent string `json:"accent"`
}

type settingsWordbookRequest struct {
	Wordbook string `json:"wordbook"`
}

func main() {
	var wordbooksDir string
	var host string
	var port int
	var openBrowser bool

	flag.StringVar(&wordbooksDir, "wordbooks-dir", "", "Directory containing .txt wordbook files")
	flag.StringVar(&host, "host", "127.0.0.1", "HTTP host")
	flag.IntVar(&port, "port", 8080, "HTTP port")
	flag.BoolVar(&openBrowser, "open-browser", false, "Open browser on startup")
	flag.Parse()

	if len(os.Args) == 1 {
		cfg, cfgPath, err := loadDefaultConfig()
		if err != nil {
			log.Fatalf("failed to load default config %q: %v", cfgPath, err)
		}
		if wordbooksDir == "" {
			wordbooksDir = cfg.WordbooksDir
		}
		if host == "127.0.0.1" && cfg.Host != "" {
			host = cfg.Host
		}
		if port == 8080 && cfg.Port != 0 {
			port = cfg.Port
		}
		openBrowser = cfg.OpenBrowser
	}

	if strings.TrimSpace(wordbooksDir) == "" {
		log.Fatal("missing required parameter: --wordbooks-dir (or WORDS_RAIN_WORDBOOKS_DIR in default config)")
	}

	if err := ensureDirExists(wordbooksDir); err != nil {
		log.Fatalf("invalid wordbooks directory: %v", err)
	}

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("failed to load static files: %v", err)
	}

	configPath, err := defaultConfigPath()
	if err != nil {
		log.Fatalf("failed to resolve config path: %v", err)
	}

	s := &server{
		wordbooksDir: wordbooksDir,
		staticFS:     staticFS,
		configPath:   configPath,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/wordbooks", s.handleWordbooks)
	mux.HandleFunc("/api/wordbooks/", s.handleWordbookWords)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/settings/accent", s.handleSettingsAccent)
	mux.HandleFunc("/api/settings/wordbook", s.handleSettingsWordbook)
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("serving on http://%s", addr)
	if openBrowser {
		url := fmt.Sprintf("http://%s:%d", browserHost(host), port)
		go func() {
			time.Sleep(250 * time.Millisecond)
			if err := openBrowserURL(url); err != nil {
				log.Printf("failed to open browser: %v", err)
			}
		}()
	}
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

func (s *server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := loadConfigOptional(s.configPath)
	if err != nil {
		http.Error(w, "failed to read settings", http.StatusInternalServerError)
		return
	}

	accent := strings.TrimSpace(cfg.Accent)
	if accent == "" {
		accent = "en-US"
	}
	writeJSON(w, settingsResponse{
		Accent:   accent,
		Wordbook: strings.TrimSpace(cfg.Wordbook),
	})
}

func (s *server) handleSettingsAccent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req settingsAccentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	accent := strings.TrimSpace(req.Accent)
	if accent != "en-US" && accent != "en-GB" {
		http.Error(w, "invalid accent", http.StatusBadRequest)
		return
	}

	cfg, err := loadConfigOptional(s.configPath)
	if err != nil {
		http.Error(w, "failed to read settings", http.StatusInternalServerError)
		return
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.WordbooksDir == "" {
		cfg.WordbooksDir = s.wordbooksDir
	}
	if cfg.Host == "127.0.0.1" && cfg.Port == 8080 && cfg.WordbooksDir == s.wordbooksDir && cfg.Accent == "" && !cfg.OpenBrowser {
		cfg.OpenBrowser = true
	}
	cfg.Accent = accent

	if err := writeConfig(s.configPath, cfg); err != nil {
		http.Error(w, "failed to write settings", http.StatusInternalServerError)
		return
	}

	writeJSON(w, settingsResponse{
		Accent:   cfg.Accent,
		Wordbook: cfg.Wordbook,
	})
}

func (s *server) handleSettingsWordbook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req settingsWordbookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	wordbook := strings.TrimSpace(req.Wordbook)
	if wordbook == "" || strings.Contains(wordbook, "/") || strings.Contains(wordbook, "\\") {
		http.Error(w, "invalid wordbook", http.StatusBadRequest)
		return
	}

	cfg, err := loadConfigOptional(s.configPath)
	if err != nil {
		http.Error(w, "failed to read settings", http.StatusInternalServerError)
		return
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.WordbooksDir == "" {
		cfg.WordbooksDir = s.wordbooksDir
	}
	if cfg.Host == "127.0.0.1" && cfg.Port == 8080 && cfg.WordbooksDir == s.wordbooksDir && cfg.Accent == "" && cfg.Wordbook == "" && !cfg.OpenBrowser {
		cfg.OpenBrowser = true
	}
	if cfg.Accent == "" {
		cfg.Accent = "en-US"
	}
	cfg.Wordbook = wordbook

	if err := writeConfig(s.configPath, cfg); err != nil {
		http.Error(w, "failed to write settings", http.StatusInternalServerError)
		return
	}

	writeJSON(w, settingsResponse{
		Accent:   cfg.Accent,
		Wordbook: cfg.Wordbook,
	})
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

type appConfig struct {
	Host         string
	Port         int
	WordbooksDir string
	OpenBrowser  bool
	Accent       string
	Wordbook     string
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user home: %w", err)
	}
	return filepath.Join(home, ".config", "words-rain", "config.env"), nil
}

func loadDefaultConfig() (appConfig, string, error) {
	path, err := defaultConfigPath()
	if err != nil {
		return appConfig{}, "", err
	}
	cfg, err := parseEnvConfig(path)
	return cfg, path, err
}

func loadConfigOptional(path string) (appConfig, error) {
	cfg, err := parseEnvConfig(path)
	if err != nil {
		if os.IsNotExist(err) {
			return appConfig{}, nil
		}
		return appConfig{}, err
	}
	return cfg, nil
}

func parseEnvConfig(path string) (appConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return appConfig{}, err
	}
	defer file.Close()

	cfg := appConfig{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return appConfig{}, fmt.Errorf("invalid line %d: expected KEY=VALUE", lineNo)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "WORDS_RAIN_HOST":
			cfg.Host = value
		case "WORDS_RAIN_PORT":
			p, err := strconv.Atoi(value)
			if err != nil {
				return appConfig{}, fmt.Errorf("invalid WORDS_RAIN_PORT at line %d: %w", lineNo, err)
			}
			cfg.Port = p
		case "WORDS_RAIN_WORDBOOKS_DIR":
			cfg.WordbooksDir = value
		case "WORDS_RAIN_OPEN_BROWSER":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return appConfig{}, fmt.Errorf("invalid WORDS_RAIN_OPEN_BROWSER at line %d: %w", lineNo, err)
			}
			cfg.OpenBrowser = b
		case "WORDS_RAIN_ACCENT":
			cfg.Accent = value
		case "WORDS_RAIN_WORDBOOK":
			cfg.Wordbook = value
		}
	}
	if err := scanner.Err(); err != nil {
		return appConfig{}, err
	}
	return cfg, nil
}

func writeConfig(path string, cfg appConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content := strings.Join([]string{
		"# words-rain default config",
		fmt.Sprintf("WORDS_RAIN_HOST=%s", cfg.Host),
		fmt.Sprintf("WORDS_RAIN_PORT=%d", cfg.Port),
		fmt.Sprintf("WORDS_RAIN_OPEN_BROWSER=%t", cfg.OpenBrowser),
		fmt.Sprintf("WORDS_RAIN_WORDBOOKS_DIR=%s", cfg.WordbooksDir),
		fmt.Sprintf("WORDS_RAIN_ACCENT=%s", cfg.Accent),
		fmt.Sprintf("WORDS_RAIN_WORDBOOK=%s", cfg.Wordbook),
		"",
	}, "\n")
	return os.WriteFile(path, []byte(content), 0o644)
}

func browserHost(host string) string {
	h := strings.TrimSpace(host)
	if h == "" || h == "0.0.0.0" || h == "::" {
		return "127.0.0.1"
	}
	if ip := net.ParseIP(h); ip != nil && ip.IsUnspecified() {
		return "127.0.0.1"
	}
	return h
}

func openBrowserURL(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}
