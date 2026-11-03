# Changelog

## [1.2.1](https://github.com/kgrubb/snare/compare/v1.2.0...v1.2.1) (2026-11-03)


### Fixed

* serve SNMP requests concurrently ([#24](https://github.com/kgrubb/snare/issues/24)) ([184e1d4](https://github.com/kgrubb/snare/commit/184e1d4b78edf110895c42904d77b0b56135cffb))

## [1.2.0](https://github.com/kgrubb/snare/compare/v1.1.4...v1.2.0) (2026-11-02)


### Added

* publish LLDP rem table from declared topology neighbors ([#21](https://github.com/kgrubb/snare/issues/21)) ([6193ed7](https://github.com/kgrubb/snare/commit/6193ed796b890733f6e911fab72e434d59694160))


### Fixed

* silence goreleaser dockers deprecation warning ([#20](https://github.com/kgrubb/snare/issues/20)) ([0313d3e](https://github.com/kgrubb/snare/commit/0313d3e6901626fab4ca4e1d187b3f6a0e6c328f))

## [1.1.4](https://github.com/kgrubb/snare/compare/v1.1.3...v1.1.4) (2026-11-02)


### Fixed

* stub full IF-MIB ifEntry/ifXEntry for NMS multi-GETs ([#18](https://github.com/kgrubb/snare/issues/18)) ([c962f4d](https://github.com/kgrubb/snare/commit/c962f4da19229768902ef907f6fd7b61b58fe50a))

## [1.1.3](https://github.com/kgrubb/snare/compare/v1.1.2...v1.1.3) (2026-10-31)


### Fixed

* logout on SIGTERM and back off on session limit ([#16](https://github.com/kgrubb/snare/issues/16)) ([afdadfd](https://github.com/kgrubb/snare/commit/afdadfd224336a8824fe178508002bf2e9c97789))

## [1.1.2](https://github.com/kgrubb/snare/compare/v1.1.1...v1.1.2) (2026-10-31)


### Fixed

* reuse GS308EP SID instead of logging in every poll ([#14](https://github.com/kgrubb/snare/issues/14)) ([2b8b3c9](https://github.com/kgrubb/snare/commit/2b8b3c922421b3aede59569fa05b0b83b1df6742))

## [1.1.1](https://github.com/kgrubb/snare/compare/v1.1.0...v1.1.1) (2026-10-29)


### Fixed

* bump chart-releaser-action for skip_packaging ([#12](https://github.com/kgrubb/snare/issues/12)) ([5294a0b](https://github.com/kgrubb/snare/commit/5294a0b6a5d8ee6c309a58a07cd1c32d9e50f818))

## [1.1.0](https://github.com/kgrubb/snare/compare/v1.0.1...v1.1.0) (2026-10-29)


### Added

* add in-repo Helm chart ([#10](https://github.com/kgrubb/snare/issues/10)) ([5c82819](https://github.com/kgrubb/snare/commit/5c828190acb662a0e69417c69e3d0a6ee4c719be))

## [1.0.1](https://github.com/kgrubb/snare/compare/v1.0.0...v1.0.1) (2026-10-28)


### Fixed

* include Dependabot dependency bumps in the changelog ([#7](https://github.com/kgrubb/snare/issues/7)) ([f9d7835](https://github.com/kgrubb/snare/commit/f9d7835b65ae4851d29fd7b3bca3f835566bdfbc))
* traps, PoE defaults, dead parse code, and Go badge ([#3](https://github.com/kgrubb/snare/issues/3)) ([09e627e](https://github.com/kgrubb/snare/commit/09e627e902c10486af0f9e0c579cdfb05c122655))


### Dependencies

* **deps:** bump github.com/sirupsen/logrus from 1.9.0 to 1.9.1 ([#4](https://github.com/kgrubb/snare/issues/4)) ([c05f35d](https://github.com/kgrubb/snare/commit/c05f35df3f92c4a05de88f7570a5bcd41156680c))

## 1.0.0 (2026-10-28)


### Added

* initial snare SNMP facade for Netgear Plus switches ([e81489f](https://github.com/kgrubb/snare/commit/e81489f25158adb773b375d2622554a2b14ebc4f))

## Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
