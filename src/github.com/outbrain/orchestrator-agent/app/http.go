/*
   Copyright 2014 Outbrain Inc.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

//
package app

import (
	"fmt"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/auth"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/render"

	nethttp "net/http"

	"github.com/outbrain/log"
	"github.com/outbrain/orchestrator-agent/agent"
	"github.com/outbrain/orchestrator-agent/config"
	"github.com/outbrain/orchestrator-agent/http"
)

// Http starts serving HTTP (api/web) requests
func Http() {
	m := martini.Classic()
	if config.Config.HTTPAuthUser != "" {
		m.Use(auth.Basic(config.Config.HTTPAuthUser, config.Config.HTTPAuthPassword))
	}

	m.Use(gzip.All())
	// Render html templates from templates directory
	m.Use(render.Renderer(render.Options{
		Directory:       "resources",
		Layout:          "templates/layout",
		HTMLContentType: "text/html",
	}))
	m.Use(martini.Static("resources/public"))

	go agent.ContinuousOperation()

	log.Infof("Starting HTTP on port %d", config.Config.HTTPPort)

	http.API.RegisterRequests(m)

	// Serve
	if config.Config.UseSSL {
		log.Info("Serving via SSL")
		err := nethttp.ListenAndServeTLS(fmt.Sprintf(":%d", config.Config.HTTPPort), config.Config.SSLCertFile, config.Config.SSLPrivateKeyFile, m)
		if err != nil {
			log.Fatale(err)
		}
	} else {
		nethttp.ListenAndServe(fmt.Sprintf(":%d", config.Config.HTTPPort), m)
	}
}
