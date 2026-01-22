package main

import (
	"log"
	"net/http"

	"shebang.run/internal/api"
	"shebang.run/internal/config"
	"shebang.run/internal/crypto"
	"shebang.run/internal/database"
	"shebang.run/internal/kms"
	"shebang.run/internal/middleware"
	"shebang.run/internal/storage"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	var store storage.Storage
	if cfg.StorageType == "s3" {
		store, err = storage.NewS3Storage(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, false)
	} else {
		store, err = storage.NewLocalStorage(cfg.LocalStoragePath)
	}
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize KMS
	var keyManager kms.KeyManager
	if cfg.MasterKeySource == "env" {
		keyManager, err = kms.NewEnvKeyManager(cfg.MasterKeyEnv)
		if err != nil {
			log.Printf("Warning: KMS not initialized: %v", err)
			log.Printf("Server-side encryption features will be disabled")
		}
	}

	// Initialize UDEK manager
	var udekManager *crypto.UDEKManager
	if keyManager != nil {
		udekManager = crypto.NewUDEKManager(db.DB, keyManager)
	}

	authHandler := api.NewAuthHandler(db, cfg)
	keyHandler := api.NewKeyHandler(db)
	scriptHandler := api.NewScriptHandler(db, store, cfg)
	publicHandler := api.NewPublicHandler(db, store, cfg)
	adminHandler := api.NewAdminHandler(db, cfg)
	accountHandler := api.NewAccountHandler(db, cfg)
	setupHandler := api.NewSetupHandler(db)
	communityHandler := api.NewCommunityHandler(db)
	webHandler := api.NewWebHandler()
	
	var secretsHandler *api.SecretsHandler
	if udekManager != nil {
		secretsHandler = api.NewSecretsHandler(db.DB, udekManager)
	}

	r := chi.NewRouter()
	
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RateLimitMiddleware(cfg.DefaultRateLimit))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Serve static files
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	
	// Serve OpenAPI spec
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "openapi.yaml")
	})

	// Setup check middleware - redirect to setup if needed
	setupCheck := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip setup check for setup page and API endpoints
			if r.URL.Path == "/setup" || r.URL.Path == "/api/setup/status" || r.URL.Path == "/api/auth/register" {
				next.ServeHTTP(w, r)
				return
			}

			isFirst, err := db.IsFirstUser()
			if err == nil && isFirst {
				http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	r.Get("/setup", webHandler.Setup)
	r.Get("/api/setup/status", setupHandler.Status)

	r.Group(func(r chi.Router) {
		r.Use(setupCheck)

		r.Get("/", webHandler.Index)
		r.Get("/login", webHandler.Login)
		r.Get("/register", webHandler.Register)
		r.Get("/dashboard", webHandler.Dashboard)
		r.Get("/community", webHandler.Community)
		r.Get("/keys", webHandler.Keys)
		r.Get("/secrets", webHandler.Secrets)
		r.Get("/select-username", webHandler.SelectUsername)
		r.Get("/account", webHandler.Account)
		r.Get("/script-editor", webHandler.ScriptEditor)
		r.Get("/privacy", webHandler.Privacy)
		r.Get("/gdpr", webHandler.GDPR)
		r.Get("/docs", webHandler.Docs)
		r.Get("/admin", webHandler.Admin)
		r.Get("/terms", webHandler.Terms)
		r.Get("/api-reference", webHandler.APIReference)
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Get("/check-username", authHandler.CheckUsername)
		r.With(middleware.AuthMiddleware(cfg.JWTSecret, db)).Post("/set-username", authHandler.SetUsername)
		r.Get("/oauth/github", func(w http.ResponseWriter, r *http.Request) {
			authHandler.OAuthLogin(w, r, "github")
		})
		r.Get("/oauth/github/callback", func(w http.ResponseWriter, r *http.Request) {
			authHandler.OAuthCallback(w, r, "github")
		})
		r.Get("/oauth/google", func(w http.ResponseWriter, r *http.Request) {
			authHandler.OAuthLogin(w, r, "google")
		})
		r.Get("/oauth/google/callback", func(w http.ResponseWriter, r *http.Request) {
			authHandler.OAuthCallback(w, r, "google")
		})
	})

	r.Route("/api/keys", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret, db))
		r.Get("/", keyHandler.List)
		r.Post("/generate", keyHandler.Generate)
		r.Post("/import", keyHandler.Import)
		r.Delete("/{id}", keyHandler.Delete)
	})

	r.Route("/api/scripts", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret, db))
		r.Get("/", scriptHandler.List)
		r.Post("/", scriptHandler.Create)
		r.Get("/{id}", scriptHandler.Get)
		r.Get("/{id}/encrypted", scriptHandler.GetEncryptedContent)
		r.Put("/{id}", scriptHandler.Update)
		r.Delete("/{id}", scriptHandler.Delete)
		r.Post("/{id}/share", scriptHandler.GenerateShareToken)
		r.Delete("/{id}/share/{token}", scriptHandler.RevokeShareToken)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret, db))
		r.Use(middleware.AdminMiddleware)
		r.Get("/users", adminHandler.ListUsers)
		r.Put("/users/{id}/limits", adminHandler.SetUserLimits)
		r.Put("/users/{id}/password", adminHandler.ResetUserPassword)
		r.Delete("/users/{id}", adminHandler.DeleteUser)
		r.Get("/config", adminHandler.GetConfig)
	})

	r.Route("/api/account", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret, db))
		r.Put("/password", accountHandler.ChangePassword)
		r.Get("/export", accountHandler.ExportData)
		r.Delete("/", accountHandler.DeleteAccount)
		r.Get("/tokens", accountHandler.ListAPITokens)
		r.Post("/tokens", accountHandler.CreateAPIToken)
		r.Delete("/tokens/{id}", accountHandler.DeleteAPIToken)
	})

	r.Route("/api/community", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret, db))
		r.Get("/scripts", communityHandler.ListPublicScripts)
	})

	// Secrets management
	if secretsHandler != nil {
		r.Route("/api/secrets", func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(cfg.JWTSecret, db))
			r.Get("/", secretsHandler.List)
			r.Post("/", secretsHandler.Create)
			r.Get("/{name}/value", secretsHandler.GetValue)
			r.Delete("/{name}", secretsHandler.Delete)
			r.Get("/{name}/audit", secretsHandler.GetAuditLog)
		})
	}

	r.Get("/{username}/{script}", publicHandler.GetScript)
	r.Get("/{username}/{script}/meta", publicHandler.GetMetadata)
	r.Get("/{username}/{script}/verify", publicHandler.VerifySignature)

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := http.ListenAndServe(":"+cfg.ServerPort, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
