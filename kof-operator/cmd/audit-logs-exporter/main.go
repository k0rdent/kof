// Copyright 2025
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
	"os"
	"os/signal"
	"syscall"

	"github.com/k0rdent/kof/kof-operator/internal/audit"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	opts := zap.Options{Development: false}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	log := ctrl.Log.WithName("audit-logs-exporter")

	cfg, err := audit.LoadConfig()
	if err != nil {
		log.Error(err, "invalid configuration")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	exporter, err := audit.NewExporter(cfg, log)
	if err != nil {
		log.Error(err, "failed to initialise exporter")
		os.Exit(1)
	}

	if err := exporter.Run(ctx); err != nil {
		log.Error(err, "export run failed")
		os.Exit(1)
	}
}
