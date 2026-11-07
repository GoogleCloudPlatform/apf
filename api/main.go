// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	firestoregorilla "github.com/GoogleCloudPlatform/firestore-gorilla-sessions"
	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.com/apf/api/agent"
	core "google.com/apf/api/core"
	"google.com/apf/api/db"
	"google.com/apf/api/integrations"
	"google.com/apf/api/workflow"
	"google.com/apf/api/workspace"
)

var appConfig *core.Configuration
var sessionCookieStore *firestoregorilla.Store
var sessionStore db.SessionStore



type Response struct {
	Title       string `json:"title,omitempty"`
	Message     string `json:"message"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

func main() {
	gob.Register(&core.UserClaims{})
	cfg := core.NewConfig()
	appConfig = cfg
	setup_telemetry(cfg)
	flag.Parse()

	ctx := context.Background()

	// Check if running in Firebase Emulator environment
	// The emulator sets FIRESTORE_EMULATOR_HOST environment variable
	// The project ID is required for the emulator, but it doesn't use the real project

	var client *firestore.Client
	var err error

	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		// Running in emulator environment
		fmt.Println("Detected Firebase Emulator environment")
		fmt.Printf("Emulator host: %s\n", os.Getenv("FIRESTORE_EMULATOR_HOST"))

		// For emulator, project ID doesn't matter for connection, but Firestore API requires one
		// Use the same project ID as production for consistency
		client, err = firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
		if err != nil {
			log.Fatalf("Failed to create Firestore client for emulator: %v", err)
		}
	} else {
		// Production environment
		fmt.Println("Using production Firestore database")
		client, err = firestore.NewClientWithDatabase(ctx, cfg.GoogleCloudProject, cfg.FirestoreDbName)
		if err != nil {
			log.Fatalf("Failed to create Firestore client: %v", err)
		}
	}

	FirestoreClient = client
	sessionCookieStore, err = firestoregorilla.New(ctx, client)
	if err != nil {
		log.Fatalf("Failed to create firestore gorilla store: %v", err)
	}
	defer client.Close()

	// Initialize Firebase Authentication with fallback to Mock Authenticator
	var authClient *auth.Client
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.GoogleCloudProject})
	if err != nil {
		log.Printf("Warning: Failed to initialize Firebase App: %v. Falling back to Mock Authenticator.", err)
	} else {
		var authErr error
		authClient, authErr = app.Auth(ctx)
		if authErr != nil {
			log.Printf("Warning: Failed to create Firebase Auth client: %v. Falling back to Mock Authenticator.", authErr)
		} else {
			globalAuthenticator = &FirebaseAuthAuthenticator{Client: authClient, FirestoreClient: client}
			log.Println("Firebase Authentication initialized successfully")
		}
	}

	// Initialize organization provisioner only if Firebase Auth is successfully initialized AND (not in local simulation OR the local emulator host is configured)
	useRealProvisioner := !cfg.BypassIAPValidation || os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != ""
	if authClient != nil && useRealProvisioner {
		globalOrganizationProvisioner = &FirebaseOrganizationProvisioner{TenantManager: authClient.TenantManager}
	} else {
		log.Println("Using Mock Organization Provisioner (Firebase Auth not initialized and emulator not configured)")
	}

	// TODO find out what is store and what is the rleationship to agent.Store and what backward compatibility issue is. Rename to scope better to avoid multithreading issues
	store = db.NewFirestoreProjectStore(client)
	db.ProjectStoreInstance = store
	requestStore = db.NewFirestoreAccessRequestStore(client)
	workflow.Store = db.NewFirestoreWorkflowStore(client)
	workflow.RunStore = db.NewFirestoreWorkflowRunStore(client)
	db.WorkflowRunStoreInstance = workflow.RunStore
	workspace.ArtifactStore = db.NewFirestoreArtifactStore(client)
	workspace.SessionStore = db.NewFirestoreSessionStore(client)
	workspace.AppConfig = cfg
	configStore = db.NewFirestoreConfigStore(client)
	workspace.ConfigStore = configStore
	sessionStore = workspace.SessionStore
	autojoinStore = db.NewFirestoreAutojoinStore(client)
	fUserStore := db.NewFirestoreUserStore(client, authClient).WithAutojoinStore(autojoinStore)
	userStore = fUserStore
	workspace.UserStore = fUserStore
	organizationStore = db.NewFirestoreOrganizationStore(client)
	db.OrganizationStoreInstance = organizationStore

	appCredentialStore = db.NewFirestoreAppCredentialStore(client)
	db.AppCredentialStoreInstance = appCredentialStore
	userProjectStore = db.NewFirestoreUserProjectStore(client)
	db.UserProjectStoreInstance = userProjectStore
	workspace.UserProjectStore = userProjectStore
	agent.Store = db.NewFirestoreMessageStore(client, sessionStore)
	Store = agent.Store // Keep for backwards compatibility
	agent.SetArtifactStore(workspace.ArtifactStore)
	agent.SetConfig(cfg)
	integrations.Store = db.NewFirestoreIntegrationStore(client)

	// Initialize Email Service & Notification Providers
	core.InitializeEmailService(cfg)





	nfsStore := db.NewFirestoreNFSStore(client)
	workspace.NFSStore = nfsStore

	var jobRunner CloudRunJobRunner
	gcpRunner, err := NewGCPCloudRunJobRunner(ctx, cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize GCP Cloud Run Job Runner: %v. Using simulated runner.", err)
		jobRunner = NewSimulatedCloudRunJobRunner(cfg)
	} else {
		jobRunner = gcpRunner
	}

	mux := http.NewServeMux()
	chatHandler := NewChatHandler(agent.Store).
		WithArtifactStore(workspace.ArtifactStore).
		WithAppConfig(cfg)
	sessionHandler := NewSessionHandler(sessionStore, agent.Store, sessionCookieStore).
		WithArtifactStore(workspace.ArtifactStore).
		WithAppConfig(cfg)
	adminHandler := NewAdminHandler(userStore, organizationStore, cfg).
		WithAutojoinStore(autojoinStore).
		WithNFSStore(nfsStore).
		WithJobRunner(jobRunner)
	authHandler := NewAuthHandler(
		globalOrganizationProvisioner,
		globalAuthenticator,
		globalTokenValidator,
		userStore,
		organizationStore,
		cfg,
	)
	featureStore := db.NewFirestoreFeatureStore(client)
	featureHandler := NewFeatureHandler(featureStore)
	feedbackStore := db.NewFirestoreFeedbackStore(client)
	feedbackHandler := NewFeedbackHandler(feedbackStore)
	sessionDB, err := db.GetSessionDBSingleton(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize Session Database: %v", err)
	}
	telemetryHandler := NewTelemetryHandler(sessionDB, client, organizationStore)
	registerRoutes(mux, chatHandler, sessionHandler, adminHandler, authHandler, featureHandler, feedbackHandler, telemetryHandler)

	address := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	fmt.Printf("Starting server on %s\n", address)

	otelHandler := otelhttp.NewHandler(mux, "api-server")

	handler := corsMiddleware(loggingMiddleware(otelHandler))
	// handler := corsMiddleware(otelHandler)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	} else {
		fmt.Printf("Server listening on %s\n", address)
	}
}

type ServiceStatus string

const (
	StatusUp   ServiceStatus = "UP"
	StatusDown ServiceStatus = "DOWN"
)

type HealthResponse struct {
	Status   string                   `json:"status"`
	Version  string                   `json:"version,omitempty"`
	Services map[string]ServiceStatus `json:"services"`
}

type VersionResponse struct {
	Version string `json:"version"`
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, VersionResponse{
		Version: getPlatformVersion(),
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	aiStatus := StatusUp

	agentURL := "http://localhost:8080"
	if appConfig != nil && appConfig.AgentBaseURL != "" {
		agentURL = appConfig.AgentBaseURL
	}

	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	probeReq, err := http.NewRequestWithContext(probeCtx, "GET", agentURL, nil)
	if err != nil {
		aiStatus = StatusDown
	} else {
		probeResp, err := http.DefaultClient.Do(probeReq)
		if err != nil {
			aiStatus = StatusDown
		} else {
			probeResp.Body.Close()
		}
	}

	status := "OK"
	statusCode := http.StatusOK
	if aiStatus == StatusDown {
		status = "ERROR"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:  status,
		Version: getPlatformVersion(),
		Services: map[string]ServiceStatus{
			"api": StatusUp,
			"ai":  aiStatus,
		},
	}

	sendJSON(w, statusCode, response)
}

func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	// TODO(enterprise): Restrict Access-Control-Allow-Origin to trusted enterprise domains before production deployment.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-goog-authenticated-user-email, x-goog-authenticated-user-id, X-Organization-Id")

	w.WriteHeader(statusCode)
	if statusCode >= 500 {
		log.Printf("Server Error %d: %v", statusCode, data)
	}
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}

}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, nil)
}

func setup_telemetry(cfg *core.Configuration) *resource.Resource {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Printf("failed to create resource: %v", err)
	}

	exporter, err := trace.New()
	if err != nil {
		log.Printf("Warning: Cloud Trace disabled, could not create exporter: %v", err)
		return res
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Telemetry: Google Cloud Trace enabled.")
	return res
}
