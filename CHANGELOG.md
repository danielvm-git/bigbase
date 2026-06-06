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
