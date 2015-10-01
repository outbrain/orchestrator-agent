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
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/outbrain/orchestrator-agent/agent"
	"github.com/outbrain/orchestrator-agent/config"
	"github.com/outbrain/orchestrator-agent/osagent"
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
	case ERROR:
		return "ERROR"
	case OK:
		return "OK"
	}
	return "unknown"
}

// APIResponse is a response returned as JSON to various requests.
type APIResponse struct {
	Code    APIResponseCode
	Message string
	Details interface{}
}

func validateToken(token string) error {
	if token == agent.ProcessToken.Hash {
		return nil
	} else {
		return errors.New("Invalid token")
	}
}

// Hostname provides information on this process
func (this *HttpAPI) Hostname(params martini.Params, r render.Render) {
	hostname, err := os.Hostname()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, hostname)
}

// ListLogicalVolumes lists logical volumes by pattern
func (this *HttpAPI) ListLogicalVolumes(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.LogicalVolumes("", params["pattern"])
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// ListSnapshotsLogicalVolumes lists logical volumes by pattern
func (this *HttpAPI) ListSnapshotsLogicalVolumes(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.LogicalVolumes("", config.Config.SnapshotVolumesFilter)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// LogicalVolume lists a logical volume by name/path/mount point
func (this *HttpAPI) LogicalVolume(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	lv := params["lv"]
	if lv == "" {
		lv = req.URL.Query().Get("lv")
	}
	output, err := osagent.LogicalVolumes(lv, "")
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// GetMount shows the configured mount point's status
func (this *HttpAPI) GetMount(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.GetMount(config.Config.SnapshotMountPoint)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// MountLV mounts a logical volume on config mount point
func (this *HttpAPI) MountLV(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	lv := params["lv"]
	if lv == "" {
		lv = req.URL.Query().Get("lv")
	}
	output, err := osagent.MountLV(config.Config.SnapshotMountPoint, lv)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// RemoveLV removes a logical volume
func (this *HttpAPI) RemoveLV(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	lv := params["lv"]
	if lv == "" {
		lv = req.URL.Query().Get("lv")
	}
	err := osagent.RemoveLV(lv)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, err == nil)
}

// Unmount umounts the config mount point
func (this *HttpAPI) Unmount(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.Unmount(config.Config.SnapshotMountPoint)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// DiskUsage returns the number of bytes of a give ndirectory (recursive)
func (this *HttpAPI) DiskUsage(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	path := req.URL.Query().Get("path")

	output, err := osagent.DiskUsage(path)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// MySQLDiskUsage returns the number of bytes on the MySQL datadir
func (this *HttpAPI) MySQLDiskUsage(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	datadir, err := osagent.GetMySQLDataDir()

	output, err := osagent.DiskUsage(datadir)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// CreateSnapshot lists dc-local available snapshots for this host
func (this *HttpAPI) CreateSnapshot(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	err := osagent.CreateSnapshot()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, err == nil)
}

// LocalSnapshots lists dc-local available snapshots for this host
func (this *HttpAPI) AvailableLocalSnapshots(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.AvailableSnapshots(true)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// Snapshots lists available snapshots for this host
func (this *HttpAPI) AvailableSnapshots(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.AvailableSnapshots(false)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// returns rows in tail of mysql error log
func (this *HttpAPI) MySQLErrorLogTail(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.MySQLErrorLogTail()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// MySQLPort returns the (heuristic) port on which MySQL executes
func (this *HttpAPI) MySQLPort(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.GetMySQLPort()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// MySQLRunning checks whether the MySQL service is up
func (this *HttpAPI) MySQLRunning(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.MySQLRunning()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// MySQLStop shuts down the MySQL service
func (this *HttpAPI) MySQLStop(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	err := osagent.MySQLStop()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, err == nil)
}

// MySQLStop starts the MySQL service
func (this *HttpAPI) MySQLStart(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	err := osagent.MySQLStart()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, err == nil)
}

// DeleteMySQLDataDir compeltely erases MySQL data directory. Use with care!
func (this *HttpAPI) DeleteMySQLDataDir(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	err := osagent.DeleteMySQLDataDir()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, err == nil)
}

// GetMySQLDataDirAvailableDiskSpace returns the number of bytes free within the MySQL datadir mount
func (this *HttpAPI) GetMySQLDataDirAvailableDiskSpace(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output, err := osagent.GetMySQLDataDirAvailableDiskSpace()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, output)
}

// PostCopy
func (this *HttpAPI) PostCopy(params martini.Params, r render.Render, req *http.Request) {
	if err := validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	err := osagent.PostCopy()
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	r.JSON(200, err == nil)
}

// ReceiveMySQLSeedData
func (this *HttpAPI) ReceiveMySQLSeedData(params martini.Params, r render.Render, req *http.Request) {
	var err error
	if err = validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	go osagent.ReceiveMySQLSeedData(params["seedId"])
	r.JSON(200, err == nil)
}

// SendMySQLSeedData
func (this *HttpAPI) SendMySQLSeedData(params martini.Params, r render.Render, req *http.Request) {
	var err error
	if err = validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	mount, err := osagent.GetMount(config.Config.SnapshotMountPoint)
	if err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	go osagent.SendMySQLSeedData(params["targetHost"], mount.MySQLDataPath, params["seedId"])
	r.JSON(200, err == nil)
}

// AbortSeed
func (this *HttpAPI) AbortSeed(params martini.Params, r render.Render, req *http.Request) {
	var err error
	if err = validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	osagent.AbortSeed(params["seedId"])
	r.JSON(200, err == nil)
}

// SeedCommandCompleted
func (this *HttpAPI) SeedCommandCompleted(params martini.Params, r render.Render, req *http.Request) {
	var err error
	if err = validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output := osagent.SeedCommandCompleted(params["seedId"])
	r.JSON(200, output)
}

// SeedCommandCompleted
func (this *HttpAPI) SeedCommandSucceeded(params martini.Params, r render.Render, req *http.Request) {
	var err error
	if err = validateToken(req.URL.Query().Get("token")); err != nil {
		r.JSON(500, &APIResponse{Code: ERROR, Message: err.Error()})
		return
	}
	output := osagent.SeedCommandSucceeded(params["seedId"])
	r.JSON(200, output)
}

// A simple status endpoint to ping to see if the agent is up and responding.  There's not much
// to do here except respond with 200 and OK
// This is pointed to by a configurable endpoint and has a configurable status message
func (this *HttpAPI) Status(params martini.Params, r render.Render, req *http.Request) {
	if uint(time.Since(agent.LastTalkback).Seconds()) > config.Config.StatusBadSeconds {
		r.JSON(500, "BAD")
	} else {
		r.JSON(200, "OK")
	}
}

// RegisterRequests makes for the de-facto list of known API calls
func (this *HttpAPI) RegisterRequests(m *martini.ClassicMartini) {
	m.Get("/api/hostname", this.Hostname)
	m.Get("/api/lvs", this.ListLogicalVolumes)
	m.Get("/api/lvs/:pattern", this.ListLogicalVolumes)
	m.Get("/api/lvs-snapshots", this.ListSnapshotsLogicalVolumes)
	m.Get("/api/lv", this.LogicalVolume)
	m.Get("/api/lv/:lv", this.LogicalVolume)
	m.Get("/api/mount", this.GetMount)
	m.Get("/api/mountlv", this.MountLV)
	m.Get("/api/removelv", this.RemoveLV)
	m.Get("/api/umount", this.Unmount)
	m.Get("/api/du", this.DiskUsage)
	m.Get("/api/mysql-du", this.MySQLDiskUsage)
	m.Get("/api/create-snapshot", this.CreateSnapshot)
	m.Get("/api/available-snapshots-local", this.AvailableLocalSnapshots)
	m.Get("/api/available-snapshots", this.AvailableSnapshots)
	m.Get("/api/mysql-error-log-tail", this.MySQLErrorLogTail)
	m.Get("/api/mysql-port", this.MySQLPort)
	m.Get("/api/mysql-status", this.MySQLRunning)
	m.Get("/api/mysql-stop", this.MySQLStop)
	m.Get("/api/mysql-start", this.MySQLStart)
	m.Get("/api/delete-mysql-datadir", this.DeleteMySQLDataDir)
	m.Get("/api/mysql-datadir-available-space", this.GetMySQLDataDirAvailableDiskSpace)
	m.Get("/api/post-copy", this.PostCopy)
	m.Get("/api/receive-mysql-seed-data/:seedId", this.ReceiveMySQLSeedData)
	m.Get("/api/send-mysql-seed-data/:targetHost/:seedId", this.SendMySQLSeedData)
	m.Get("/api/abort-seed/:seedId", this.AbortSeed)
	m.Get("/api/seed-command-completed/:seedId", this.SeedCommandCompleted)
	m.Get("/api/seed-command-succeeded/:seedId", this.SeedCommandSucceeded)
	m.Get(config.Config.StatusEndpoint, this.Status)
}
