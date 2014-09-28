orchestrator-agent
==================

MySQL topology agent (daemon)

**orchestrator-agent** is a sub-project of [orchestrator](https://github.com/outbrain/orchestrator).
It is a service that runs on MySQL hosts and communicates with *orchestrator*.

**orchestrator-agent** is capable of proving operating system, file system and LVM information to *orchestrator*, as well
as invoke certain commands and scripts.

The primary drive for developing **orchestrator-agent** was [Outbrain](https://github.com/outbrain)'s need for a controlled
seeding mechanism, where MySQL instances would duplicate onto new/corrupt installations. 
As such, **orchestrator-agent**'s functionality is tightly coupled with the backup/seed mechanisms used by Outbrain, 
as described following. Though easily extendible, **orchestrator-agent** was not developed with a general purpose
backup/restore/orchestration capabilities in mind.

### Generic functionality offered by **orchestrator-agent**:

- Detection of the MySQL service, starting and stopping (start/stop/status commands provided via configuration)
- Detection of MySQL port, data directory (assumes configuration is `/etc/my.cnf`)
- Calculation of disk usage on data directory mount point
- Tailing the error log file
 
### Specialized functionality offered by **orchestrator-agent**:

- Detection of LVM snapshots on MySQL host (snapshots that are MySQL specific)
- Mounting/umounting of LVM snapshots
- 

### Requirements:

- Linux, 64bit. Tested on CentOS 5 and Ubuntu Server 12.04+
-  

 