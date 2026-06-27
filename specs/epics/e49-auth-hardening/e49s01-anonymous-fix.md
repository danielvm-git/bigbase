# e49s01: Fix anonymous tokens — add org_id context
## Story ID: e49s01 | Epic: e49 | BCPs: 2 | Status: planned

## Summary
Anonymous tokens are currently broken — they always return 403 because the JWT lacks `org_id` and the auth middleware rejects tokens with `OrgID == 0`. Fix by using the `Claims` struct with `role=anonymous` and adding a special case in the middleware to bypass org isolation for anonymous tokens. Downstream handlers enforce read-only access for `role=anonymous`.

## Acceptance Criteria (Gherkin)
```gherkin
Scenario: Anonymous token is accepted by middleware
  Given an anonymous token is minted via POST /api/auth/anonymous
  When the token is used to access GET /api/collections/items
  Then the response is 200 OK (not 403 "no organization")

Scenario: Anonymous user cannot create records
  Given an anonymous token
  When POST /api/collections/items is called
  Then the response is 403 "insufficient permissions"

Scenario: Anonymous token does not include refresh token
  Given an anonymous token is minted
  Then the response body has no refresh_token field
```
