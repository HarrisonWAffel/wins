# 4. Rancher Wins Service Configuration And Binary Upgrades

Date: 2024-8-16

## Status

Draft

## Context

To properly facilitate the maintenance of rancher-wins on existing nodes, as well as to begin transitioning away from the wins named pipe, the existing SUC image will be migrated to Golang. This will increase the maintainability of the SUC image and reduce the size of the `rancher/wins` dockerhub image substantially. Due to the introduction of Host Process containers in recent releases of Kubernetes, the rancher-wins named pipe is no longer needed, and introduces unneeded complexity when performing privileged operations on Windows nodes.

## Decision

TBD

## Consequences

The long deprecated `rancher-wins-upgrader` chart will be removed from this repository. This includes the `upgrade` command implemented in Go, the relevant Dockerfiles, dedicated PowerShell scripts, and all end-to-end test cases. The `rancher-wins-upgrader` chart specified a `catalog.cattle.io/rancher-version` annotation with the value of `>= 2.6.0-0 <= 2.6.100-0`. As Rancher v2.6 is EOL, this chart is effectively useless.

The existing `Dockerfile` used to create the `rancher/wins` dockerhub image will be updated to use a `nanoserver` base image, reducing the image size from ~2GB to ~120MB. 

