package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
)

var (
	port        = flag.Int("p", 13377, "Port to serve on")
	dir         = flag.String("dir", ".", "Directory to serve")
	readOnly    = flag.Bool("read-only", false, "Enable read-only mode")
	uploadOnly  = flag.Bool("upload-only", false, "Enable upload-only mode")
	authCreds   = flag.String("auth", "", "Enable basic auth in the format user:pass")
	runAsUser   = flag.String("user", "", "Drop privileges to this UNIX user")
	logFilePath = flag.String("log", "", "Path to log file")
)

//go:embed static/* templates/*
var embeddedFiles embed.FS

func main() {
	flag.Parse()

	// logging
	if *logFilePath != "" {
		f, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Unable to open log file: %v", err)
		}
		log.SetOutput(f)
	}

	// drop privileges
	if *runAsUser != "" {
		u, err := user.Lookup(*runAsUser)
		if err != nil {
			log.Fatalf("Failed to find user: %v", err)
		}
		if err := dropPrivileges(u.Uid, u.Gid); err != nil {
			log.Fatalf("Failed to drop privileges: %v", err)
		}
	}

	// main handler
	h := &kuttaHandler{
		Dir:          filepath.Clean(*dir),
		ReadOnly:     *readOnly,
		UploadOnly:   *uploadOnly,
		AuthEnabled:  *authCreds != "",
		AuthCreds:    *authCreds,
		FS:           embeddedFiles,
		UploadedOnly: *port == 13377, // only show uploads in default port mode
	}

	// If default port, don't show existing dir
	if *port == 13377 {
		h.Dir = "." // base dir still exists, but listing will be filtered in handlers.go
	}

	h.RegisterRoutes()

	// serve static files correctly
	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalf("Failed to load static FS: %v", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Kuttañ serving payloads on http://localhost%s (dir: %s)", addr, *dir)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
