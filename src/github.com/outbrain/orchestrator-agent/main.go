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

package main 

import (
	"flag"
	"github.com/outbrain/orchestrator-agent/process"
	"github.com/outbrain/orchestrator-agent/app"
	"github.com/outbrain/orchestrator-agent/config"
	"github.com/outbrain/log"
)

// main is the application's entry point. It will either spawn a CLI or HTTP itnerfaces.
func main() {
	configFile := flag.String("config", "", "config file name")
	verbose := flag.Bool("verbose", false, "verbose")
	debug := flag.Bool("debug", false, "debug mode (very verbose)")
	stack := flag.Bool("stack", false, "add stack trace upon error")
	flag.Parse();
	
	log.SetLevel(log.ERROR)
	if *verbose {
		log.SetLevel(log.INFO)
	}
	if *debug {
		log.SetLevel(log.DEBUG)
	}
	if *stack {
		log.SetPrintStackTrace(*stack)
	}

	log.Info("starting")

	if len(*configFile) > 0 {
		config.ForceRead(*configFile)
	} else {
		config.Read("/etc/orchestrator-agent.conf.json", "conf/orchestrator-agent.conf.json", "orchestrator-agent.conf.json")
	}
	
	log.Debugf("Process token: %s", token.ProcessToken.Hash)
	
	app.Http()
}
