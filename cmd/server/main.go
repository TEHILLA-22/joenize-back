package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/tehilla-22/b2b-api/internal/config"
	"github.com/tehilla-22/b2b-api/internal/database"
	"github.com/tehilla-22/b2b-api/internal/router"
)

func main() {
	loadEnv()

	cfg := config.Load()

	db := database.Connect(cfg)
	database.Migrate(db)
	database.Seed(db)

	r := router.New(cfg)

	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	port := os.Getenv("PORT")
	if port == "" {
		port = fmt.Sprintf("%d", cfg.AppPort)
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("%s starting on %s", cfg.AppName, addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func loadEnv() {
	candidates := []string{
		".env",
		"../.env",
		filepath.Join(os.Getenv("PWD"), ".env"),
	}
	for _, p := range candidates {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			godotenv.Load(abs)
			log.Printf("Loaded env from %s", abs)
			return
		}
	}
	log.Println("No .env file found, using defaults and system env")
}
