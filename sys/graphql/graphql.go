package graphql

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"saas-starter-api/res/auth"
	"saas-starter-api/res/mail"
	"saas-starter-api/res/notification"
	"saas-starter-api/res/store"
	"saas-starter-api/sys/graphql/directive"
	"saas-starter-api/sys/graphql/gen"
	"saas-starter-api/sys/http/middleware"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
)

//go:generate go run github.com/99designs/gqlgen

type Config struct {
	Logger              *log.Logger
	Store               store.Store
	MailService         mail.MailService
	NotificationService notification.NotificationService
	Auth                auth.Auth
}

type Resolver struct {
	*Config
}

type queryResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }

func (r *Resolver) Query() gen.QueryResolver       { return &queryResolver{r} }
func (r *Resolver) Mutation() gen.MutationResolver { return &mutationResolver{r} }

func New(cfg *Config) http.Handler {
	schemaCfg := gen.Config{Resolvers: &Resolver{Config: cfg}}

	schemaCfg.Directives.AuthRequired = directive.AuthRequired

	// Create server without default transports to have full control
	gqlServerHandler := handler.New(gen.NewExecutableSchema(schemaCfg))

	// Add default transports
	gqlServerHandler.AddTransport(transport.POST{})
	gqlServerHandler.AddTransport(transport.GET{})
	gqlServerHandler.AddTransport(transport.Options{})
	gqlServerHandler.AddTransport(transport.MultipartForm{})

	// Configure WebSocket transport with proper origin checking and authentication
	gqlServerHandler.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				environment := os.Getenv("ENVIRONMENT")

				// In production, only allow connections from trusted frontend domain
				if environment == "production" {
					origin := r.Header.Get("Origin")
					allowedOrigin := os.Getenv("FRONTEND_URL")
					if allowedOrigin == "" {
						// Fallback to default production domain if not set
						allowedOrigin = "https://app.example.com"
					}
					return origin == allowedOrigin
				}

				// In development, allow all origins for easier testing
				return true
			},
		},
		InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
			// Extract authorization from connection init payload
			if authHeader, ok := initPayload["authorization"].(string); ok && strings.TrimSpace(authHeader) != "" {
				// Parse the Bearer token
				parts := strings.Split(strings.TrimSpace(authHeader), " ")
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(parts[1]) != "" {
					// Validate the token
					var accessTokenClaims auth.AccessTokenClaims
					err := cfg.Auth.ValidateToken(parts[1], &accessTokenClaims)
					if err == nil {
						// Get the user from the store
						currentUser, err := cfg.Store.Users().Get(ctx, accessTokenClaims.UserID)
						if err == nil && currentUser != nil {
							// Add user to context using the same key as the middleware
							ctx = context.WithValue(ctx, middleware.GetCurrentUserKey(), currentUser)
							cfg.Logger.Printf("WebSocket authenticated user: %s (ID: %s)", currentUser.DisplayName, currentUser.ID)
							return ctx, &initPayload, nil
						} else {
							cfg.Logger.Printf("WebSocket authentication failed: user not found")
							return ctx, nil, errors.New("AUTHENTICATION_FAILED")
						}
					} else {
						cfg.Logger.Printf("WebSocket authentication failed: invalid token")
						return ctx, nil, errors.New("INVALID_TOKEN")
					}
				} else {
					// Don't log for empty or malformed headers to reduce noise
					return ctx, nil, errors.New("MALFORMED_AUTH_HEADER")
				}
			} else {
				// Allow connections without authorization - they will be handled by subscription resolvers
				// This prevents constant error logs for connections that don't need authentication
				return ctx, &initPayload, nil
			}
		},
	})

	// Introspection allows clients to discover the schema
	environment := os.Getenv("ENVIRONMENT")
	gqlServerHandler.Use(extension.Introspection{})
	cfg.Logger.Printf("GraphQL introspection enabled (environment: %s)", environment)

	gqlServerHandler.Use(extension.FixedComplexityLimit(90))

	return gqlServerHandler
}

func NewPlayground(introspectionPath string) http.HandlerFunc {
	return playground.Handler("GraphQL playground", introspectionPath)
}

// UTILITIES

const (
	paginationLimitDefault = 50
	paginationLimitMax     = 200
)

func parseForwardPaginationInput(in *gen.ForwardPaginationInput) (first int, afterID *string) {
	if in == nil {
		return paginationLimitDefault, nil
	} else if in.First == nil {
		return paginationLimitDefault, in.After
	} else if *in.First > paginationLimitMax {
		return paginationLimitMax, in.After
	}
	return *in.First, in.After
}

func readRequiredEnvVar(name string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		log.Fatalf("Env variable not set: %s", name)
	}
	return val
}
