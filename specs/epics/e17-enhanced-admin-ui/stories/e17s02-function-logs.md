---
id: e17s02
title: Function logs viewer
status: done
legacy_slice: "017-B"
tasks:
  - desc: Function logs viewer and GET /api/functions/:id/logs
    verify: "go test ./components/functions/ -run TestLogs -v"
---
