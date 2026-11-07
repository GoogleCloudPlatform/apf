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
	"log"
	"os"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.com/apf/ai/core"
	"google.com/apf/ai/plugins/bmad"
	"google.com/apf/ai/registry"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/plugin"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/telemetry"
)

func main() {
	ctx := context.Background()
	cfg := core.NewConfig()

	registry.RegisterAllAgents()

	var err error
	var agentLoader agent.Loader
	if cfg.AgentLoaderName == "" || cfg.AgentLoaderName == "super" {
		agentLoader, err = registry.GetSuperLoader(ctx, cfg)
	} else {
		agentLoaderFunc := registry.GetAgentLoader(cfg.AgentLoaderName)
		agentLoader, err = agentLoaderFunc(ctx, cfg)
	}

	if err != nil {
		log.Fatalf("Failed to initialize agent loader: %v", err)
	}

	sessionService, err := core.NewSessionService(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create session service: %v", err)
	}

	artifactService, err := core.NewArtifactService(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create artifact service: %v", err)
	}

	bmadInterceptorPlugin, err := bmad.NewBmadInterceptorPlugin(cfg)
	if err != nil {
		log.Fatalf("Failed to create BmadInterceptorPlugin: %v", err)
	}

	r := setup_telemetry(cfg)

	config := &launcher.Config{
		ArtifactService: artifactService,
		SessionService:  sessionService,
		AgentLoader:     agentLoader,
		PluginConfig: runner.PluginConfig{
			Plugins: []*plugin.Plugin{bmadInterceptorPlugin},
		},
		TelemetryOptions: []telemetry.Option{
			telemetry.WithResource(r),
		},
	}

	args := process_args(os.Args[1:], cfg)

	l := full.NewLauncher()
	if err = l.Execute(ctx, config, args); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
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

	// Set up Cloud Trace exporter.
	// It will use Application Default Credentials (ADC) to find the project ID.
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

	// Set up W3C trace context propagation to link spans correctly across services.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Telemetry: Google Cloud Trace enabled.")
	return res
}

func process_args(args []string, cfg *core.Configuration) []string {

	// Inject -a2a_agent_url flag after "a2a" sublauncher for correct agent card URL
	appURL := cfg.AppUrl

	var newArgs []string
	for _, arg := range args {
		newArgs = append(newArgs, arg)
		if arg == "a2a" {
			newArgs = append(newArgs, "-a2a_agent_url", appURL)
		}
		if arg == "webui" {
			newArgs = append(newArgs, "-api_server_address", appURL+"/api")
		}
	}
	return newArgs

}
