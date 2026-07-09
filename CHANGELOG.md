## [2.71.1](https://github.com/danielvm-git/bigbase/compare/v2.71.0...v2.71.1) (2026-07-09)


### Bug Fixes

* **proxy:** stabilize TestProxyAuthPolicy against startup race ([cf89868](https://github.com/danielvm-git/bigbase/commit/cf898685d8186910047f2c0e71e9dcf237402572))

# [2.71.0](https://github.com/danielvm-git/bigbase/compare/v2.70.0...v2.71.0) (2026-07-09)


### Features

* **proxy:** implement site route auth policy, path matching, and passthrough identity header injection ([e236031](https://github.com/danielvm-git/bigbase/commit/e23603145a3cd159a674133ae51f3c1ed8ddb4a6))

# [2.70.0](https://github.com/danielvm-git/bigbase/compare/v2.69.0...v2.70.0) (2026-07-08)


### Features

* **auth:** DB-backed OTP store & rate limiting with audit logging ([b0c3e43](https://github.com/danielvm-git/bigbase/commit/b0c3e43108d3183b177f1206c67572b5076d4696))

# [2.69.0](https://github.com/danielvm-git/bigbase/compare/v2.68.0...v2.69.0) (2026-07-08)


### Features

* **mcp:** implement MCP bearer authentication and scope verification ([642e3db](https://github.com/danielvm-git/bigbase/commit/642e3db2070618b0007af46d84f5bda541db92e6))

# [2.68.0](https://github.com/danielvm-git/bigbase/compare/v2.67.0...v2.68.0) (2026-07-07)


### Features

* **mcp:** add site discovery and deploy key lifecycle tools ([890a5e7](https://github.com/danielvm-git/bigbase/commit/890a5e7ab67398002355cbd51e727dd55bb2aef5)), closes [#57](https://github.com/danielvm-git/bigbase/issues/57)

# [2.67.0](https://github.com/danielvm-git/bigbase/compare/v2.66.1...v2.67.0) (2026-07-07)


### Bug Fixes

* **auth:** prevent accidental JSON serialization of site key raw token ([4aec994](https://github.com/danielvm-git/bigbase/commit/4aec994f5c305629f3aa7809695638628b541b3f))


### Features

* **deploy:** inject native DB connection string into app env ([6d31b46](https://github.com/danielvm-git/bigbase/commit/6d31b46384aecb7293b76655863ab1623bfa6c46))

## [2.66.1](https://github.com/danielvm-git/bigbase/compare/v2.66.0...v2.66.1) (2026-07-07)


### Bug Fixes

* **auth:** prevent accidental JSON serialization of site key raw token ([25b4b29](https://github.com/danielvm-git/bigbase/commit/25b4b2950d8d4c84437e4491036d2505a550fb1c))

# [2.66.0](https://github.com/danielvm-git/bigbase/compare/v2.65.0...v2.66.0) (2026-07-07)


### Features

* **mcp:** add provisioning toolchain — create_repo, create_site, provision_ci_credentials, get_ci_template ([6ba8ea4](https://github.com/danielvm-git/bigbase/commit/6ba8ea4a275faa75c9a55c1fdfaf6dc2494cadac))

# [2.65.0](https://github.com/danielvm-git/bigbase/compare/v2.64.1...v2.65.0) (2026-07-07)


### Features

* **kernel:** add WithProjectID and ProjectIDFromContext helpers ([e2802f7](https://github.com/danielvm-git/bigbase/commit/e2802f7ead53a8e7da858909eccbcfcb6ec9ed2f))

## [2.64.1](https://github.com/danielvm-git/bigbase/compare/v2.64.0...v2.64.1) (2026-07-07)


### Bug Fixes

* **sites:** batch latest deployment lookup to fix slow incomplete list ([34d6c51](https://github.com/danielvm-git/bigbase/commit/34d6c51d46680c3abe813b7ff1346d8bc8037a12))

# [2.64.0](https://github.com/danielvm-git/bigbase/compare/v2.63.0...v2.64.0) (2026-06-30)


### Features

* **ui:** add zap icon for Events nav item ([dc83320](https://github.com/danielvm-git/bigbase/commit/dc83320590b47f3a98fd7368bee0f5aa19ee9f63))

# [2.63.0](https://github.com/danielvm-git/bigbase/compare/v2.62.2...v2.63.0) (2026-06-30)


### Features

* **ui:** add Events nav item to sidebar DevOps section ([4fc4c75](https://github.com/danielvm-git/bigbase/commit/4fc4c7531fdb9e506546c5ca895ac4ab7f7bdf6d))

## [2.62.2](https://github.com/danielvm-git/bigbase/compare/v2.62.1...v2.62.2) (2026-06-29)


### Bug Fixes

* **sdk:** pass token to AuthClient in Astro middleware ([1674400](https://github.com/danielvm-git/bigbase/commit/1674400089b303d53317963ac95924b6a58f7758))

## [2.62.1](https://github.com/danielvm-git/bigbase/compare/v2.62.0...v2.62.1) (2026-06-29)


### Bug Fixes

* **status:** mark e46 (custom domains + ACME) as done — all stories complete, code verified ([69a2a9e](https://github.com/danielvm-git/bigbase/commit/69a2a9e5ff6320ef6e273932c371476da10165c9))

# [2.62.0](https://github.com/danielvm-git/bigbase/compare/v2.61.0...v2.62.0) (2026-06-28)


### Bug Fixes

* **ui:** resolve TS compilation errors in Tooltip & test, mark e51 as done in epic.yaml ([9ec1af5](https://github.com/danielvm-git/bigbase/commit/9ec1af52610f4812b224c9dd46925a494b437019))


### Features

* **ui:** add design system foundation — tokens, 22 components, a11y audit ([6fb588e](https://github.com/danielvm-git/bigbase/commit/6fb588e63f79ce75dad9a27c5cd8dc0bf796f27c))

# [2.61.0](https://github.com/danielvm-git/bigbase/compare/v2.60.1...v2.61.0) (2026-06-28)


### Bug Fixes

* **auth:** validate programmatic Options.Secret and fix refresh rotation race ([ab09c96](https://github.com/danielvm-git/bigbase/commit/ab09c961b573174ae1efde0f8aaea913c56399df))


### Features

* **auth:** e50 JWT secret persistence, configurable lifetimes, refresh token revocation ([cd72c6a](https://github.com/danielvm-git/bigbase/commit/cd72c6a2b3fa3f9bcc5c4c967f06cb8b6c817abd))

## [2.60.1](https://github.com/danielvm-git/bigbase/compare/v2.60.0...v2.60.1) (2026-06-28)


### Bug Fixes

* **specs:** escape colon in release-plan YAML note field ([c6eca05](https://github.com/danielvm-git/bigbase/commit/c6eca05c2f0f747ba4e1e965dab7e984212b50b8))

# [2.60.0](https://github.com/danielvm-git/bigbase/compare/v2.59.1...v2.60.0) (2026-06-28)


### Bug Fixes

* **auth:** use configured PublicURL for OAuth redirect URIs ([3a86390](https://github.com/danielvm-git/bigbase/commit/3a86390d0459d17210d2c86e3184c807ed320ce9))
* **security:** address e49 code review findings ([0aac517](https://github.com/danielvm-git/bigbase/commit/0aac517a5e0aafa56630a264ae44d084e491d167)), closes [#1](https://github.com/danielvm-git/bigbase/issues/1) [#2](https://github.com/danielvm-git/bigbase/issues/2) [#3](https://github.com/danielvm-git/bigbase/issues/3) [#4](https://github.com/danielvm-git/bigbase/issues/4) [#5](https://github.com/danielvm-git/bigbase/issues/5) [#6](https://github.com/danielvm-git/bigbase/issues/6) [#7](https://github.com/danielvm-git/bigbase/issues/7)
* **storage:** add path traversal defense to file downloads ([41dd50b](https://github.com/danielvm-git/bigbase/commit/41dd50b430e6abbf1740c58c794911155965526d))


### Features

* **auth:** fix anonymous tokens with Claims struct and middleware bypass ([9d587c8](https://github.com/danielvm-git/bigbase/commit/9d587c8f8bcd8ec5d4d49de6eb8c82cc439fe917))

## [2.59.1](https://github.com/danielvm-git/bigbase/compare/v2.59.0...v2.59.1) (2026-06-28)


### Bug Fixes

* **security:** resolve case-insensitive git path bypass, health timing leak, and admin CSP headers ([4b11c70](https://github.com/danielvm-git/bigbase/commit/4b11c703d7177e5e9bb001298c12a1e308a319f5))
* **specs:** prevent validate-specs-yaml crash on capsule task paths ([9efd8cd](https://github.com/danielvm-git/bigbase/commit/9efd8cdc96308eb082daf7488b6e33b46c8447a7))

# [2.59.0](https://github.com/danielvm-git/bigbase/compare/v2.58.0...v2.59.0) (2026-06-27)


### Features

* **security:** DAST baseline scan and header audit scripts ([#52](https://github.com/danielvm-git/bigbase/issues/52)) ([0f0d58b](https://github.com/danielvm-git/bigbase/commit/0f0d58bc5d0be1d0171d371d408a1b7c928e93bf))

# [2.58.0](https://github.com/danielvm-git/bigbase/compare/v2.57.0...v2.58.0) (2026-06-27)


### Features

* **proxy:** add Permissions-Policy, Cache-Control headers and CI scanning preflight ([#51](https://github.com/danielvm-git/bigbase/issues/51)) ([f5b5a32](https://github.com/danielvm-git/bigbase/commit/f5b5a32c881880f7cb6201db3d5399cdffcdb1d3))

# [2.57.0](https://github.com/danielvm-git/bigbase/compare/v2.56.0...v2.57.0) (2026-06-27)


### Features

* **proxy:** block .git exposure and harden /health with Bearer auth ([#50](https://github.com/danielvm-git/bigbase/issues/50)) ([085db49](https://github.com/danielvm-git/bigbase/commit/085db4948cf01f1fa6f1399ba124ce0e542dad90))

# [2.56.0](https://github.com/danielvm-git/bigbase/compare/v2.55.1...v2.56.0) (2026-06-27)


### Bug Fixes

* **config:** harden FlagOrEnvBool test coverage and add convention comment ([1dc9c64](https://github.com/danielvm-git/bigbase/commit/1dc9c64b3f5c0bfe09983cae590014ccc5d13a84))


### Features

* **auth:** wire rate limiter with graceful shutdown and env prefix ([57829a7](https://github.com/danielvm-git/bigbase/commit/57829a79851daff81804d4a16ec95008f4caab6c))

## [2.55.1](https://github.com/danielvm-git/bigbase/compare/v2.55.0...v2.55.1) (2026-06-27)


### Bug Fixes

* **state:** restore e47 done state after rebase ([e560b35](https://github.com/danielvm-git/bigbase/commit/e560b35be6401f5b9ba1f8e77d765194360a62e1))

# [2.55.0](https://github.com/danielvm-git/bigbase/compare/v2.54.2...v2.55.0) (2026-06-27)


### Features

* **e47:** wire rate limiter with CLI flags and env fallbacks ([64a8888](https://github.com/danielvm-git/bigbase/commit/64a8888eeb99fe75cd0c2c4a5cae89436460d706))

## [2.54.2](https://github.com/danielvm-git/bigbase/compare/v2.54.1...v2.54.2) (2026-06-27)


### Bug Fixes

* **e46:** address 10 post-review findings on custom domains ([742be86](https://github.com/danielvm-git/bigbase/commit/742be869d5ed1138090ffc6466826ac7fc7eb7bb))

## [2.54.1](https://github.com/danielvm-git/bigbase/compare/v2.54.0...v2.54.1) (2026-06-27)


### Bug Fixes

* **e46:** address 10 post-review findings on custom domains ([781e4e9](https://github.com/danielvm-git/bigbase/commit/781e4e9f60a3c6496b91f4a5ddca9c7613518469))

# [2.54.0](https://github.com/danielvm-git/bigbase/compare/v2.53.0...v2.54.0) (2026-06-27)


### Features

* **ui:** add custom domain management tab to SiteDetailPage (e46s03) ([cad5956](https://github.com/danielvm-git/bigbase/commit/cad59563567bd3e00e0d07c353ba6d216ebb3fbd))

# [2.53.0](https://github.com/danielvm-git/bigbase/compare/v2.52.0...v2.53.0) (2026-06-27)


### Features

* **proxy:** auto SSL via Let's Encrypt ACME + HTTP to HTTPS redirect (e46s02) ([34aba05](https://github.com/danielvm-git/bigbase/commit/34aba05233196ca6b7c99002e1f4b42470bdc56c))

# [2.52.0](https://github.com/danielvm-git/bigbase/compare/v2.51.0...v2.52.0) (2026-06-27)


### Features

* **deploy:** custom domain proxy routing for deployments (e46s01) ([e84c60a](https://github.com/danielvm-git/bigbase/commit/e84c60aed6a16d7b02beafd4a9a7da9e3934a5c6))

# [2.51.0](https://github.com/danielvm-git/bigbase/compare/v2.50.1...v2.51.0) (2026-06-27)


### Features

* **deploy:** zero-downtime connection draining on deployment switchover (e45) ([40287fb](https://github.com/danielvm-git/bigbase/commit/40287fbc5ea901d5337adc152abe67c04b113dd1))

## [2.50.1](https://github.com/danielvm-git/bigbase/compare/v2.50.0...v2.50.1) (2026-06-26)


### Bug Fixes

* **ui:** add missing tests for e44s02 rollback functions ([a0e8852](https://github.com/danielvm-git/bigbase/commit/a0e8852094a08009ba368066d9b5636e49448b05))

# [2.50.0](https://github.com/danielvm-git/bigbase/compare/v2.49.0...v2.50.0) (2026-06-26)


### Features

* **ui:** add rollback button to deployment detail + rollback history timeline ([b97fd9f](https://github.com/danielvm-git/bigbase/commit/b97fd9f17fe4f8f9feb34ea9d32cb2c43c5c7c8a))

# [2.49.0](https://github.com/danielvm-git/bigbase/compare/v2.48.0...v2.49.0) (2026-06-26)


### Features

* **deploy:** add one-click rollback endpoint + artifact reuse ([#49](https://github.com/danielvm-git/bigbase/issues/49)) ([77132c7](https://github.com/danielvm-git/bigbase/commit/77132c782be2345c6e399456065d8e129419e6e1))

# [2.48.0](https://github.com/danielvm-git/bigbase/compare/v2.47.0...v2.48.0) (2026-06-26)


### Features

* **deploy:** semantic health check — probe before marking deployment live ([9b46836](https://github.com/danielvm-git/bigbase/commit/9b468360a23e4f4955d9a5209e5ddf35eb8f7b82))

# [2.47.0](https://github.com/danielvm-git/bigbase/compare/v2.46.0...v2.47.0) (2026-06-25)


### Features

* **deploy:** cache management UI — per-site tab + global panel (e42s02) ([#48](https://github.com/danielvm-git/bigbase/issues/48)) ([a173b3f](https://github.com/danielvm-git/bigbase/commit/a173b3f7599a22033ba71537b28f17950a32a8de)), closes [#47](https://github.com/danielvm-git/bigbase/issues/47)

# [2.46.0](https://github.com/danielvm-git/bigbase/compare/v2.45.1...v2.46.0) (2026-06-25)


### Features

* **deploy:** build dependency cache — skip npm install on lockfile cache hit (e42s01) ([#47](https://github.com/danielvm-git/bigbase/issues/47)) ([29238a6](https://github.com/danielvm-git/bigbase/commit/29238a6334c45612507ce8dbe4cb41de61d032d0))

## [2.45.1](https://github.com/danielvm-git/bigbase/compare/v2.45.0...v2.45.1) (2026-06-25)


### Bug Fixes

* **env-vars:** harden secret storage — server-side masking, shared crypto, and error propagation ([d62388d](https://github.com/danielvm-git/bigbase/commit/d62388d7459e1269381963269d5b9a1e202b2e40))

# [2.45.0](https://github.com/danielvm-git/bigbase/compare/v2.44.0...v2.45.0) (2026-06-24)


### Features

* **ui:** add Env Vars tab to SiteDetailPage with CRUD and .env import/export ([4686942](https://github.com/danielvm-git/bigbase/commit/4686942f76f423898f1bfdb09e6861eca80b07d2))

# [2.44.0](https://github.com/danielvm-git/bigbase/compare/v2.43.0...v2.44.0) (2026-06-24)


### Bug Fixes

* **ui:** reduce poll interval to 2s and include 'deploying' in active-status check ([a98b303](https://github.com/danielvm-git/bigbase/commit/a98b303e0173f671f3545a7e54aad71996fdb002))


### Features

* **ui:** enhance StatusTimeline with animations, Live badge, and failed indicator ([491afa4](https://github.com/danielvm-git/bigbase/commit/491afa44b825318daddbc4b6eb47326a6e1d25c0))

# [2.43.0](https://github.com/danielvm-git/bigbase/compare/v2.42.0...v2.43.0) (2026-06-24)


### Features

* **deploy:** introduce Supervisor seam — Runner/Instance/Spec interfaces + restart policy (e53s01-s02) ([a0d6bb7](https://github.com/danielvm-git/bigbase/commit/a0d6bb7f6908142b34ab0158d3630aea2b321094)), closes [#40](https://github.com/danielvm-git/bigbase/issues/40)
* **deploy:** Supervisor restart loop with crash-loop detection (e53s03) ([5a9dd77](https://github.com/danielvm-git/bigbase/commit/5a9dd7718233a72467f14dec300ed08168df619e))
* **deploy:** wire Supervisor into resume path — static apps supervised on restart (e53s04) ([d195bd4](https://github.com/danielvm-git/bigbase/commit/d195bd4ed0302937f57b00361f4e93868acd0d46))

# [2.42.0](https://github.com/danielvm-git/bigbase/compare/v2.41.0...v2.42.0) (2026-06-23)


### Bug Fixes

* **ci:** resolve CI failures — test branch assumption + stale allure action SHA ([55b6dd2](https://github.com/danielvm-git/bigbase/commit/55b6dd20d589be1f1f29e0216485f938a2136051))
* **monitoring:** init NR agent before logger so log forwarding actually works ([8487171](https://github.com/danielvm-git/bigbase/commit/84871717e9928f5d5df356d2ce780cf1247eb1a2))
* **sites:** address e40s03 audit findings — error leak, DRY violation, long function, missing test ([b1daf3f](https://github.com/danielvm-git/bigbase/commit/b1daf3fa720db0bd850e6aa67fe281b1e4b3d63f))
* **sites:** set git config Dir in commitManifestToRepo ([2f9132c](https://github.com/danielvm-git/bigbase/commit/2f9132c81e82aca37ddf8ef81aa013e719fabc41))


### Features

* **monitoring:** extract buildHandler for New Relic log forwarding ([4eb28a0](https://github.com/danielvm-git/bigbase/commit/4eb28a074e8c110ce4c93ee9642284d954becfc1))

# [2.41.0](https://github.com/danielvm-git/bigbase/compare/v2.40.0...v2.41.0) (2026-06-22)


### Features

* **e40s03:** implement Admin UI manifest view and editor ([03a01e2](https://github.com/danielvm-git/bigbase/commit/03a01e23c47d24f03f45e60c952a9e64a8e3f780))

# [2.40.0](https://github.com/danielvm-git/bigbase/compare/v2.39.0...v2.40.0) (2026-06-22)


### Features

* **deploy:** add bigbase init CLI command and --manifest deploy flag ([7d419f0](https://github.com/danielvm-git/bigbase/commit/7d419f0f0c3a7cb70d8be9c82a67f4043147007c))

# [2.39.0](https://github.com/danielvm-git/bigbase/compare/v2.38.0...v2.39.0) (2026-06-22)


### Features

* **deploy:** integrate bigbase.yaml manifest into deploy flow ([0598726](https://github.com/danielvm-git/bigbase/commit/0598726c16b04d9ebe952d9ee373c7b950994098))

# [2.38.0](https://github.com/danielvm-git/bigbase/compare/v2.37.0...v2.38.0) (2026-06-21)


### Features

* **ui:** add WebSocket streaming to useBuildLogs hook with isStreaming state ([608e5a7](https://github.com/danielvm-git/bigbase/commit/608e5a7a0cd5ffc54a23446f64f2150aaa2485a8))
* **ui:** create TerminalLogViewer component and replace BuildLogs in SiteDetailPage ([512b3da](https://github.com/danielvm-git/bigbase/commit/512b3da4be82abfa3a887fdb5bc3277e1fc1e7b4))
* **ui:** enhance StreamLog with terminal toolbar, search, copy, timestamps, and ANSI color support ([b8acd65](https://github.com/danielvm-git/bigbase/commit/b8acd65841394e3ed2ec464a8458c39a0219e9f4))

# [2.37.0](https://github.com/danielvm-git/bigbase/compare/v2.36.2...v2.37.0) (2026-06-21)


### Features

* **deploy:** deployment state machine with validated transitions and events ([64ac6c5](https://github.com/danielvm-git/bigbase/commit/64ac6c5235062f7f2676f09439ba029bd20f1c4b))

## [2.36.2](https://github.com/danielvm-git/bigbase/compare/v2.36.1...v2.36.2) (2026-06-21)


### Bug Fixes

* **deploy:** resume process-based apps (Python/Go/Node SSR) after BigBase restart ([dc5b5c6](https://github.com/danielvm-git/bigbase/commit/dc5b5c643c73126a20fb303a6a98b8d0926faa20))

## [2.36.1](https://github.com/danielvm-git/bigbase/compare/v2.36.0...v2.36.1) (2026-06-21)


### Reverts

* Revert "feat(deploy): deployment state machine with validated transitions and events" ([c058782](https://github.com/danielvm-git/bigbase/commit/c058782f95b32d8aa3a74c52ac44863ad572627b))

# [2.36.0](https://github.com/danielvm-git/bigbase/compare/v2.35.0...v2.36.0) (2026-06-21)


### Features

* **deploy:** deployment state machine with validated transitions and events ([6780538](https://github.com/danielvm-git/bigbase/commit/6780538cdb4ea45e376bb9cdbcc9cfad563bbe19))

# [2.35.0](https://github.com/danielvm-git/bigbase/compare/v2.34.0...v2.35.0) (2026-06-21)


### Bug Fixes

* **deploy,monitoring:** harden PEP 668 fix and duration_seconds migration ([673a92e](https://github.com/danielvm-git/bigbase/commit/673a92e76f033a4ef16cc6439216f16d4a824431))


### Features

* **deploy:** real-time WebSocket log streaming with lifecycle integration ([56cf3e1](https://github.com/danielvm-git/bigbase/commit/56cf3e1a91e0172c0df7461538b815848c0aa0e3))

# [2.34.0](https://github.com/danielvm-git/bigbase/compare/v2.33.0...v2.34.0) (2026-06-21)


### Features

* **deploy:** add app_type override to deploy API and sites trigger ([451ea5e](https://github.com/danielvm-git/bigbase/commit/451ea5ec935246d3dc0c83e1d27e3ea3a09f87f2))


### Reverts

* **ci:** restore original semantic-release flow; remove branch protection ([95876d6](https://github.com/danielvm-git/bigbase/commit/95876d6f1da3bf3d728b0c5cac7accbc55824c84))

# [2.32.0](https://github.com/danielvm-git/bigbase/compare/v2.31.0...v2.32.0) (2026-06-20)


### Features

* **deploy:** passthrough paths, metadata injection, SPA fallback docs ([54d40cf](https://github.com/danielvm-git/bigbase/commit/54d40cfca33492782ff3af4b1cbdeb1d74167d47))

# [2.31.0](https://github.com/danielvm-git/bigbase/compare/v2.30.3...v2.31.0) (2026-06-20)


### Features

* **deploy:** build resilience — Puppeteer skip, build script validation, stdout capture, failure stats ([9e87f6f](https://github.com/danielvm-git/bigbase/commit/9e87f6fe8ecb7542397f77341841109dbf5752a0))

## [2.30.3](https://github.com/danielvm-git/bigbase/compare/v2.30.2...v2.30.3) (2026-06-20)


### Bug Fixes

* **mcp:** reuse streamable handler across requests for session persistence ([2152b67](https://github.com/danielvm-git/bigbase/commit/2152b67fb8151a0c6836b6b430351995137fb58c))

## [2.30.2](https://github.com/danielvm-git/bigbase/compare/v2.30.1...v2.30.2) (2026-06-20)


### Bug Fixes

* **mcp:** disable DNS rebinding protection for reverse-proxied MCP server ([7afffaf](https://github.com/danielvm-git/bigbase/commit/7afffaf7b0d66afd1c99a908becefc46f4210464))

## [2.30.1](https://github.com/danielvm-git/bigbase/compare/v2.30.0...v2.30.1) (2026-06-20)


### Bug Fixes

* **mcp:** allow GET /mcp for SSE connections (remove 405 method guard) ([28e05ee](https://github.com/danielvm-git/bigbase/commit/28e05ee507cedd2dda24b66c70afb9348205d401))

# [2.30.0](https://github.com/danielvm-git/bigbase/compare/v2.29.0...v2.30.0) (2026-06-20)


### Features

* **proxy:** route service hosts to internal backends ([3ed14e8](https://github.com/danielvm-git/bigbase/commit/3ed14e8485f15130b8fd80c5647c942edf77c5d6))

# [2.29.0](https://github.com/danielvm-git/bigbase/compare/v2.28.0...v2.29.0) (2026-06-20)


### Features

* **mcp:** complete epic e38 — knowledge tools, deploy workflow, agent discovery ([7275c27](https://github.com/danielvm-git/bigbase/commit/7275c27f751ffa8040f620b2686788aa75272a30))

# [2.28.0](https://github.com/danielvm-git/bigbase/compare/v2.27.0...v2.28.0) (2026-06-20)


### Features

* **mcp:** add agent discovery endpoint and deploy workflow tests ([f26120d](https://github.com/danielvm-git/bigbase/commit/f26120dc6e922a0216a5f4782dab5c88ff3c1b4c))

# [2.27.0](https://github.com/danielvm-git/bigbase/compare/v2.26.6...v2.27.0) (2026-06-20)


### Features

* **cli): add 'bigbase deploy' subcommand; feat(mcp:** deploy_site triggers real deploys ([01ab424](https://github.com/danielvm-git/bigbase/commit/01ab42437762f02884ca20ad9b0c338d2d35e409))

## [2.26.6](https://github.com/danielvm-git/bigbase/compare/v2.26.5...v2.26.6) (2026-06-20)


### Bug Fixes

* **proxy:** forward /api/* to BigBase for deployment hosts; export hostInfo ([84078d3](https://github.com/danielvm-git/bigbase/commit/84078d3308e8d66bb58958c31f3e4bec6608924a))

## [2.26.5](https://github.com/danielvm-git/bigbase/compare/v2.26.4...v2.26.5) (2026-06-20)


### Bug Fixes

* **deploy,proxy:** redeployment now replaces previous — no more stale content ([#35](https://github.com/danielvm-git/bigbase/issues/35)) ([0dd6cfb](https://github.com/danielvm-git/bigbase/commit/0dd6cfba843d26a4196d56a1adf3eb95a6a32ca2))

## [2.26.4](https://github.com/danielvm-git/bigbase/compare/v2.26.3...v2.26.4) (2026-06-19)


### Bug Fixes

* **deploy:** fix ghost running status and wrong Python detection ([5935168](https://github.com/danielvm-git/bigbase/commit/59351682e07d7daec75235903857df4dfa4e80af))

## [2.26.3](https://github.com/danielvm-git/bigbase/compare/v2.26.2...v2.26.3) (2026-06-19)


### Bug Fixes

* **ui:** fix build logs URL from /api/deployments to /api/deploy ([8ce4173](https://github.com/danielvm-git/bigbase/commit/8ce417391d038cce5f8756a41220f8be87b3eb84))

## [2.26.2](https://github.com/danielvm-git/bigbase/compare/v2.26.1...v2.26.2) (2026-06-19)


### Bug Fixes

* **deploy:** use python3 with python fallback for Python app startup ([8c342a2](https://github.com/danielvm-git/bigbase/commit/8c342a21bb4f8b83bec8c4a3d7a6ec92d96c8af5))

## [2.26.1](https://github.com/danielvm-git/bigbase/compare/v2.26.0...v2.26.1) (2026-06-19)


### Bug Fixes

* **deploy,monitoring:** fix Python deploys on Ubuntu 24.04 and alert checker column ([09d8015](https://github.com/danielvm-git/bigbase/commit/09d8015896eba91c2487c87bbeca8a8ad554810b))

# [2.26.0](https://github.com/danielvm-git/bigbase/compare/v2.25.0...v2.26.0) (2026-06-19)


### Features

* **vps:** add New Relic infrastructure agent to VPS setup script ([bea6b08](https://github.com/danielvm-git/bigbase/commit/bea6b08e69ed1fab8eb45a3100c6524a25847c8b))

# [2.25.0](https://github.com/danielvm-git/bigbase/compare/v2.24.0...v2.25.0) (2026-06-19)


### Features

* **observability:** add New Relic infrastructure agent setup script ([fa7ad96](https://github.com/danielvm-git/bigbase/commit/fa7ad961bec0859acda43230853a56217d31ac3d))

# [2.24.0](https://github.com/danielvm-git/bigbase/compare/v2.23.0...v2.24.0) (2026-06-19)


### Features

* **mcp:** register deploy workflow tools ([68ccbdd](https://github.com/danielvm-git/bigbase/commit/68ccbdda6875f218a03073876da6313727694296))

# [2.23.0](https://github.com/danielvm-git/bigbase/compare/v2.22.0...v2.23.0) (2026-06-19)


### Bug Fixes

* **sites:** handle missing columns gracefully in site deletion ([90c4aac](https://github.com/danielvm-git/bigbase/commit/90c4aac4db7030ef8e6047b315daf11224ad0bef))


### Features

* **mcp:** register knowledge tools — list_services, get_service_docs, get_code_example, list_frameworks ([71a23b5](https://github.com/danielvm-git/bigbase/commit/71a23b54e5e944feeb5932b68bb9055a1ec3b6c4))
* **mcp:** scaffold component with ping tool ([a069eac](https://github.com/danielvm-git/bigbase/commit/a069eacf8bc473956370e406b3c6320b0122b379))
* **mcp:** stdio + HTTP transport, ping tool, kernel registration ([0a00872](https://github.com/danielvm-git/bigbase/commit/0a008720c57c132037262c1b98460c407cee4ca5))

# [2.22.0](https://github.com/danielvm-git/bigbase/compare/v2.21.0...v2.22.0) (2026-06-19)


### Features

* **deploy:** add build/ detection in resumeCandidates for SvelteKit ([7f5f854](https://github.com/danielvm-git/bigbase/commit/7f5f85462e9c9a584301911e84a2e654b9e5ef59))

# [2.21.0](https://github.com/danielvm-git/bigbase/compare/v2.20.0...v2.21.0) (2026-06-19)


### Features

* **ui:** add site delete action to Sites grid and SiteDetail danger zone ([e2ba57a](https://github.com/danielvm-git/bigbase/commit/e2ba57a189f3a1f0225482aff2d06da5d9e0a6a7))

# [2.20.0](https://github.com/danielvm-git/bigbase/compare/v2.19.0...v2.20.0) (2026-06-19)


### Features

* **sites,deploy:** add deploy cleanup hook for site deletion ([a24fbee](https://github.com/danielvm-git/bigbase/commit/a24fbeeac9587733e366b763677377cf8791254d))

# [2.19.0](https://github.com/danielvm-git/bigbase/compare/v2.18.1...v2.19.0) (2026-06-19)


### Bug Fixes

* **lint:** address pre-existing lint issues in react hooks and fast refresh ([1f93769](https://github.com/danielvm-git/bigbase/commit/1f93769e16f1ee3bf8d3fbb71c297d67eb2bbcce))
* **ui:** address pre-existing lint issues in react hooks and fast refresh ([e9bfa42](https://github.com/danielvm-git/bigbase/commit/e9bfa42d2ca13d34e6ba379534edf109dc683012))
* **ui:** address pre-existing lint issues in react hooks and fast refresh ([300408f](https://github.com/danielvm-git/bigbase/commit/300408f234640c22275e5272f57d6abf7b351cc8))
* **ui:** address pre-existing lint issues in react hooks and fast refresh ([69ea838](https://github.com/danielvm-git/bigbase/commit/69ea8387925828ff49a1e46b34d1cab696235145))
* **ui:** address pre-existing lint issues in react hooks and fast refresh ([f31f769](https://github.com/danielvm-git/bigbase/commit/f31f769b66cf94a1ea582b00029be25df3e2360d))
* **ui:** address pre-existing lint issues in react hooks and fast refresh ([761a33b](https://github.com/danielvm-git/bigbase/commit/761a33ba11796adf9111aabcaf4dd25808d4072c))


### Features

* **sites:** add DELETE /api/sites/:id with safe cascade cleanup ([8667e1e](https://github.com/danielvm-git/bigbase/commit/8667e1ea6a8b57e8969f05a6cfc10f745ffcaeff))
* **sites:** add DELETE /api/sites/:id with safe cascade cleanup ([ae23e3f](https://github.com/danielvm-git/bigbase/commit/ae23e3f29c78f9b4b9f1f5199516cc8b9905f4b0))

## [2.18.1](https://github.com/danielvm-git/bigbase/compare/v2.18.0...v2.18.1) (2026-06-19)


### Bug Fixes

* **lint:** Add missing script to ui/package.json for typecheck, adjust lint scope for baseline ([010b908](https://github.com/danielvm-git/bigbase/commit/010b90814b05f46334bb4f807aa05b190ec48c53))
* **lint:** address pre-existing UI lint issues across multiple components ([38739a2](https://github.com/danielvm-git/bigbase/commit/38739a2eafa554ef0a69aef1da6909b10d32ce01))

# [2.18.0](https://github.com/danielvm-git/bigbase/compare/v2.17.1...v2.18.0) (2026-06-18)


### Bug Fixes

* **ci:** add semantic-release plugins as devDependencies, use local install ([ce183a9](https://github.com/danielvm-git/bigbase/commit/ce183a9be18fff29c2ede2c0e254aeaa7d602db6))


### Features

* **auth:** add phone auth, anonymous token, OAuth popup ([#34](https://github.com/danielvm-git/bigbase/issues/34)) ([2af8713](https://github.com/danielvm-git/bigbase/commit/2af8713fffb089b189341f0ae319a91536e7c12f))
* **sdk:** add multi-framework auth SDK — @bigbase/auth + 5 adapters ([#32](https://github.com/danielvm-git/bigbase/issues/32)) ([554dbe0](https://github.com/danielvm-git/bigbase/commit/554dbe032348bf0a5af3e8b4d5bed85d5c0de699))
* **ui:** add Svelte auth UI components — @bigbase/auth-ui-svelte ([#33](https://github.com/danielvm-git/bigbase/issues/33)) ([5a4d48f](https://github.com/danielvm-git/bigbase/commit/5a4d48f474b894de0ff8bb15b0d4ccdfa7cfcc8f))

## [2.17.1](https://github.com/danielvm-git/bigbase/compare/v2.17.0...v2.17.1) (2026-06-18)


### Bug Fixes

* **ui:** resolve TS build errors — unused import and missing node types ([43253e9](https://github.com/danielvm-git/bigbase/commit/43253e96e30c80d9a87aad9a81c7b6f845489b5d))

# [2.17.0](https://github.com/danielvm-git/bigbase/compare/v2.16.0...v2.17.0) (2026-06-18)


### Features

* **auth:** add passwordless auth — OTP, magic link, user management ([#31](https://github.com/danielvm-git/bigbase/issues/31)) ([03ff564](https://github.com/danielvm-git/bigbase/commit/03ff564c81c2d6bb640b8ed9b2b770e68140bfe1))

# [2.16.0](https://github.com/danielvm-git/bigbase/compare/v2.15.0...v2.16.0) (2026-06-18)


### Features

* **auth:** add POST /api/auth/logout endpoint ([#30](https://github.com/danielvm-git/bigbase/issues/30)) ([3f51ebf](https://github.com/danielvm-git/bigbase/commit/3f51ebfb6c31161b581dca4c46ddf0060457b287))

# [2.15.0](https://github.com/danielvm-git/bigbase/compare/v2.14.0...v2.15.0) (2026-06-18)


### Features

* **auth:** add configurable OAuth redirect and SPA token delivery ([#29](https://github.com/danielvm-git/bigbase/issues/29)) ([5d1ab6e](https://github.com/danielvm-git/bigbase/commit/5d1ab6ec5c36f3de6e98a92ae82810ade0273cd6)), closes [#28](https://github.com/danielvm-git/bigbase/issues/28)

# [2.14.0](https://github.com/danielvm-git/bigbase/compare/v2.13.0...v2.14.0) (2026-06-18)


### Features

* **auth,proxy:** add configurable CORS middleware for browser SPA support ([#28](https://github.com/danielvm-git/bigbase/issues/28)) ([f775bfc](https://github.com/danielvm-git/bigbase/commit/f775bfc19c86122056600ed43405a4fec929eeeb))

# [2.13.0](https://github.com/danielvm-git/bigbase/compare/v2.12.0...v2.13.0) (2026-06-18)


### Features

* **api,ui:** add filter and sort to collections API with Data Studio UI ([#27](https://github.com/danielvm-git/bigbase/issues/27)) ([06868a3](https://github.com/danielvm-git/bigbase/commit/06868a39560801ebc1b582739c4d34e0a2c2d193))
* **functions:** inject env, fetch, db, request context, and schedule loop into jsRuntime ([#25](https://github.com/danielvm-git/bigbase/issues/25)) ([afce751](https://github.com/danielvm-git/bigbase/commit/afce751379e3b94a667cb7d258e94a4bde7c7d30))
* **messaging:** add WebhookProvider and telegram messaging endpoint ([#26](https://github.com/danielvm-git/bigbase/issues/26)) ([0728992](https://github.com/danielvm-git/bigbase/commit/0728992bc65e1ebb1bfe4743249713b6ae2a2633))

# [2.12.0](https://github.com/danielvm-git/bigbase/compare/v2.11.0...v2.12.0) (2026-06-18)


### Features

* **specs:** add e30 Backend for Bots & Integrations epic with e30s01 Functions Runtime plan ([bab659d](https://github.com/danielvm-git/bigbase/commit/bab659dc8e04d803ce129c582022d5c3085f1bac))

# [2.11.0](https://github.com/danielvm-git/bigbase/compare/v2.10.2...v2.11.0) (2026-06-12)


### Features

* **deploy,sites,proxy:** implement site build and request logs ([6704606](https://github.com/danielvm-git/bigbase/commit/670460601bdcbd5a72dfb21f475ab3b1f336a58a))
* **ui:** add tabs to SiteDetailPage (Deployments, Build Logs) ([2b15f00](https://github.com/danielvm-git/bigbase/commit/2b15f008e4bbf1534e565a55d24f4217aedb4c8b))
* **ui:** add useBuildLogs hook for fetching deployment logs ([c9b75a5](https://github.com/danielvm-git/bigbase/commit/c9b75a516e2849adc92a44875ba458177e9a302f))

## [2.10.2](https://github.com/danielvm-git/bigbase/compare/v2.10.1...v2.10.2) (2026-06-12)


### Bug Fixes

* **proxy:** allow inline styles and fonts in CSP & add root README ([04c4534](https://github.com/danielvm-git/bigbase/commit/04c4534840db49c8d64d64b27c46f7ae11427e04))

## [2.10.1](https://github.com/danielvm-git/bigbase/compare/v2.10.0...v2.10.1) (2026-06-12)


### Bug Fixes

* **deploy:** use site name for deployment subdomain ([#24](https://github.com/danielvm-git/bigbase/issues/24)) ([69ad483](https://github.com/danielvm-git/bigbase/commit/69ad4830a7572e436f5cb965824ea57f49270422))

# [2.10.0](https://github.com/danielvm-git/bigbase/compare/v2.9.0...v2.10.0) (2026-06-12)


### Features

* **auth:** security vulnerability fixes and regression tests ([#23](https://github.com/danielvm-git/bigbase/issues/23)) ([122dd0a](https://github.com/danielvm-git/bigbase/commit/122dd0a0b7ad3b3088fc21027553773bfe0cd83a))

# [2.9.0](https://github.com/danielvm-git/bigbase/compare/v2.8.0...v2.9.0) (2026-06-12)


### Features

* **epics:** plan e29 security vuln fixes — OAuth CSRF, cross-tenant isolation, /api/sql gate, users enumeration ([7a4e361](https://github.com/danielvm-git/bigbase/commit/7a4e361030d319ebc936af6f96b19ae8cbd6eb28))

# [2.8.0](https://github.com/danielvm-git/bigbase/compare/v2.7.0...v2.8.0) (2026-06-12)


### Features

* **e28:** Delete Deployment — DELETE /api/deployments/:id, process teardown, UI delete button ([766dceb](https://github.com/danielvm-git/bigbase/commit/766dcebe1c069cd9f5982a2f44cc52e4cbf25e4d))
* **epics:** plan observability epics and update state ([29a3a6f](https://github.com/danielvm-git/bigbase/commit/29a3a6fd10b606247a4efa28afe88a5b5b19e812))

# [2.6.0](https://github.com/danielvm-git/bigbase/compare/v2.5.1...v2.6.0) (2026-06-06)


### Features

* **api:** e23s03 resource isolation by org_id ([1c166ca](https://github.com/danielvm-git/bigbase/commit/1c166ca28e806b03a2650bec72e78e4dac852cfd))
* **auth:** e23s02 team membership and invitations ([2b832f8](https://github.com/danielvm-git/bigbase/commit/2b832f86b6764d2c8b3f7e3ff1ca409c181d7ad5))
* **auth:** e23s04 org-scoped API key management ([1ba1550](https://github.com/danielvm-git/bigbase/commit/1ba155058b50663cb4f418461310d56315ddd36b))
* **e23:** Multi-Tenancy — team membership, isolation, API keys, usage ([60e267e](https://github.com/danielvm-git/bigbase/commit/60e267e669606b72a49d45efc94b609c121f5c86))
* **monitoring:** e23s05 usage tracking per org ([e4c4fe0](https://github.com/danielvm-git/bigbase/commit/e4c4fe0096ce679ba367dc9579aefd7474a02128))

## [2.5.1](https://github.com/danielvm-git/bigbase/compare/v2.5.0...v2.5.1) (2026-06-05)


### Bug Fixes

* **auth:** address review gaps — slug validation, PATCH body check, DRY refactors ([29cf9b9](https://github.com/danielvm-git/bigbase/commit/29cf9b9c10e351c18a80566540e7f0572b08442f))

# [2.5.0](https://github.com/danielvm-git/bigbase/compare/v2.4.2...v2.5.0) (2026-06-05)


### Bug Fixes

* **monitoring:** make Prometheus metrics endpoint publicly accessible ([101d4b5](https://github.com/danielvm-git/bigbase/commit/101d4b59a63216b074c6013b4548aa72ae77cde5))
* **proxy:** skip loopback addresses in deployment host middleware ([88f46fe](https://github.com/danielvm-git/bigbase/commit/88f46fea9023450b3e800a3cff01c6c759065239))


### Features

* **observability:** add --log-level flag to serve command ([de076b7](https://github.com/danielvm-git/bigbase/commit/de076b77d5de30e6fde3cb5fa9df29bf4a3aa857))
* **observability:** add X-Request-ID middleware with context propagation ([16aa0c2](https://github.com/danielvm-git/bigbase/commit/16aa0c25d8818bc321925e53cc7a36273f113441))
* **observability:** complete wire observability epic ([62800c1](https://github.com/danielvm-git/bigbase/commit/62800c1fe20f6eeea728d59a5f838a31794f22c3))

## [2.4.2](https://github.com/danielvm-git/bigbase/compare/v2.4.1...v2.4.2) (2026-06-05)


### Bug Fixes

* **proxy:** skip loopback addresses in deployment host middleware ([#16](https://github.com/danielvm-git/bigbase/issues/16)) ([72688fe](https://github.com/danielvm-git/bigbase/commit/72688feeac156a40beb14426af7e74fc14f75ae8))

## [2.4.1](https://github.com/danielvm-git/bigbase/compare/v2.4.0...v2.4.1) (2026-06-05)


### Bug Fixes

* **deploy:** stop service before rollback binary copy and add health check diagnostics ([#15](https://github.com/danielvm-git/bigbase/issues/15)) ([dbaed0b](https://github.com/danielvm-git/bigbase/commit/dbaed0bd979d1e82c636943ce9a5eb707ff34355))

# [2.4.0](https://github.com/danielvm-git/bigbase/compare/v2.3.0...v2.4.0) (2026-06-05)


### Features

* **specs:** add BigBase Console design prototype bundle ([#14](https://github.com/danielvm-git/bigbase/issues/14)) ([a23e055](https://github.com/danielvm-git/bigbase/commit/a23e0557dfbac224329c12e39bd2439b37f2a8e8))

# [2.3.0](https://github.com/danielvm-git/bigbase/compare/v2.2.0...v2.3.0) (2026-06-05)


### Features

* **deploy:** live build terminal on create-site step 3 ([#13](https://github.com/danielvm-git/bigbase/issues/13)) ([c3794be](https://github.com/danielvm-git/bigbase/commit/c3794be59a2ccb9431dba427d9cd749aa97d464f))

# [2.2.0](https://github.com/danielvm-git/bigbase/compare/v2.1.9...v2.2.0) (2026-06-04)


### Features

* **ui:** close all 19 prototype-vs-codebase gaps ([#12](https://github.com/danielvm-git/bigbase/issues/12)) ([ef3caac](https://github.com/danielvm-git/bigbase/commit/ef3caacacb9f34a66513687ba295a7ceab40bbce)), closes [#4](https://github.com/danielvm-git/bigbase/issues/4) [#6](https://github.com/danielvm-git/bigbase/issues/6)

## [2.1.9](https://github.com/danielvm-git/bigbase/compare/v2.1.8...v2.1.9) (2026-06-04)


### Bug Fixes

* **deploy:** Node 20 VPS setup, npm HOME env, and deploy error UX ([2c7d660](https://github.com/danielvm-git/bigbase/commit/2c7d660114245e9ab0c7375e7f37cd1411fdf9cf))

## [2.1.8](https://github.com/danielvm-git/bigbase/compare/v2.1.7...v2.1.8) (2026-06-04)


### Bug Fixes

* **ui:** align dashboard with prototype and live CPU/memory metrics ([4f55e00](https://github.com/danielvm-git/bigbase/commit/4f55e004c707db49ad243a84a13dad60defc95ea))

## [2.1.7](https://github.com/danielvm-git/bigbase/compare/v2.1.6...v2.1.7) (2026-06-04)


### Bug Fixes

* **deploy:** use https://slug.bigbase.click for production site URLs ([#11](https://github.com/danielvm-git/bigbase/issues/11)) ([582d143](https://github.com/danielvm-git/bigbase/commit/582d143bfc592df61011280941d06e6354326c1e))

## [2.1.6](https://github.com/danielvm-git/bigbase/compare/v2.1.5...v2.1.6) (2026-06-04)


### Bug Fixes

* **ui:** port Sites prototype CSS and block unstyled deploy regressions ([26aba30](https://github.com/danielvm-git/bigbase/commit/26aba30a291f16ae2a8c97dd336ccb99ee1b1a01))

## [2.1.5](https://github.com/danielvm-git/bigbase/compare/v2.1.4...v2.1.5) (2026-06-03)


### Bug Fixes

* **github:** close audit gaps for create-site reconnect UX ([4767cf7](https://github.com/danielvm-git/bigbase/commit/4767cf7da869c8f79685797139184e344cb22057))

## [2.1.4](https://github.com/danielvm-git/bigbase/compare/v2.1.3...v2.1.4) (2026-06-03)


### Bug Fixes

* **github:** surface repos API errors in create-site UI ([d47becf](https://github.com/danielvm-git/bigbase/commit/d47becfe6fde594ee1a085942ee4aadc50d4b2a7))

## [2.1.3](https://github.com/danielvm-git/bigbase/compare/v2.1.2...v2.1.3) (2026-06-03)


### Bug Fixes

* **github:** allow install callback and webhook without JWT auth ([485181f](https://github.com/danielvm-git/bigbase/commit/485181f51bdedb65be6e5f686470bff51fce3d90))

## [2.1.2](https://github.com/danielvm-git/bigbase/compare/v2.1.1...v2.1.2) (2026-06-03)


### Bug Fixes

* **deploy:** allow scp-action to read GitHub App PEM on runner ([c6d647e](https://github.com/danielvm-git/bigbase/commit/c6d647e76128c5cf554c6e539498d6148b5b97ea))

## [2.1.1](https://github.com/danielvm-git/bigbase/compare/v2.1.0...v2.1.1) (2026-06-03)


### Bug Fixes

* **deploy:** wire GitHub App secrets to production via env and VPS PEM ([8b76a9a](https://github.com/danielvm-git/bigbase/commit/8b76a9a727a186924646d13c74ec9113aa7e2be9))

# [2.1.0](https://github.com/danielvm-git/bigbase/compare/v2.0.0...v2.1.0) (2026-06-03)


### Features

* **ui:** complete epic e17 Enhanced Admin UI prototype parity ([#10](https://github.com/danielvm-git/bigbase/issues/10)) ([f610b14](https://github.com/danielvm-git/bigbase/commit/f610b14da299314c0471e960126666d168f34a32))

# [2.0.0](https://github.com/danielvm-git/bigbase/compare/v1.4.0...v2.0.0) (2026-06-02)


### Bug Fixes

* **ui:** address code review findings ([f38a037](https://github.com/danielvm-git/bigbase/commit/f38a037ecf76d4a827bd86a363a658da36a6c5e8))


### Code Refactoring

* **skills:** decouple from ECC — vendor all deps locally ([7a3a4af](https://github.com/danielvm-git/bigbase/commit/7a3a4af3bc35b376cf9ca1b22ec34d20c1356c01))


### Features

* **skills:** upgrade bigpowers skillset to v2.1.0 ([61a8660](https://github.com/danielvm-git/bigbase/commit/61a8660ac7dd2e570410449257e925d04bd3cc81))
* **ui:** add MetricCard, RequestChart, ComponentHealthGrid, QuickActions dashboard primitives with full test coverage (20 tests) ([5afb8b6](https://github.com/danielvm-git/bigbase/commit/5afb8b6107898634fa39b10ac5ecf096afb6d3fc))


### BREAKING CHANGES

* **skills:** opencode.json no longer depends on ECC opensrc cache

# [1.4.0](https://github.com/danielvm-git/bigbase/compare/v1.3.2...v1.4.0) (2026-06-02)


### Bug Fixes

* **ui:** add tests for DashboardMetrics, fix redeploy error handling, address review warnings ([5a069af](https://github.com/danielvm-git/bigbase/commit/5a069af6077cca5e9f85db27b7b1d819bdaef65b))


### Features

* **ui:** add deploy status timeline, dashboard metrics grid, and sidebar hamburger toggle ([d818c0c](https://github.com/danielvm-git/bigbase/commit/d818c0c41a7460b3ec113bb0612d7a165c13b80c))

## [1.3.2](https://github.com/danielvm-git/bigbase/compare/v1.3.1...v1.3.2) (2026-06-02)


### Bug Fixes

* **ui:** resolve TS errors — export ToastVariant, remove unused catch params ([ab90f5d](https://github.com/danielvm-git/bigbase/commit/ab90f5d76436496c78fae4ec8e5a575adf675e52))

## [1.3.1](https://github.com/danielvm-git/bigbase/compare/v1.3.0...v1.3.1) (2026-06-02)


### Bug Fixes

* address code review findings for Epic 017 ([f7669f2](https://github.com/danielvm-git/bigbase/commit/f7669f2b3a7899232bd84aecb5a3c62d0efd0361))

# [1.3.0](https://github.com/danielvm-git/bigbase/compare/v1.2.1...v1.3.0) (2026-06-02)


### Features

* **deploy:** add SSE log stream and enhanced detail pages ([a4abae1](https://github.com/danielvm-git/bigbase/commit/a4abae1120a620a856070f9eb15495bf6748f16c))
* **functions:** add execution logs persistence and viewer page ([914d490](https://github.com/danielvm-git/bigbase/commit/914d490e17450f80a3083776c8ce0665e8035706))
* **realtime:** add status endpoint and Realtime inspector page ([567a41a](https://github.com/danielvm-git/bigbase/commit/567a41a6ad025bbfbbcbe6c70618fc2b034fd090))
* **storage:** add thumbnail endpoint for image previews ([cd2ed09](https://github.com/danielvm-git/bigbase/commit/cd2ed093b9ef85d82b41e87d0fcdd8b6aac0b6ad))
* **ui:** add dark mode toggle, toast notifications, and dashboard enhancements ([0d1138c](https://github.com/danielvm-git/bigbase/commit/0d1138cd35b3057d54c5320b71e237cd516f6c6d))
* **ui:** add grid/list view toggle and image preview to Storage ([484cea6](https://github.com/danielvm-git/bigbase/commit/484cea6992e3067867b043d22d308b770d7be7a7))
* **ui:** extract design tokens to styles/, add Vitest test infrastructure ([bf8b662](https://github.com/danielvm-git/bigbase/commit/bf8b66251084846f39dc5c2b646a46ebdc58ea17))

## [1.2.1](https://github.com/danielvm-git/bigbase/compare/v1.2.0...v1.2.1) (2026-06-02)


### Bug Fixes

* **opencode:** check-stack references release-deploy.yml, not ci.yml ([3f8f492](https://github.com/danielvm-git/bigbase/commit/3f8f492519d138638c7b0953f74e8c9940ab46f8))

# [1.2.0](https://github.com/danielvm-git/bigbase/compare/v1.1.1...v1.2.0) (2026-06-02)


### Features

* **opencode:** wire /check-stack command and agentic stack ([6306a3d](https://github.com/danielvm-git/bigbase/commit/6306a3da6b38af3e6f6b36a91d7b89fe362ab67e))

## [1.1.1](https://github.com/danielvm-git/bigbase/compare/v1.1.0...v1.1.1) (2026-06-02)


### Bug Fixes

* **main:** register GitHub App CLI flags and wire to github component ([0c7fea6](https://github.com/danielvm-git/bigbase/commit/0c7fea6fdf0a635f2a8747d05729b5a65d231e8a))

# [1.1.0](https://github.com/danielvm-git/bigbase/compare/v1.0.2...v1.1.0) (2026-06-01)


### Features

* **config:** configure bigbase.click domain with HTTPS ([7226b98](https://github.com/danielvm-git/bigbase/commit/7226b9832799238b75dd3d4dbc2c1beab84d290d))

## [1.0.2](https://github.com/danielvm-git/bigbase/compare/v1.0.1...v1.0.2) (2026-06-01)


### Bug Fixes

* **main:** wire github and sites components into backend, proxy, and UI ([#9](https://github.com/danielvm-git/bigbase/issues/9)) ([540a2da](https://github.com/danielvm-git/bigbase/commit/540a2dab0437235ccd4f7a390a8a1e3fb40965dd))

## [1.0.1](https://github.com/danielvm-git/bigbase/compare/v1.0.0...v1.0.1) (2026-06-01)


### Bug Fixes

* **ci:** merge release and deploy into single workflow to fix GITHUB_TOKEN event suppression ([f05faf3](https://github.com/danielvm-git/bigbase/commit/f05faf3c6d5b2eb292dd9af9280f06ac263e1983))

# 1.0.0 (2026-06-01)


### Bug Fixes

* address review findings — rows.Err, routing, YAML leak, repo filter ([45e79c3](https://github.com/danielvm-git/bigbase/commit/45e79c39440ae6327525ca662b53e02f4d57d6ee))
* address review findings — security, authz, race conditions ([2a49228](https://github.com/danielvm-git/bigbase/commit/2a49228e1fc32f95d22e29aebf03cc3719490e4f))
* **api db:** address all 18 code review findings ([9c693a4](https://github.com/danielvm-git/bigbase/commit/9c693a433d006a26e1ca48164444a5f94576afa7))
* **ci:** run smoke test binary with ./ prefix ([83b2de1](https://github.com/danielvm-git/bigbase/commit/83b2de10b643f22d25ff458d763b432db1a045ad))
* **deploy:** CI git safe.directory and Caddy http:// catch-all ([d0af465](https://github.com/danielvm-git/bigbase/commit/d0af4650fdb84b21ad26d571b581132371e9c0c3))
* **deploy:** enable bigbase systemd unit on VPS setup ([2aa3361](https://github.com/danielvm-git/bigbase/commit/2aa3361db77c02c25ec310359194f0730edcc241))
* **deploy:** resolve relative paths and URL update timing ([9ca144b](https://github.com/danielvm-git/bigbase/commit/9ca144b2b585e30fd6f8a974c90db627a21d25f3))
* **deploy:** serve static files via HTTP file server ([575f780](https://github.com/danielvm-git/bigbase/commit/575f780ed0e063a5f8072e2c942cc4e6c111a925))
* **functions:** address review findings — interrupt timeout, validate update, runtime interface, body limit, injectable runtimes ([a45fa7a](https://github.com/danielvm-git/bigbase/commit/a45fa7a90f7100d96473bdb978e3cb7a4f220cdd))
* harden kernel, api, auth, forge, git, and storage components ([ef4ea80](https://github.com/danielvm-git/bigbase/commit/ef4ea80440bd62ca63199138d6d0fbba2f686408))
* **realtime:** fix Hub data race, add event unsubscribe, log emit errors ([f10ed83](https://github.com/danielvm-git/bigbase/commit/f10ed8398be27a6fc5f7a6de5107f891789723db))
* **ui:** export missing components from index.ts ([994bc53](https://github.com/danielvm-git/bigbase/commit/994bc531c75d4026f2ca72c9425ae8b77362d95d))


### Features

* **admin-ui:** port Appwrite design tokens and create shared component system ([da3f118](https://github.com/danielvm-git/bigbase/commit/da3f118678f0b80b5874e0d7dc559b5700cd3447))
* **admin:** add Admin UI, SQL Editor, and User Management ([e19e977](https://github.com/danielvm-git/bigbase/commit/e19e9775f62605ca4174a457e0d62ba021e47d32))
* **auth:** add email/password auth with JWT tokens ([#1](https://github.com/danielvm-git/bigbase/issues/1)) ([35c6096](https://github.com/danielvm-git/bigbase/commit/35c60969656a3a51a0ecff3a720f97f0a174f177))
* **auth:** add Google OAuth social login via embedded relay ([d4970ec](https://github.com/danielvm-git/bigbase/commit/d4970ec7b7d89e32652c9811f67b93ee668f7a29))
* **cici:** add CI/CD pipeline component (Slice 9) ([5d2ba35](https://github.com/danielvm-git/bigbase/commit/5d2ba355b884661e9938e051d00adcf05303b5ac))
* **cli proxy kernel:** add CLI commands, proxy server, and observability ([fc23a09](https://github.com/danielvm-git/bigbase/commit/fc23a09eb75b67436a08200e876eef687ad8d237))
* **db api:** add SQLite database and auto REST CRUD API ([696befb](https://github.com/danielvm-git/bigbase/commit/696befbe2b772042cfe1c23193e6a45ebea17876))
* **deploy:** add deploy component with build detection, process execution, and port allocation ([c8d001a](https://github.com/danielvm-git/bigbase/commit/c8d001aa34aac7d4d0665db256de235310376761))
* **deploy:** improve deployment pipeline and add admin UI pages ([d056ce1](https://github.com/danielvm-git/bigbase/commit/d056ce1757c3460861913113fa18f01cedd30ad1))
* **forge:** add issues, labels, comments, kanban board, and wiki ([e893817](https://github.com/danielvm-git/bigbase/commit/e893817528dd57a27fd918387a417b68e0346a76))
* **functions:** add Functions component with CRUD operations ([00ea81c](https://github.com/danielvm-git/bigbase/commit/00ea81c45485f2d1971227d1fc39c9c67fcd7260))
* **functions:** add goja JS runtime with console.log capture and timeout ([44a74f5](https://github.com/danielvm-git/bigbase/commit/44a74f5cba9002559f0aed918a4c895f4abd6e5f))
* **functions:** wire Functions component into main.go ([9b89585](https://github.com/danielvm-git/bigbase/commit/9b89585a569ac42e9dd505847675c908ee4b3649))
* **git:** add repository management with bare repo creation ([0481cfc](https://github.com/danielvm-git/bigbase/commit/0481cfc785e6fbfd0ec0ad58c7ff416a781bde9d))
* integrate semantic-release and display version in admin footer ([3e6b86c](https://github.com/danielvm-git/bigbase/commit/3e6b86cdf5df726fc506ab568a05ed9819f71d56))
* integrate semantic-release and display version in admin footer ([f042fe2](https://github.com/danielvm-git/bigbase/commit/f042fe2361ed476a9ac03f784f87c44374ac545a))
* **messaging:** add email, SMS, and push notification endpoints with message log ([fd7cb61](https://github.com/danielvm-git/bigbase/commit/fd7cb61b82c42f473e19cac76565ef8dd435debf))
* **messaging:** wire messaging component into main.go with auth middleware ([5e48755](https://github.com/danielvm-git/bigbase/commit/5e487553f624c8093d267c1b4a908ee942f141f7))
* **monitoring:** add /api/monitoring/metrics endpoint with system and request metrics ([dcd0430](https://github.com/danielvm-git/bigbase/commit/dcd04303045bef4131262e517c704882fdde2838))
* **monitoring:** add alert CRUD endpoints ([1b356e6](https://github.com/danielvm-git/bigbase/commit/1b356e63f8644510a74b813227a86aacdb09dd99))
* **monitoring:** add component shell with health endpoint ([b292c81](https://github.com/danielvm-git/bigbase/commit/b292c81635e7fee26049253a5402af3329fb0d16))
* **monitoring:** add log storage and search endpoints ([6d3a6b7](https://github.com/danielvm-git/bigbase/commit/6d3a6b75331dd46e4603da2c31bcfc04d87a47e2))
* **monitoring:** add request metrics middleware ([4dfbbc1](https://github.com/danielvm-git/bigbase/commit/4dfbbc1318a7f5a8cb4ee8ecd12ac48cd49bcb43))
* **monitoring:** add system metrics collector ([fc6bd9e](https://github.com/danielvm-git/bigbase/commit/fc6bd9e883f4b8b1a39c62d43ecd53a09bec7f92))
* **monitoring:** wire into main.go and add admin UI page ([0a683cd](https://github.com/danielvm-git/bigbase/commit/0a683cd2663b412eb0554ccb875265aa4a2fc43f))
* **proxy:** add commercial landing page and documentation ([c3197e6](https://github.com/danielvm-git/bigbase/commit/c3197e6066744ef52f18cec62f3f4e03273eaf9f))
* **proxy:** add commercial landing page with GitHub stars cache ([fea3761](https://github.com/danielvm-git/bigbase/commit/fea37618f2e70ddccb5d983b487c53c0a013c4cd))
* **proxy:** add documentation page with sidebar navigation ([dcaff68](https://github.com/danielvm-git/bigbase/commit/dcaff68f7fac2a19583c14b619b6c3b614974cc8))
* **realtime:** add WebSocket subscriptions, event bus broadcasts, and Hub pattern ([4ab37bf](https://github.com/danielvm-git/bigbase/commit/4ab37bf4a3a7ab5a895d9f8897bf5bbb597bf560))
* **storage:** add file upload, download, list, and delete ([78ebe92](https://github.com/danielvm-git/bigbase/commit/78ebe9205aa9eb4feb290ab8ca6810e6e46ffff8))
* **ui:** enhance dashboard with charts and fix Google login visibility ([a453256](https://github.com/danielvm-git/bigbase/commit/a453256a715c56eb2df2c8584045d3b48fa901ee))


### BREAKING CHANGES

* Deploy now triggers on release.published instead of every push to main.
