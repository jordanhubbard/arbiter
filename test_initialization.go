package main

import (
"context"
"fmt"
"log"
"time"

"github.com/jordanhubbard/arbiter/internal/arbiter"
"github.com/jordanhubbard/arbiter/pkg/config"
)

func main() {
fmt.Println("🧪 Testing Arbiter Initialization")
fmt.Println("=================================\n")

// Load config
fmt.Println("📄 Loading config.yaml...")
cfg, err := config.LoadConfig("config.yaml")
if err != nil {
log.Fatalf("❌ Failed to load config: %v", err)
}
fmt.Printf("✅ Config loaded successfully\n")
fmt.Printf("   - HTTP Port: %d\n", cfg.Server.HTTPPort)
fmt.Printf("   - Database: %s (%s)\n", cfg.Database.Type, cfg.Database.Path)
fmt.Printf("   - Beads path: %s\n", cfg.Beads.BDPath)
fmt.Printf("   - Persona path: %s\n", cfg.Agents.DefaultPersonaPath)
fmt.Printf("   - Projects configured: %d\n\n", len(cfg.Projects))

// Initialize arbiter
fmt.Println("🎯 Initializing Arbiter...")
arb := arbiter.New(cfg)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := arb.Initialize(ctx); err != nil {
log.Fatalf("❌ Failed to initialize arbiter: %v", err)
}
fmt.Println("✅ Arbiter initialized successfully\n")

// List projects
fmt.Println("📋 Loaded Projects:")
projects := arb.ListProjects()
if len(projects) == 0 {
fmt.Println("   (no projects loaded)")
}
for _, proj := range projects {
fmt.Printf("   ✅ %s (ID: %s)\n", proj.Name, proj.ID)
fmt.Printf("      Repo: %s\n", proj.GitRepo)
fmt.Printf("      Branch: %s\n", proj.Branch)
fmt.Printf("      Beads: %s\n", proj.BeadsPath)
if desc, ok := proj.Context["description"]; ok {
fmt.Printf("      Description: %s\n", desc)
}
}
fmt.Println()

// List personas
fmt.Println("🎭 Available Personas:")
personas, err := arb.ListPersonas()
if err != nil {
log.Fatalf("❌ Failed to list personas: %v", err)
}
if len(personas) == 0 {
fmt.Println("   (no personas found)")
}
for _, p := range personas {
fmt.Printf("   ✅ %s\n", p)
}
fmt.Println()

fmt.Println("✅ All initialization tests passed!")
fmt.Println("\n📊 Summary:")
fmt.Println("   - Configuration loaded and parsed")
fmt.Println("   - Arbiter initialized successfully")
fmt.Printf("   - %d project(s) registered\n", len(projects))
fmt.Printf("   - %d persona(s) available\n", len(personas))
fmt.Println("\n✨ Arbiter is ready to orchestrate!")
}
