<h1 align="center">snare</h1>

<p align="center">
  <strong>SNMP facade for Netgear Plus / Easy Smart switches</strong><br />
  GS308EP and kin.
</p>

<p align="center">
  <a href="https://github.com/kgrubb/snare/actions/workflows/ci.yml"><img src="https://github.com/kgrubb/snare/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/kgrubb/snare/actions/workflows/release.yml"><img src="https://github.com/kgrubb/snare/actions/workflows/release.yml/badge.svg" alt="Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg" alt="AGPL-3.0" /></a>
  <img src="https://img.shields.io/github/go-mod/go-version/kgrubb/snare" alt="Go" />
  <a href="https://github.com/kgrubb/snare/pkgs/container/snare"><img src="https://img.shields.io/badge/ghcr-snare-blue.svg" alt="GHCR" /></a>
</p>

---

These switches ship a web UI and no SNMP agent. snare scrapes that UI the same way [py-netgear-plus](https://github.com/foxey/py-netgear-plus) and ProSAFE exporters do, then serves a real SNMP agent so your NMS can poll and receive traps.

## Protocols

| Protocol | Support |
|---|---|
| SNMPv2c | community GET / GETNEXT / GETBULK |
| SNMPv3 | USM authPriv (MD5/SHA + DES/AES) |
| SNMPv1 | not implemented (use v2c) |
| Traps | SNMPv2c `linkUp` / `linkDown` on port flips |

## MIB layout

Mirrors what Netgear Smart Managed Pro (for example GS310TP) exposes for the data we can scrape:

| MIB | OID | Content |
|---|---|---|
| SNMPv2-MIB | `1.3.6.1.2.1.1` | sysDescr, sysObjectID, sysUpTime, sysName, … |
| IF-MIB | `1.3.6.1.2.1.2` / `.31` | ifTable + ifXTable (octets, errors, oper status) |
| POWER-ETHERNET-MIB | `1.3.6.1.2.1.105` | PoE admin, detection, class, budget |

`sysObjectID` is `1.3.6.1.4.1.4526.100.308` (Netgear enterprise, Plus-facade leaf).

## Quick start

```bash
cp config.example.yaml config.yaml
# set switch.url + switch.password
go run ./cmd/snare --config config.yaml
```

```bash
snmpwalk -v2c -c public 127.0.0.1:1161 1.3.6.1.2.1.1
snmpwalk -v2c -c public 127.0.0.1:1161 1.3.6.1.2.1.2.2.1
snmpwalk -v2c -c public 127.0.0.1:1161 1.3.6.1.2.1.105
```

### Container

```bash
docker run --rm -p 161:161/udp \
  -v "$PWD/config.yaml:/config/config.yaml:ro" \
  ghcr.io/kgrubb/snare:latest
```

Image base: `gcr.io/distroless/static-debian12:nonroot`.

[Release Please](https://github.com/googleapis/release-please) opens a release PR
from conventional commits (semver + `CHANGELOG.md`). Merging that PR tags
`vX.Y.Z` and writes the GitHub release notes. GoReleaser then renames the
release to `snare-X.Y.Z`, attaches binaries, and publishes multi-arch GHCR
images.

## Kubernetes

```bash
kubectl -n snare create secret generic snare-config \
  --from-file=config.yaml=./config.yaml \
  --dry-run=client -o yaml | kubectl apply -f -

helm repo add kgrubb-snare https://kgrubb.github.io/snare
helm repo update
helm install snare kgrubb-snare/snare -n snare --create-namespace \
  --set existingSecret=snare-config
```

Or from a checkout: `helm install snare ./charts/snare --set existingSecret=snare-config`.

The chart ships a UDP LoadBalancer on port 161. The pod listens on 1161
(distroless nonroot cannot bind privileged ports), so `config.yaml` must set
`listen: ":1161"` (or match `service.targetPort`). Point your NMS at the
Service IP, not the Plus switch. Bump `charts/snare/Chart.yaml` `version` on
every chart change so chart-releaser publishes it.

## Scope

PoC target is **GS308EP** (GS30xEP CGI: `login.cgi`, `dashboard.cgi`, `portStatistics.cgi`, `PoEPortConfig.cgi`, `getPoePortStatus.cgi`). Other Plus models may work if they share that UI generation.

License: [AGPL-3.0](LICENSE).
