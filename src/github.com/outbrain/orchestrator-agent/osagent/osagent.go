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

package osagent

import (
	"fmt"
	"errors"
	"os"
	"os/exec"
	"io/ioutil"
	"path"
	"strings"
	"strconv"
	"regexp"
	"github.com/outbrain/log"

	"github.com/outbrain/orchestrator-agent/config"
)

const (
	SeedTransferPort	= 21234
)

// LogicalVolume describes an LVM volume
type LogicalVolume struct {
	Name			string
	GroupName		string
	Path			string
	IsSnapshot		bool
	SnapshotPercent	float64
}


// Equals tests equality of this corrdinate and another one.
func (this *LogicalVolume) IsSnapshotValid() bool {
	if !this.IsSnapshot { return false }
	if this.SnapshotPercent >= 100.0 { return false }
	return true
}


// Mount describes a file system mount point
type Mount struct {
	Path			string
	Device			string
	LVPath			string
	FileSystem		string
	IsMounted		bool
	DiskUsage		int64
	MySQLDataPath	string
	MySQLDiskUsage	int64
}


func commandSplit(commandText string) (string, []string) {
	tokens := regexp.MustCompile(`[ ]+`).Split(strings.TrimSpace(commandText), -1)
	return tokens[0], tokens[1:]
}

func execCmd(commandText string) (*exec.Cmd, string, error) {
	commandBytes := []byte(commandText)
	tmpFile, err := ioutil.TempFile("", "orchestrator-agent-cmd-")
	if err != nil {
		return nil, "", log.Errore(err)
	}
	ioutil.WriteFile(tmpFile.Name(), commandBytes, 0644)
	return exec.Command("bash", tmpFile.Name()), tmpFile.Name(), nil
}

// commandOutput executes a command and return output bytes
func commandOutput(commandText string) ([]byte, error) {
//	commandBytes := []byte(commandText)
//	tmpFile, err := ioutil.TempFile("", "orchestrator-agent-")
//	if err != nil {
//		return nil, log.Errore(err)
//	}
//	ioutil.WriteFile(tmpFile.Name(), commandBytes, 0644)
//	outputBytes, err := exec.Command("bash", tmpFile.Name()).Output()
	
	cmd, tmpFileName, err := execCmd(commandText)
	if err != nil {	return nil, log.Errore(err)}
	
	outputBytes, err := cmd.Output()
	if err != nil {	return nil, log.Errore(err)}
//	commandName, commandArgs := commandSplit(commandText)
//	
//	log.Debugf("commandOutput: %s", commandText)
//	outputBytes, err := exec.Command(commandName, commandArgs...).Output()
//	if err != nil {	return nil, log.Errore(err)}
	os.Remove(tmpFileName)
	
	return outputBytes, nil
}

// commandStart executes a command and does not wait for completion
func commandStart(commandText string) error {
	cmd, _, err := execCmd(commandText)
	if err != nil {	return log.Errore(err)}
	return cmd.Start()
}


func outputLines(commandOutput []byte, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}
	text := strings.Trim(fmt.Sprintf("%s", commandOutput), "\n")
	lines := strings.Split(text, "\n")
	return lines, err
}

func outputTokens(delimiterPattern string, commandOutput []byte, err error) ([][]string, error) {
	lines, err := outputLines(commandOutput, err)
	if err != nil {
		return nil, err
	}
	tokens := make([][]string, len(lines))
	for i := range tokens {
		tokens[i] = regexp.MustCompile(delimiterPattern).Split(lines[i], -1)
	}
	return tokens, err
}


func Hostname() (string, error) {
	return os.Hostname()
}


func LogicalVolumes(volumeName string, filterPattern string) ([]LogicalVolume, error) {
	output, err := commandOutput(fmt.Sprintf("lvs --noheading -o lv_name,vg_name,lv_path,snap_percent %s", volumeName))
	tokens, err := outputTokens(`[ \t]+`, output, err)
	if err != nil { return nil, err }
	
	logicalVolumes := []LogicalVolume{}
	for _, lineTokens := range tokens {
		logicalVolume := LogicalVolume {
			Name:		lineTokens[1],
			GroupName: 	lineTokens[2],
			Path:		lineTokens[3],
		}
		logicalVolume.SnapshotPercent, err = strconv.ParseFloat(lineTokens[4], 32)
		logicalVolume.IsSnapshot = (err == nil)
		if strings.Contains(logicalVolume.Name, filterPattern) {
	    	logicalVolumes = append(logicalVolumes, logicalVolume)
	    }
	}
	return logicalVolumes, nil
}


func GetLogicalVolumePath(volumeName string) (string, error) {
	if logicalVolumes, err := LogicalVolumes(volumeName, ""); err == nil && len(logicalVolumes) > 0 {
		return logicalVolumes[0].Path, err
	}
	return "", errors.New(fmt.Sprintf("logical volume not found: %+v", volumeName))
}



func GetLogicalVolumeFSType(volumeName string) (string, error) {
	command := fmt.Sprintf("blkid %s", volumeName)
	output, err := commandOutput(command)
	lines, err := outputLines(output, err)
	re := regexp.MustCompile(`TYPE="(.*?)"`)
	for _, line := range lines {
		fsType := re.FindStringSubmatch(line)[1]
		return fsType, nil
	}
	return "", errors.New(fmt.Sprintf("Cannot find FS type for logical volume %s", volumeName))
}


func GetMount(mountPoint string) (Mount, error) {
	mount := Mount {
		Path: mountPoint,	
		IsMounted: false,	
	}

	output, err := commandOutput(fmt.Sprintf("grep %s /etc/mtab", mountPoint))
	tokens, err := outputTokens(`[ \t]+`, output, err)
	if err != nil {
		// when grep does not find rows, it returns an error. So this is actually OK 
		return mount, nil
	}
	
	for _, lineTokens := range tokens {
		mount.IsMounted = true
		mount.Device = lineTokens[0]
		mount.Path = lineTokens[1]
		mount.FileSystem = lineTokens[2]
		mount.LVPath, _ = GetLogicalVolumePath(mount.Device)
		mount.DiskUsage, _ = DiskUsage(mountPoint)
		mount.MySQLDataPath, _ = HeuristicMySQLDataPath(mountPoint)
		mount.MySQLDiskUsage, _ = DiskUsage(mount.MySQLDataPath)
	}
	return mount, nil
}

func MountLV(mountPoint string, volumeName string) (Mount, error) {
	mount := Mount {
		Path: mountPoint,	
		IsMounted: false,	
	}
	if volumeName == "" {
		return mount, errors.New("Empty columeName in MountLV")
	}
	fsType, err := GetLogicalVolumeFSType(volumeName)
	if err != nil {	return mount, err}

	mountOptions := ""
	if fsType == "xfs" {
		mountOptions = "-o nouuid"
	}	
	_, err = commandOutput(fmt.Sprintf("mount %s %s %s", mountOptions, volumeName, mountPoint))
	if err != nil {	return mount, err}

	return GetMount(mountPoint)
}

func Unmount(mountPoint string) (Mount, error) {
	mount := Mount {
		Path: mountPoint,	
		IsMounted: false,	
	}
	_, err := commandOutput(fmt.Sprintf("umount %s", mountPoint))
	if err != nil {
		return mount, err
	}
	return GetMount(mountPoint)
}


func DiskUsage(path string) (int64, error) {
	var result int64

	output, err := commandOutput(fmt.Sprintf("du -sb %s", path))
	tokens, err := outputTokens(`[ \t]+`, output, err)
	if err != nil {
		return result, err
	}
	
	for _, lineTokens := range tokens {
		result, err = strconv.ParseInt(lineTokens[0], 10, 0)
		return result, err
	}
	return result, err
}



// DeleteMySQLDataDir self explanatory. Be responsible! This function does not verify the MySQL service is down
func DeleteMySQLDataDir() error {
	
	directory, err := GetMySQLDataDir() 
	if err != nil {return err}

	directory = strings.TrimSpace(directory)
	if directory == "" {
		return errors.New("refusing to delete empty directory")
	}
	if path.Dir(directory) == directory {
		return errors.New(fmt.Sprintf("Directory %s seems to be root; refusing to delete", directory))
	}
	_, err = commandOutput(config.Config.MySQLDeleteDatadirContentCommand)

	return err
}


func HeuristicMySQLDataPath(mountPoint string) (string, error) {
	datadir, err := GetMySQLDataDir()
	if err != nil {return "", err}
	
	heuristicFileName := "ibdata1"
	
	re := regexp.MustCompile(`/[^/]+(.*)`)
	for {
		heuristicFullPath := path.Join(mountPoint, datadir, heuristicFileName)
		log.Debugf("search for %s", heuristicFullPath)
		if _, err := os.Stat(heuristicFullPath); err == nil {
		    return path.Join(mountPoint, datadir), nil
		}
		if datadir == "" {
			return "", errors.New("Cannot detect MySQL datadir")
		}
		datadir = re.FindStringSubmatch(datadir)[1]
	}
}


func GetMySQLDataDir() (string, error) {
	command := config.Config.MySQLDatadirCommand 
	output, err := commandOutput(command)
	return strings.TrimSpace(fmt.Sprintf("%s", output)), err
}


func GetMySQLPort() (int64, error) {
	command := config.Config.MySQLPortCommand 
	output, err := commandOutput(command)
	if err != nil {return 0, err}
	return strconv.ParseInt(strings.TrimSpace(fmt.Sprintf("%s", output)), 10, 0)
}


func AvailableSnapshots(requireLocal bool) ([]string, error) {
	var command string
	if requireLocal { 
		command = config.Config.AvailableLocalSnapshotHostsCommand 
	} else {
		command = config.Config.AvailableSnapshotHostsCommand
	}
	output, err := commandOutput(command)
	hosts, err := outputLines(output, err)

	return hosts, err
}


func MySQLRunning() (bool, error) {
	_, err := commandOutput(config.Config.MySQLServiceStatusCommand)
	// status command exits with 0 when MySQL is running, or otherwise if not running
	return err == nil, nil
}

func MySQLStop() error {
	_, err := commandOutput(config.Config.MySQLServiceStopCommand)
	return err
}

func MySQLStart() error {
	_, err := commandOutput(config.Config.MySQLServiceStartCommand)
	return err
}


func ReceiveMySQLSeedData() error {
	directory, err := GetMySQLDataDir() 
	if err != nil {return err}

	_, err = commandOutput(fmt.Sprintf("%s %s %d", config.Config.ReceiveSeedDataCommand, directory, SeedTransferPort))
	if err != nil { return log.Errore(err)	}
	return err
}


func SendMySQLSeedData(targetHostname string, directory string) error {
	if directory == "" {
		return log.Error("Empty directory in SendMySQLSeedData")
	}
	_, err := commandOutput(fmt.Sprintf("%s %s %s %d", config.Config.SendSeedDataCommand, directory, targetHostname, SeedTransferPort))
	if err != nil { return log.Errore(err)	}
	return err
}


