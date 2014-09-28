orchestrator-agent
==================

MySQL topology agent (daemon)

**orchestrator-agent** is a sub-project of [orchestrator](https://github.com/outbrain/orchestrator).
It is a service that runs on MySQL hosts and communicates with *orchestrator*.

**orchestrator-agent** is capable of proving operating system, file system and LVM information to *orchestrator*, as well
as invoke certain commands and scripts.

The primary drive for developing **orchestrator-agent** was [Outbrain](https://github.com/outbrain)'s need for a controlled
seeding mechanism, where MySQL instances would duplicate onto new/corrupt installations. 
As such, **orchestrator-agent**'s functionality is tightly coupled with the backup/seed mechanisms used by Outbrain.