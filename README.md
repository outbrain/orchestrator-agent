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

Whether or not **orchestrator-agent** is useful to you depends on your needs.

##### Generic functionality offered by **orchestrator-agent**:

- Detection of the MySQL service, starting and stopping (start/stop/status commands provided via configuration)
- Detection of MySQL port, data directory (assumes configuration is `/etc/my.cnf`)
- Calculation of disk usage on data directory mount point
- Tailing the error log file
- Discovery (the mere existence of the *orchestrator-agent* service on a host may suggest the existence or need of existence of a MySQL service)
 
##### Specialized functionality offered by **orchestrator-agent**:

- Detection of LVM snapshots on MySQL host (snapshots that are MySQL specific)
- Creation of new snapshots
- Mounting/umounting of LVM snapshots
- Detection of DC-local and DC-agnostic snapshots available for a given cluster
- Transmitting/receiving seed data

#### The Outbrain seed method

The following does not discuss hard-backups where binary/logical data is written to an external/remote disk or a tape.

At Outbrain we use LVM snapshots. Each MySQL replication topology has designated slaves which serve as snapshot servers.
These servers do LVM snapshots daily, and keep such snapshots open for a few days. Thus it is possible that a server
has, say, 5 open (and unmounted) LVM snapshots at a given time.  

Upon need, we are able to mount any such snapshot in near zero time and restart MySQL on the host using mounted data directory.
We are thus able to recover any one of few days back at speed of InnoDB crash-recovery.

This serves two purposes: an immediate recovery/sanity check for destructive operations (unintentional or malicious `UPDATE` or `DROP`)
as well as a seed source for new/corrupt servers.

The choice of snapshot slaves is not random; we have multiple data centers and we keep at least one snapshot server per topology per data center,
such that upon need we can make DC-local copies. For this purpose, the daily snapshot process reports, upon success, the
availability of the snapshot along with any metadata required on cluster/DC.

**orchestrator-agent** thus depends on external (configurable) commands to:

- Detect where in local and remote DCs it can find an appropriate snapshot
- Find said snapshot on server, mount it
- Stop MySQL on target host, clear data on MySQL data directory
- Initiate send/receive process
- Cleanup data after copy (e.g. remove `.pid` files if any)
- Unmount snapshot
- etc.
  

#### Requirements:

- Linux, 64bit. Tested on CentOS 5 and Ubuntu Server 12.04+


Authored by [Shlomi Noach](https://github.com/shlomi-noach) at [Outbrain](https://github.com/outbrain)



 