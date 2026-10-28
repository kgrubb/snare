# snare

**SNMP facade for Netgear Plus / Easy Smart switches** (GS308EP and kin).

These switches expose a web UI and no SNMP agent. snare scrapes that UI the
same way [py-netgear-plus](https://github.com/foxey/py-netgear-plus) and
ProSAFE exporters do, then serves a real SNMP agent so NMS tools can poll and
receive traps.

License: [AGPL-3.0](LICENSE).

## Speak

| Protocol | Support |
|---|---|
| SNMPv2c | community GET / GETNEXT / GETBULK |
| SNMPv3 | USM authPriv (MD5/SHA + DES/AES) |
| SNMPv1 | not implemented (use v2c) |
| Traps | SNMPv2c `linkUp` / `linkDown` on port flips |

## MIB layout

Mirrors what Netgear Smart Managed Pro (e.g. GS310TP) exposes for the data we
can actually scrape:

| MIB | OID | Content |
|---|---|---|
| SNMPv2-MIB | `1.3.6.1.2.1.1` | sysDescr, sysObjectID, sysUpTime, sysName, … |
| IF-MIB | `1.3.6.1.2.1.2` / `.31` | ifTable + ifXTable (octets, errors, oper status) |
| POWER-ETHERNET-MIB | `1.3.6.1.2.1.105` | PoE admin, detection, class, budget |

`sysObjectID` is `1.3.6.1.4.1.4526.100.308` (Netgear enterprise, Plus-facade leaf).

## Run

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

Image is `gcr.io/distroless/static-debian12:nonroot`.

## Releases

Tagged `v*` pushes run GoReleaser: multi-arch binaries + GHCR images.

## Kubernetes sketch

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: snare
spec:
  replicas: 1
  selector:
    matchLabels: {app: snare}
  template:
    metadata:
      labels: {app: snare}
    spec:
      containers:
        - name: snare
          image: ghcr.io/kgrubb/snare:latest
          args: ["--config", "/config/config.yaml"]
          ports:
            - containerPort: 161
              protocol: UDP
          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
      volumes:
        - name: config
          secret:
            secretName: snare-config
---
apiVersion: v1
kind: Service
metadata:
  name: snare
spec:
  type: LoadBalancer
  selector: {app: snare}
  ports:
    - port: 161
      protocol: UDP
```

Point your Netgear NMS / snmp reader at the Service IP, not the Plus switch.

## Scope

PoC targets **GS308EP** (GS30xEP CGI: `login.cgi`, `dashboard.cgi`,
`portStatistics.cgi`, `PoEPortConfig.cgi`, `getPoePortStatus.cgi`). Other Plus
models may work if they share that UI generation.
