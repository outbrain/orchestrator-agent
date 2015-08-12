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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/outbrain/golib/log"

	"github.com/outbrain/orchestrator-agent/config"
)

const (
	SeedTransferPort = 21234
)

var activeCommands map[string](*exec.Cmd) = make(map[string](*exec.Cmd))

// LogicalVolume describes an LVM volume
type LogicalVolume struct {
	Name            string
	GroupName       string
	Path            string
	IsSnapshot      bool
	SnapshotPercent float64
}

// Equals tests equality of this corrdinate and another one.
func (this *LogicalVolume) IsSnapshotValid() bool {
	if !this.IsSnapshot {
		return false
	}
	if this.SnapshotPercent >= 100.0 {
		return false
	}
	return true
}

// Mount describes a file system mount point
type Mount struct {
	Path           string
	Device         string
	LVPath         string
	FileSystem     string
	IsMounted      bool
	DiskUsage      int64
	MySQLDataPath  string
	MySQLDiskUsage int64
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
	log.Debugf("execCmd: %s", commandText)
	if config.Config.ExecWithSudo {
		return exec.Command("sudo", "bash", tmpFile.Name()), tmpFile.Name(), nil
	}
	return exec.Command("bash", tmpFile.Name()), tmpFile.Name(), nil
}

// commandOutput executes a command and return output bytes
func commandOutput(commandText string) ([]byte, error) {
	cmd, tmpFileName, err := execCmd(commandText)
	if err != nil {
		return nil, log.Errore(err)
	}

	outputBytes, err := cmd.Output()
	if err != nil {
		return nil, log.Errore(err)
	}
	os.Remove(tmpFileName)

	return outputBytes, nil
}

// commandRun executes a command
func commandRun(commandText string, onCommand func(*exec.Cmd)) error {
	cmd, tmpFileName, err := execCmd(commandText)
	if err != nil {
		return log.Errore(err)
	}
	onCommand(cmd)

	err = cmd.Run()
	if err != nil {
		return log.Errore(err)
	}
	os.Remove(tmpFileName)

	return nil
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
	if err != nil {
		return nil, err
	}

	logicalVolumes := []LogicalVolume{}
	for _, lineTokens := range tokens {
		logicalVolume := LogicalVolume{
			Name:      lineTokens[1],
			GroupName: lineTokens[2],
			Path:      lineTokens[3],
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
	mount := Mount{
		Path:      mountPoint,
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
	mount := Mount{
		Path:      mountPoint,
		IsMounted: false,
	}
	if volumeName == "" {
		return mount, errors.New("Empty columeName in MountLV")
	}
	fsType, err := GetLogicalVolumeFSType(volumeName)
	if err != nil {
		return mount, err
	}

	mountOptions := ""
	if fsType == "xfs" {
		mountOptions = "-o nouuid"
	}
	_, err = commandOutput(fmt.Sprintf("mount %s %s %s", mountOptions, volumeName, mountPoint))
	if err != nil {
		return mount, err
	}

	return GetMount(mountPoint)
}

func RemoveLV(volumeName string) error {
	_, err := commandOutput(fmt.Sprintf("lvremove --force %s", volumeName))
	return err
}

func CreateSnapshot() error {
	_, err := commandOutput(config.Config.CreateSnapshotCommand)
	return err
}

func Unmount(mountPoint string) (Mount, error) {
	mount := Mount{
		Path:      mountPoint,
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
	if err != nil {
		return err
	}

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

func GetMySQLDataDirAvailableDiskSpace() (int64, error) {
	directory, err := GetMySQLDataDir()
	if err != nil {
		return 0, log.Errore(err)
	}

	output, err := commandOutput(fmt.Sprintf("df -PT -B 1 %s | sed -e /^Filesystem/d", directory))
	if err != nil {
		return 0, log.Errore(err)
	}

	tokens, err := outputTokens(`[ \t]+`, output, err)
	for _, lineTokens := range tokens {
		result, err := strconv.ParseInt(lineTokens[4], 10, 0)
		return result, err
	}
	return 0, log.Errore(errors.New(fmt.Sprintf("No rows found by df in GetMySQLDataDirAvailableDiskSpace, %s", directory)))
}

// PostCopy executes a post-copy command -- after LVM copy is done, before service starts. Some cleanup may go here.
func PostCopy() error {
	_, err := commandOutput(config.Config.PostCopyCommand)
	return err
}

func HeuristicMySQLDataPath(mountPoint string) (string, error) {
	datadir, err := GetMySQLDataDir()
	if err != nil {
		return "", err
	}

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
	if err != nil {
		return 0, err
	}
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

func MySQLErrorLogTail() ([]string, error) {
	output, err := commandOutput(`tail -n 20 $(egrep "log[-_]error" /etc/my.cnf | cut -d "=" -f 2)`)
	tail, err := outputLines(output, err)
	return tail, err
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

func ReceiveMySQLSeedData(seedId string) error {
	directory, err := GetMySQLDataDir()
	if err != nil {
		return log.Errore(err)
	}

	err = commandRun(
		fmt.Sprintf("%s %s %d", config.Config.ReceiveSeedDataCommand, directory, SeedTransferPort),
		func(cmd *exec.Cmd) {
			activeCommands[seedId] = cmd
			log.Debug("ReceiveMySQLSeedData command completed")
		})
	if err != nil {
		return log.Errore(err)
	}

	return err
}

func SendMySQLSeedData(targetHostname string, directory string, seedId string) error {
	if directory == "" {
		return log.Error("Empty directory in SendMySQLSeedData")
	}
	err := commandRun(fmt.Sprintf("%s %s %s %d", config.Config.SendSeedDataCommand, directory, targetHostname, SeedTransferPort),
		func(cmd *exec.Cmd) {
			activeCommands[seedId] = cmd
			log.Debug("SendMySQLSeedData command completed")
		})
	if err != nil {
		return log.Errore(err)
	}
	return err
}

func SeedCommandCompleted(seedId string) bool {
	if cmd, ok := activeCommands[seedId]; ok {
		if cmd.ProcessState != nil {
			return cmd.ProcessState.Exited()
		}
	}
	return false
}

func SeedCommandSucceeded(seedId string) bool {
	if cmd, ok := activeCommands[seedId]; ok {
		if cmd.ProcessState != nil {
			return cmd.ProcessState.Success()
		}
	}
	return false
}

func AbortSeed(seedId string) error {
	if cmd, ok := activeCommands[seedId]; ok {
		log.Debugf("Killing process %d", cmd.Process.Pid)
		return cmd.Process.Kill()
	} else {
		log.Debug("Not killing: Process not found")
	}
	return nil
}
