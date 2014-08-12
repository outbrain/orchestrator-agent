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

package http

import (
	"os"
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

type HttpAPI struct{}

var API HttpAPI = HttpAPI{}


// APIResponseCode is an OK/ERROR response code
type APIResponseCode int

const (
	ERROR APIResponseCode = iota
	OK
)

func (this *APIResponseCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.String())
}

func (this *APIResponseCode) String() string {
	switch *this {
		case ERROR: return "ERROR"
		case OK: return "OK"
	}
	return "unknown"
}


// APIResponse is a response returned as JSON to various requests.
type APIResponse struct {
	Code	APIResponseCode
	Message	string
	Details	interface{}
}

// Info provides information on this process
func (this *HttpAPI) Info(params martini.Params, r render.Render) {

	hostname, err := os.Hostname()

	if err != nil {
		r.JSON(200, &APIResponse{Code:ERROR, Message: err.Error(),})
		return
	}

	r.JSON(200, hostname)
}

// RegisterRequests makes for the de-facto list of known API calls
func (this *HttpAPI) RegisterRequests(m *martini.ClassicMartini) {
	m.Get("/api/info", this.Info) 
}
