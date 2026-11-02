<h1 align="center">snare</h1>

<p align="center">
  <a href="https://github.com/kgrubb/snare/actions/workflows/ci.yml"><img src="https://github.com/kgrubb/snare/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/kgrubb/snare/actions/workflows/release.yml"><img src="https://github.com/kgrubb/snare/actions/workflows/release.yml/badge.svg" alt="Release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue.svg" alt="AGPL-3.0" /></a>
  <img src="https://img.shields.io/github/go-mod/go-version/kgrubb/snare" alt="Go" />
  <a href="https://github.com/kgrubb/snare/pkgs/container/snare"><img src="https://img.shields.io/badge/ghcr-snare-blue.svg" alt="GHCR" /></a>
</p>

Netgear Plus switches (GS308EP and friends) ship a web UI and no SNMP agent. snare scrapes that UI and serves SNMPv2c or SNMPv3 so an NMS can poll and receive traps.

## Protocols

| Protocol | Support |
|---|---|
| SNMPv2c | community GET / GETNEXT / GETBULK |
| SNMPv3 | USM authPriv (MD5/SHA + DES/AES) |
| SNMPv1 | not implemented |
| Traps | SNMPv2c `linkUp` / `linkDown` |

## MIB layout

| MIB | OID | Content |
|---|---|---|
| SNMPv2-MIB | `1.3.6.1.2.1.1` | system group |
| IF-MIB | `1.3.6.1.2.1.2` / `.31` | ifEntry + ifXEntry (scraped octets, errors, status) |
| POWER-ETHERNET-MIB | `1.3.6.1.2.1.105` | PoE admin, detection, class, budget |

`sysObjectID` is `1.3.6.1.4.1.4526.100.308`.

## Quick start

```bash
cp config.example.yaml config.yaml
# set switch.url and switch.password
go run ./cmd/snare --config config.yaml

snmpwalk -v2c -c public 127.0.0.1:1161 1.3.6.1.2.1.1
```

## Container

```bash
docker run --rm -p 161:161/udp \
  -v "$PWD/config.yaml:/config/config.yaml:ro" \
  ghcr.io/kgrubb/snare:latest
```

## Kubernetes

```bash
helm repo add kgrubb-snare https://kgrubb.github.io/snare
helm install snare kgrubb-snare/snare -n snare --create-namespace \
  --set config.switch.url=http://192.0.2.10 \
  --set config.switch.password=changeme
```

Point the NMS at the Service IP on UDP 161. To load config from a Secret instead, set `existingSecret`.

## License

[AGPL-3.0](LICENSE)
