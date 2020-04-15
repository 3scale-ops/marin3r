// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package controller

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// NewLogger returns a logger object from the given params
func NewLogger(logLevel string) *zap.SugaredLogger {

	rawJSON := []byte(`{
		"level": "info",
		"outputPaths": ["stdout"],
		"errorOutputPaths": ["stderr"],
		"encoding": "json",
		"encoderConfig": {
			"messageKey": "message",
			"levelKey": "level",
			"levelEncoder": "lowercase"
		}
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	if logLevel != "info" {
		switch logLevel {
		case "debug":
			cfg.Level.SetLevel(zap.DebugLevel)
		case "warn":
			cfg.Level.SetLevel(zap.WarnLevel)
		case "error":
			cfg.Level.SetLevel(zap.ErrorLevel)
		}
	}

	defer logger.Sync() // flushes buffer, if any
	return logger.Sugar()
}

// RunSignalWatcher listens for system calls to gracefully shutdown all
// components when the SIGHUP, SIGINT, SIGTERM ot SIGQUIT is received
func RunSignalWatcher(logger *zap.SugaredLogger) context.Context {
	// Create a context and cancel it when proper
	// signals are received
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		oscall := <-sigc
		logger.Infof("Received system call: %+v", oscall)
		cancel()
	}()

	return ctx
}
