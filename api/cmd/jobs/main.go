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
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.com/apf/api/core"
	"google.com/apf/api/jobs"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, jobs.DefaultRegistry, core.NewConfig)
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, reg *jobs.Registry, configLoader func() *core.Configuration) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if reg == nil {
		reg = jobs.DefaultRegistry
	}
	if configLoader == nil {
		configLoader = core.NewConfig
	}

	fs := flag.NewFlagSet("apf-jobs", flag.ContinueOnError)
	fs.SetOutput(stderr)

	listFlag := fs.Bool("list", false, "List all registered housekeeping jobs")
	nameFlag := fs.String("name", "", "Name of the housekeeping job to run")
	dryRunFlag := fs.Bool("dry-run", false, "Execute the housekeeping job in dry-run mode without modifying disk or data")

	var positional []string
	remainingArgs := args
	for len(remainingArgs) > 0 {
		if err := fs.Parse(remainingArgs); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0
			}
			fmt.Fprintf(stderr, "Error parsing arguments: %v\n", err)
			return 1
		}
		extra := fs.Args()
		if len(extra) == 0 {
			break
		}
		positional = append(positional, extra[0])
		remainingArgs = extra[1:]
	}

	if *listFlag {
		printJobsList(stdout, reg)
		return 0
	}

	jobName := strings.TrimSpace(*nameFlag)
	if jobName == "" && len(positional) > 0 {
		jobName = strings.TrimSpace(positional[0])
		if len(positional) > 1 {
			fmt.Fprintf(stderr, "Error: unexpected extra arguments: %v\n", positional[1:])
			return 1
		}
	} else if jobName != "" && len(positional) > 0 {
		fmt.Fprintf(stderr, "Error: unexpected extra arguments: %v\n", positional)
		return 1
	}

	if jobName == "" {
		slog.Error("Missing required job name")
		fmt.Fprintln(stderr, "Error: missing required job name.")
		fmt.Fprintln(stderr, "\nUsage:")
		fmt.Fprintln(stderr, "  apf-jobs --list")
		fmt.Fprintln(stderr, "  apf-jobs <job-name>")
		fmt.Fprintln(stderr, "  apf-jobs --name <job-name>")
		fmt.Fprintln(stderr, "  apf-jobs <job-name> --dry-run")
		fmt.Fprintln(stderr, "")
		printJobsList(stderr, reg)
		return 1
	}

	if _, ok := reg.Get(jobName); !ok {
		slog.Error("Unknown job name", "job", jobName)
		fmt.Fprintf(stderr, "Error: unknown job %q\n\n", jobName)
		printJobsList(stderr, reg)
		return 1
	}

	if *dryRunFlag {
		ctx = jobs.ContextWithDryRun(ctx, true)
	}

	cfg := configLoader()

	slog.Info("Starting housekeeping job", "job", jobName, "dryRun", *dryRunFlag)
	start := time.Now()
	err := reg.Execute(ctx, jobName, cfg)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Housekeeping job failed", "job", jobName, "duration", duration, "error", err)
		fmt.Fprintf(stderr, "Error: job %q failed: %v\n", jobName, err)
		return 1
	}

	slog.Info("Housekeeping job completed successfully", "job", jobName, "duration", duration)
	fmt.Fprintf(stdout, "Job %q completed successfully in %v\n", jobName, duration)
	return 0
}

func printJobsList(w io.Writer, reg *jobs.Registry) {
	jobList := reg.List()
	fmt.Fprintln(w, "Available Housekeeping Jobs:")
	if len(jobList) == 0 {
		fmt.Fprintln(w, "  (no jobs registered)")
		return
	}

	maxLen := 0
	for _, j := range jobList {
		if l := len(j.Name()); l > maxLen {
			maxLen = l
		}
	}
	if maxLen < 20 {
		maxLen = 20
	}

	format := fmt.Sprintf("  %%-%ds %%s\n", maxLen)
	for _, j := range jobList {
		fmt.Fprintf(w, format, j.Name(), j.Description())
	}
}
