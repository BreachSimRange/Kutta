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
	"strconv"
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

//go:embed templates/* static/*
var embeddedFiles embed.FS

func main() {
	flag.Parse()

	if *logFilePath != "" {
		f, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Unable to open log file: %v", err)
		}
		log.SetOutput(f)
	}


	chosenDir := filepath.Clean(*dir)
	if *port == 13377 {
		chosenDir = "."
	}

	uploadDir := filepath.Join(chosenDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		log.Fatalf("Failed to create upload dir %s: %v", uploadDir, err)
	}

	if *runAsUser != "" && os.Geteuid() == 0 {
		u, err := user.Lookup(*runAsUser)
		if err != nil {
			log.Fatalf("Failed to find user %s: %v", *runAsUser, err)
		}

		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)

		if err := os.Chown(uploadDir, uid, gid); err != nil {
			log.Fatalf("Failed to chown upload dir %s to %s: %v", uploadDir, *runAsUser, err)
		}
	}

	if *runAsUser != "" {
		u, err := user.Lookup(*runAsUser)
		if err != nil {
			log.Fatalf("Failed to find user: %v", err)
		}
		if err := dropPrivileges(u.Uid, u.Gid); err != nil {
			log.Fatalf("Failed to drop privileges: %v", err)
		}
	}

	h := &kuttaHandler{
		Dir:          chosenDir,
		ReadOnly:     *readOnly,
		UploadOnly:   *uploadOnly,
		AuthEnabled:  *authCreds != "",
		AuthCreds:    *authCreds,
		FS:           embeddedFiles,
		UploadedOnly: *port == 13377,
	}

	h.RegisterRoutes()

	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalf("Failed to load static FS: %v", err)
	}
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Kuttañ serving payloads on http://localhost%s (dir: %s)", addr, chosenDir)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
