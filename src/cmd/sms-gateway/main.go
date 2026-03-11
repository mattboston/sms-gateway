// @title SMS Gateway API
// @version 1.0
// @description REST API for sending and receiving SMS messages via a USB GSM modem.
// @license.name MIT

// @host localhost:5174
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token with Bearer prefix (e.g. "Bearer eyJhbG...")

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API key for programmatic access
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	smsgateway "github.com/mattboston/sms-gateway"
	"github.com/mattboston/sms-gateway/internal/api"
	"github.com/mattboston/sms-gateway/internal/auth"
	"github.com/mattboston/sms-gateway/internal/config"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/modem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sms-gateway",
		Short: "SMS Gateway - Send and receive SMS via a USB GSM modem",
	}

	// Persistent flags bound to viper.
	rootCmd.PersistentFlags().String("db-driver", "sqlite", "Database driver: sqlite or postgres")
	rootCmd.PersistentFlags().String("db-dsn", "/opt/sms-gateway/sms-gateway.db", "Database connection string")
	rootCmd.PersistentFlags().String("device-path", "", "Serial device path (e.g., /dev/ttyUSB0)")
	rootCmd.PersistentFlags().Int("baud-rate", 9600, "Serial baud rate")
	rootCmd.PersistentFlags().Int("port", 5174, "HTTP server port")
	rootCmd.PersistentFlags().Bool("dev-mode", false, "Enable development mode (mock modem, CORS)")
	rootCmd.PersistentFlags().String("jwt-secret", "", "JWT signing secret")
	rootCmd.PersistentFlags().String("config-file", "/opt/sms-gateway/sms-gateway.conf", "Path to config file")

	mustBindPFlag := func(key, flag string) {
		if err := viper.BindPFlag(key, rootCmd.PersistentFlags().Lookup(flag)); err != nil {
			log.Fatalf("binding flag %q: %v", flag, err)
		}
	}

	mustBindPFlag("db_driver", "db-driver")
	mustBindPFlag("db_dsn", "db-dsn")
	mustBindPFlag("device_path", "device-path")
	mustBindPFlag("baud_rate", "baud-rate")
	mustBindPFlag("port", "port")
	mustBindPFlag("dev_mode", "dev-mode")
	mustBindPFlag("jwt_secret", "jwt-secret")
	mustBindPFlag("config_file", "config-file")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(userCmd())
	rootCmd.AddCommand(apikeyCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Open database.
			db, err := database.New(cfg.DBDriver, cfg.DBDSN)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			// Auto-run migrations.
			log.Println("Running database migrations...")
			if err := database.RunMigrations(db, cfg.DBDriver, smsgateway.MigrationsFS); err != nil {
				return fmt.Errorf("running migrations: %w", err)
			}

			repo := database.NewRepository(db)

			// Seed default admin user if no users exist.
			hash, err := auth.HashPassword("admin123")
			if err != nil {
				return fmt.Errorf("hashing default password: %w", err)
			}
			seeded, err := repo.SeedDefaultAdmin(hash)
			if err != nil {
				return fmt.Errorf("seeding default admin: %w", err)
			}
			if seeded {
				log.Println("Created default admin user (username: admin, password: admin123)")
				log.Println("You will be required to change the password on first login.")
			}

			// Initialize modem.
			var m modem.Modem
			if cfg.DevMode {
				log.Println("Development mode enabled, using mock modem")
				m = modem.NewMockModem()
			} else {
				if cfg.DevicePath == "" {
					return fmt.Errorf("device-path is required when not in dev mode")
				}
				serialModem, err := modem.NewSerialModem(cfg.DevicePath, cfg.BaudRate)
				if err != nil {
					return fmt.Errorf("opening modem: %w", err)
				}
				m = serialModem
			}
			defer m.Close()

			// Start SMS receiver.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m.StartReceiver(ctx, func(from, body string) error {
				log.Printf("Received SMS from %s: %s", from, body)
				_, err := repo.CreateMessage("inbound", from, body, "received", nil)
				if err != nil {
					log.Printf("Error saving inbound SMS: %v", err)
					return err
				}
				return nil
			})

			// Build router.
			router := api.NewRouter(repo, m, cfg)

			// Start HTTP server.
			addr := fmt.Sprintf(":%d", cfg.Port)
			srv := &http.Server{
				Addr:         addr,
				Handler:      router,
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 15 * time.Second,
				IdleTimeout:  60 * time.Second,
			}

			// Graceful shutdown.
			done := make(chan os.Signal, 1)
			signal.Notify(done, os.Interrupt, syscall.SIGTERM)

			go func() {
				log.Printf("SMS Gateway listening on %s", addr)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("HTTP server error: %v", err)
				}
			}()

			<-done
			log.Println("Shutting down...")

			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("server shutdown: %w", err)
			}

			log.Println("Server stopped")
			return nil
		},
	}
}

// openDB loads config and opens a database connection.
func openDB() (*database.Repository, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	db, err := database.New(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}
	repo := database.NewRepository(db)
	cleanup := func() { db.Close() }
	return repo, cleanup, nil
}

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			username, _ := cmd.Flags().GetString("username")
			password, _ := cmd.Flags().GetString("password")
			isAdmin, _ := cmd.Flags().GetBool("admin")

			if username == "" || password == "" {
				return fmt.Errorf("--username and --password are required")
			}

			repo, cleanup, err := openDB()
			if err != nil {
				return err
			}
			defer cleanup()

			hash, err := auth.HashPassword(password)
			if err != nil {
				return fmt.Errorf("hashing password: %w", err)
			}

			user, err := repo.CreateUser(username, hash, isAdmin, false)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}

			fmt.Printf("User created:\n")
			fmt.Printf("  ID:       %s\n", user.ID)
			fmt.Printf("  Username: %s\n", user.Username)
			fmt.Printf("  Admin:    %t\n", user.IsAdmin)
			fmt.Printf("  Created:  %s\n", user.CreatedAt.Format(time.RFC3339))
			return nil
		},
	}
	createCmd.Flags().String("username", "", "Username for the new user")
	createCmd.Flags().String("password", "", "Password for the new user")
	createCmd.Flags().Bool("admin", false, "Grant admin privileges")

	cmd.AddCommand(createCmd)
	return cmd
}

func apikeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apikey",
		Short: "API key management commands",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			label, _ := cmd.Flags().GetString("label")
			userID, _ := cmd.Flags().GetString("user-id")

			if label == "" || userID == "" {
				return fmt.Errorf("--label and --user-id are required")
			}

			repo, cleanup, err := openDB()
			if err != nil {
				return err
			}
			defer cleanup()

			key, err := auth.GenerateAPIKey()
			if err != nil {
				return fmt.Errorf("generating API key: %w", err)
			}

			apiKey, err := repo.CreateAPIKey(key, label, userID)
			if err != nil {
				return fmt.Errorf("creating API key: %w", err)
			}

			fmt.Printf("API key created:\n")
			fmt.Printf("  ID:    %s\n", apiKey.ID)
			fmt.Printf("  Label: %s\n", apiKey.Label)
			fmt.Printf("  User:  %s\n", apiKey.UserID)
			fmt.Printf("  Key:   %s\n", apiKey.Key)
			fmt.Println("\nSave this key now - it will not be shown again.")
			return nil
		},
	}
	createCmd.Flags().String("label", "", "Label for the API key")
	createCmd.Flags().String("user-id", "", "User ID to associate with the key")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all API keys",
		RunE: func(_ *cobra.Command, _ []string) error {
			repo, cleanup, err := openDB()
			if err != nil {
				return err
			}
			defer cleanup()

			keys, err := repo.ListAPIKeys()
			if err != nil {
				return fmt.Errorf("listing API keys: %w", err)
			}

			if len(keys) == 0 {
				fmt.Println("No API keys found.")
				return nil
			}

			fmt.Printf("%-36s  %-20s  %-36s  %-8s  %s\n", "ID", "LABEL", "USER_ID", "ACTIVE", "CREATED")
			for _, k := range keys {
				fmt.Printf("%-36s  %-20s  %-36s  %-8t  %s\n",
					k.ID, k.Label, k.UserID, k.IsActive, k.CreatedAt.Format(time.RFC3339))
			}
			return nil
		},
	}

	revokeCmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke (deactivate) an API key",
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, _ := cmd.Flags().GetString("id")
			if id == "" {
				return fmt.Errorf("--id is required")
			}

			repo, cleanup, err := openDB()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := repo.DeactivateAPIKey(id); err != nil {
				return fmt.Errorf("revoking API key: %w", err)
			}

			fmt.Printf("API key %s has been revoked.\n", id)
			return nil
		},
	}
	revokeCmd.Flags().String("id", "", "ID of the API key to revoke")

	cmd.AddCommand(createCmd, listCmd, revokeCmd)
	return cmd
}

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			db, err := database.New(cfg.DBDriver, cfg.DBDSN)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			return database.RunMigrations(db, cfg.DBDriver, smsgateway.MigrationsFS)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Roll back the last migration",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			db, err := database.New(cfg.DBDriver, cfg.DBDSN)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			return database.MigrateDown(db, cfg.DBDriver, smsgateway.MigrationsFS)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			db, err := database.New(cfg.DBDriver, cfg.DBDSN)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			return database.MigrateStatus(db, cfg.DBDriver, smsgateway.MigrationsFS)
		},
	})

	return cmd
}
