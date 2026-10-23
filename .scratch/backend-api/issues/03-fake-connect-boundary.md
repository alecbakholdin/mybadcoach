# Fake the Connect boundary in frontend-only tests

Type: research
Status: resolved

## Question

`docs/testing.md` requires frontend-only integration tests to use a single network-boundary fake
and no mocks. With Connect (connect-web rides `fetch`), determine the minimal, standards-compliant
way to fake the backend boundary in frontend-only Playwright tests:

- MSW-style HTTP interception of Connect requests
- connect-es transport/interceptor test doubles
- a stub Connect server

Report the chosen technique and why, plus what the fake must implement (unary + streaming calls,
protobuf encoding). Findings feed the testing tickets of the implementation effort.

## Answer

**Chosen: a stub Connect server** — a Node process using `@connectrpc/connect-node`'s
`connectNodeAdapter` over the generated service definitions, backed by an in-memory fake store,
started by Playwright's `webServer` array. The client points at it through the same public
base-URL env var used in production, so no application code changes and no module mocks.

Why this wins:

- **It is a real network boundary.** The app's real transport, `fetch`, serialization, stores,
  rendering, and navigation run unchanged; the only double is a stateful fake service
  implementation ("fakes over mocks", `docs/testing.md`).
- **It is the only option that covers unary *and* server-streaming with binary Protobuf without
  hand-writing Connect framing.** The Connect router handles both codecs, 5-byte stream
  envelopes, end-stream messages, and `ConnectError` encoding; the fake author writes message
  handlers only.
- **It preserves the semantics browser-level fakes lose:** real CORS preflight (a service worker
  cannot intercept preflight — Fetch standard §4.8), real `Set-Cookie`/`HttpOnly` behavior, real
  HTTP status/error handling, and SvelteKit SSR-side calls.
- **It is explicitly allowed by `docs/testing.md`** ("a fake API at the network boundary (Mock
  Service Worker or a stub HTTP server)").

Costs: one dev-only dependency (`@connectrpc/connect-node`, subject to ticket 01 vetting), a
second process managed by Playwright, an in-memory reset between tests, and real CORS/credentials
configuration — which mirrors production and forces ticket 04's cookie/transport posture to be
settled rather than hidden by the fake.

Rejected alternatives:

- **connect-es transport/interceptor doubles** (`createRouterTransport`, custom `fetch`,
  interceptors): in-process, and using them requires a test seam inside the app (transport
  injection or a conditional import). They bypass real `fetch`, HTTP, CORS, cookies, and SSR, so
  they are a mock of the transport, not a network-boundary fake. Keep them for component/unit
  tests only.
- **MSW:** workable only same-origin, unary-or-buffered, and without HttpOnly-cookie assertions.
  Service-worker mode cannot see CORS preflight at all, and its `HttpResponse` cookie workaround
  writes through `document.cookie`, which cannot reproduce `HttpOnly`. Handlers must hand-encode
  Protobuf and Connect stream framing. `@msw/playwright` avoids in-page startup but fully buffers
  responses via `route.fulfill` and is 0.x.
- **Notable fourth option:** `@connectrpc/connect-playwright` is Connect's official Playwright
  recommendation and is typed/binary-capable with no app changes, but it is unary-only (it throws
  on non-unary methods), 0.x beta, and adds no CORS headers. It is the runner-up if the seam is
  ever restricted to unary-only and an extra process is unacceptable.

**The fake must implement:** every in-scope RPC with generated request/response types; Connect
error codes/details; server-streaming methods as async iterables; a stateful in-memory store plus
a reset mechanism between tests; faithful auth cookie behavior per ticket 04; and CORS/credentials
matching production if the API origin differs from the app origin.

Full findings with primary-source citations:
<https://github.com/alecbakholdin/mybadcoach/blob/research/connect-test-boundary/docs/research/connect-test-boundary.md>
(throwaway branch `research/connect-test-boundary`, never merged).
