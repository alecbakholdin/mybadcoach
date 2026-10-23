# Faking the Connect boundary in frontend-only Playwright tests

**Date:** 2026-10-22
**Ticket:** `.scratch/backend-api/issues/03-fake-connect-boundary.md`
**Branch:** `research/connect-test-boundary` (throwaway; never merged)

## Research question

`docs/testing.md` requires the frontend-only Playwright seam to use exactly one test double,
at the network boundary, with no mocks. The client uses `@connectrpc/connect-web` over the
Connect protocol with binary Protobuf. What is the minimal, mock-minimized way to fake the
backend for that seam:

1. MSW-style fetch interception of Connect requests,
2. connect-es transport/interceptor test doubles, or
3. a stub Connect server?

The answer must say what the fake has to implement: unary and server-streaming calls, plus
Protobuf encoding.

## What the boundary actually is

`createConnectTransport` from `@connectrpc/connect-web` is documented as making "unary and
server-streaming methods available to web browsers" and "uses the fetch API to make HTTP
requests" ([connect-web source, `connect-transport.ts`](https://github.com/connectrpc/connect-es/blob/main/packages/connect-web/src/connect-transport.ts)).
Client-streaming and bidirectional streaming are rejected in the browser because the Fetch API
does not support streaming request bodies ([same source, `methodKind != "server_streaming"` check](https://github.com/connectrpc/connect-es/blob/main/packages/connect-web/src/connect-transport.ts)).
So the fake only ever needs **unary + server-streaming**.

Two transport facts matter for every option:

- `useBinaryFormat` defaults to **false**: `const useBinaryFormat = options.useBinaryFormat ?? false;`
  and the JSDoc says "By default, connect-web clients use the JSON format"
  ([source](https://github.com/connectrpc/connect-es/blob/main/packages/connect-web/src/connect-transport.ts)).
  The map's settled design is binary Protobuf, so production will likely set `useBinaryFormat: true`.
- `fetch?: typeof globalThis.fetch` exists specifically so callers can set options "such as
  credentials" ([source](https://github.com/connectrpc/connect-es/blob/main/packages/connect-web/src/connect-transport.ts)).
  Without a wrapper, fetch's default `credentials: "same-origin"` applies, so cross-origin cookie
  auth needs an explicit wrapper. That is an auth/transport decision for tickets 04/05, but the
  fake must exercise whatever posture is chosen.

### Connect wire format the fake must produce

From the [Connect protocol specification](https://connectrpc.com/docs/protocol):

- **Unary:** `POST` with `Content-Type: application/proto` or `application/json`; one bare
  serialized message, no framing. Generated clients "should send" `Connect-Protocol-Version: 1`.
  Success is HTTP 200; errors use a non-200 status and an `application/json` body
  `{"code": ..., "message": ..., "details": [...]}`. There is no `connect-error` header.
- **Server-streaming:** `Content-Type: application/connect+proto` or
  `application/connect+json`. Every message is a 5-byte envelope: one flags byte then a 4-byte
  big-endian message length. The end-stream flag is bit 2 (`2`). Streaming responses always use
  HTTP 200; success or failure is carried in the final end-stream envelope, and an error is an
  end-stream message with an `"error"` object.

A fake that pretends to be a Connect server therefore has to decode the request codec, encode the
response codec, and (for streaming) frame and unframe envelopes.

### SvelteKit SSR caveat

Browser-level interception only sees browser requests. SvelteKit universal `+page.ts` load
functions run on the server for the first request (SSR) and again during hydration; they "will
then run again during hydration" and subsequent invocations happen in the browser
([SvelteKit load docs](https://svelte.dev/docs/kit/load)). With `export const ssr = false`,
universal load "will only run in the browser"
([SvelteKit page options](https://svelte.dev/docs/kit/page-options)). If Connect calls are made
from a universal load during SSR, MSW/`page.route` fakes miss them and the test hits a real
network. A real stub HTTP server does not have this gap.

## Option 1: MSW-style fetch interception

### How MSW works

MSW's browser mode registers a **service worker** that intercepts requests "on the network
level" ([MSW browser integration](https://mswjs.io/docs/integrations/browser)). Handlers are
written against `http.post(...)` and return `HttpResponse` objects. In Playwright there are two
ways to use it:

1. **In-page service worker** (`setupWorker`). The worker must be served at
   `/mockServiceWorker.js`, `worker.start()` must be awaited before the app makes requests
   ([MSW browser integration](https://mswjs.io/docs/integrations/browser)). MSW's own guidance
   for Playwright is to initialize MSW inside the app and expose it on `window.msw` for
   `page.evaluate` overrides, because the worker lives in the browser process and the test runs
   in Node ([mswjs/msw discussion #2404](https://github.com/mswjs/msw/discussions/2404)).
   That requires test-only wiring inside the app, which conflicts with keeping the app
   production-identical.
2. **`@msw/playwright`** (official mswjs binding). It does not use the service worker at all: it
   "will rely on the `page.route()` API", so "you don't have to initialize the worker script",
   and "any `page.route()` limitations now affect this library"
   ([README](https://github.com/mswjs/playwright/blob/main/README.md)). It runs MSW handlers in
   the Node test process and calls `route.fulfill`.

### Intercepting Connect with MSW

Connect requests are ordinary `fetch` POSTs to `/<package>.<Service>/<Method>`, so an MSW handler
can match the path. The cost is protocol work:

- **Binary unary:** handler must read `request.arrayBuffer()`, decode with the generated
  Protobuf-ES schema, and return `new HttpResponse(ResponseMessage.toBinary(msg), { headers: {
  "content-type": "application/proto" } })`.
- **Server-streaming:** handler must build the 5-byte envelopes and an end-stream envelope
  itself. MSW supports `ReadableStream` bodies (`new HttpResponse(stream, ...)`)
  ([MSW streaming docs](https://mswjs.io/docs/http/mocking-responses/streaming)), so the
  service-worker mode can deliver messages incrementally. `@msw/playwright` **cannot**: it fully
  buffers every response with `Buffer.from(await response.arrayBuffer())` before
  `route.fulfill` ([fixture source](https://github.com/mswjs/playwright/blob/main/src/fixture.ts)),
  and Playwright's `route.fulfill` body is `string | Buffer`
  ([Playwright Route docs](https://playwright.dev/docs/api/class-route)). A complete,
  pre-framed body still parses correctly, but tests cannot observe incremental delivery.
- **Errors:** handler must reproduce Connect's status + JSON error shape for unary, and the
  end-stream error envelope for streaming.

### The blockers

**CORS preflight.** Cross-origin Connect calls always preflight: `application/json` and
`application/proto` are not CORS-safelisted content types, and `Connect-Protocol-Version` is a
custom header. A service worker cannot intercept that preflight. The Fetch spec's
CORS-preflight algorithm says explicitly:

> "The service-workers mode of preflight does not matter as this algorithm uses
> HTTP-network-or-cache fetch rather than HTTP fetch."
> — [Fetch standard §4.8](https://fetch.spec.whatwg.org/#cors-preflight-fetch)

So in service-worker mode, MSW cannot fake a cross-origin Connect API in a real browser. It only
works if the app points at a same-origin base URL in tests (which then stops exercising the real
cross-origin/credentials posture). `@msw/playwright` uses `page.route` instead; Playwright's
Chromium auto-answers preflights for routed URLs, but the fulfilled actual response still needs
`Access-Control-Allow-Origin` or the request hangs
([microsoft/playwright#12929](https://github.com/microsoft/playwright/issues/12929)); MSW handlers
would have to add CORS headers by hand.

**Cookies.** MSW's `HttpResponse` can mock `set-cookie`; the docs say the header "will set the
cookies on the `document` as if they were received from the server"
([MSW cookies docs](https://mswjs.io/docs/http/mocking-responses/cookies)). Writing through
`document.cookie` cannot reproduce `HttpOnly`: JavaScript cannot read or set HttpOnly cookies
([MDN `document.cookie`](https://developer.mozilla.org/en-US/docs/Web/API/Document/cookie)), so
the auth flow's real cookie semantics cannot be tested through the worker. The documented
Playwright escape hatch is `browserContext.addCookies()` for custom cookies
([Playwright Route docs](https://playwright.dev/docs/api/class-route)), i.e. seeding state rather
than letting the app's login call establish it.

**SSR.** Browser-only interception misses SvelteKit server-side load calls (above).

### Verdict

MSW is workable only for same-origin, unary-or-buffered-streaming, non-HttpOnly-cookie tests, and
it pushes Protobuf/framing logic into every fake handler. It is the most "mock-like" of the three
in practice because the handlers re-implement the wire protocol by hand.

## Option 2: connect-es transport/interceptor test doubles

### What exists

- The **`Transport` interface** (`unary` + `stream`) can be implemented by hand and passed to
  `createClient(service, transport)`.
- `createConnectTransport` accepts a custom `fetch`, and interceptors wrap calls. Both are
  in-process seams, not network doubles.
- **`createRouterTransport`** is Connect's official test helper. It "Creates a Transport that
  routes requests to the configured router" and is "Useful for testing and calling services
  running in the same process"
  ([source](https://github.com/connectrpc/connect-es/blob/main/packages/connect/src/router-transport.ts)).
  It uses `createUniversalHandlerClient(router.handlers)` with `baseUrl: "https://in-memory"`; it
  never calls `fetch`. The official testing guide recommends it for component tests and says it
  supports "mock streaming responses" and `ConnectError` with details
  ([Connect testing guide](https://connectrpc.com/docs/web/testing)).

```ts
import { createRouterTransport } from "@connectrpc/connect";

const transport = createRouterTransport(({ service }) => {
  service(TeamService, {
    createTeam: (req) => ({ /* fake response */ }),
  });
});
```

### Why it is wrong for this seam

Using it requires the app to accept an injected transport (Svelte context, prop, or a
test-conditional import). That is a seam *inside the application*, exactly what `docs/testing.md`
forbids ("The only test double is a fake API at the network boundary"). It also bypasses the
real `fetch`, serializer, HTTP status handling, CORS, cookies, and the SvelteKit server/browser
split. It is an excellent **component/unit** double, but for the frontend-only integration seam
it is a mock of the transport boundary, not of the network.

### Notable fourth option surfaced by the docs: `@connectrpc/connect-playwright`

The Connect testing guide explicitly recommends this package for Playwright: it "can intercept
requests and return mocked responses to the web application under test" and brings "the
type-safety of your schema to Playwright's API Mocks"
([Connect testing guide](https://connectrpc.com/docs/web/testing)). It is a **network-boundary**
fake via Playwright's `context.route` — not a transport swap. Its API:

```ts
const mock = createMockRouter(context, { baseUrl: "http://localhost:5173" });

await mock.service(TeamService, {
  createTeam: () => ({ /* fake response */ }),
  deleteTeam: "mock", // default-valued response
});
await mock.rpc(TeamService.method.getTeam, () => ({ /* ... */ }));
```

How it works, from the [source](https://github.com/connectrpc/connect-playwright-es/blob/main/packages/connect-playwright/src/create-mock-router.ts):

- It matches `baseUrl + /<package>.<Service>/<Method>` with `context.route`.
- It builds a real Connect router (`createConnectRouter`) and delegates encoding/decoding to its
  universal handler. Binary and JSON requests both work, and `ConnectError` becomes a real
  Connect error response; the fake does not hand-encode Protobuf.
- It fully buffers the response (`readAllBytes(response.body, 0xffffffff, null)`) and the source
  comment says "Assumes non-stream for now."
- It rejects non-unary methods at registration with `"Cannot add non-unary method."`; the README
  says "Currently, only unary requests are able to be mocked" and "We hope to add streaming
  support soon" ([README](https://github.com/connectrpc/connect-playwright-es/blob/main/packages/connect-playwright/README.md)).
- Version 0.6.1 (published 2026-04-22 per `npm view`), peer-depends on `@connectrpc/connect ^2.0.0`.
  The project calls itself beta.

**Verdict:** the least-code network-boundary option for unary-only APIs, and the one Connect's own
docs recommend. But it cannot satisfy the ticket's "unary + streaming" requirement, it is 0.x
beta, it assumes the mock origin matches the app's `baseUrl` (cross-origin would need CORS
headers it does not add), and it is not the "Mock Service Worker or stub HTTP server" named in
`docs/testing.md`.

## Option 3: a stub Connect server

### How it works

Run a real HTTP server in Node that hosts the generated Connect service definitions with fake
implementations, and point the unmodified SvelteKit app at it through the same env var used in
production. `@connectrpc/connect-node` exposes `connectNodeAdapter`, which "Create[s] a Node.js
request handler from a ConnectRouter" ([source](https://github.com/connectrpc/connect-es/blob/main/packages/connect-node/src/connect-node-adapter.ts)),
and the README shows it on Node's `http`/`https`/`http2` modules with the note "you can make
requests with any gRPC, gRPC-Web, or Connect client"
([README](https://github.com/connectrpc/connect-es/blob/main/packages/connect-node/README.md)).
A Fastify plugin (`@connectrpc/connect-fastify`) is the newer documented variant
([Node getting started](https://connectrpc.com/docs/node/getting-started)); vanilla `http` keeps
the dependency surface smaller.

```ts
// fake-api/routes.ts
import type { ConnectRouter } from "@connectrpc/connect";

export default function routes(router: ConnectRouter) {
  router.service(TeamService, {
    createTeam(req) { /* mutate in-memory store, return response */ },
    getTeam(req) { /* read store, throw ConnectError if missing */ },
  });
}
```

```ts
// fake-api/server.ts
import { createServer } from "node:http";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import routes from "./routes";

const handler = connectNodeAdapter({ routes });
createServer((req, res) => {
  if (req.url === "/healthz") { res.writeHead(200).end("ok"); return; }
  handler(req, res);
}).listen(process.env.FAKE_API_PORT ?? 5174);
```

Playwright starts it alongside SvelteKit with the array form of `webServer`; each entry has
`command`, `url` (readiness; 2xx/3xx/400–403), `reuseExistingServer`, and `env`
([Playwright web server docs](https://playwright.dev/docs/test-webserver)):

```ts
webServer: [
  {
    command: "npm run dev -w apps/client",
    url: "http://localhost:5173",
    env: { PUBLIC_API_BASE_URL: "http://localhost:5174" },
  },
  {
    command: "node test/fake-api/server.mjs",
    url: "http://localhost:5174/healthz",
  },
],
```

### What it gives you for free

- The real Connect wire protocol in both directions: JSON and binary unary, server-streaming with
  5-byte envelopes and end-stream messages, `ConnectError` status/body encoding, and compression.
  The fake implementation returns message objects or async iterables; the router does the rest.
- Real HTTP semantics: status codes, headers, `Set-Cookie` (including `HttpOnly`), and **real
  CORS preflight handling**, because it is an actual server. Cross-origin `credentials: "include"`
  plus CORS can be modelled exactly as production will need it.
- Real serialization and transport behavior in the client; the app's real `fetch`, client,
  stores, rendering, and navigation all run unchanged. No test-only branch in production code.
- A stateful fake API with recorded state, which is exactly the "fakes over mocks" shape in
  `docs/testing.md`.
- SSR safety: server-side load calls hit the stub over HTTP too.

### Costs

- A second process in the frontend-only command (Playwright manages it).
- A dev-only dependency: `@connectrpc/connect-node` (plus the already-planned
  `@connectrpc/connect`/`@bufbuild/protobuf`) — dependency vetting per ticket 01.
- The fake must implement CORS/cookie policy if the app is cross-origin. This is real work, but
  it forces the production transport/auth posture (ticket 04) to be settled rather than hidden by
  a fake.
- Test isolation: the server outlives individual tests, so it needs an in-memory reset (a
  test-only RPC or a fresh server per worker).
- Fake service implementations can drift from the Go server; shared generated Protobuf types
  (ticket 07) make drift a type error rather than silent.

## Comparison

| Criterion | MSW service worker | `@msw/playwright` | `createRouterTransport` / custom transport | `@connectrpc/connect-playwright` | Stub Connect server |
| --- | --- | --- | --- | --- | --- |
| Boundary faked | Browser network (SW) | Browser network (`page.route`) | In-process transport | Browser network (`context.route`) | Real HTTP network |
| App code changes | None (but in-page startup needs test wiring) | None | **Yes** (transport injection) | None | None |
| Unary | Yes (hand codec) | Yes (hand codec) | Yes | Yes (typed) | Yes (typed) |
| Server-streaming | Yes via `ReadableStream` (hand-framed) | Buffered only (hand-framed) | Yes | **No** (throws on registration) | Yes (typed async iterable) |
| Binary Protobuf | Manual encode/decode | Manual encode/decode | Automatic | Automatic | Automatic |
| Connect errors | Manual | Manual | Automatic | Automatic | Automatic |
| Real CORS preflight | **No** (SW cannot see it) | Partial (headers by hand) | No | Partial (headers absent) | Yes |
| Cookies incl. HttpOnly | No (document.cookie) | Header-dependent | No | Generic headers only | Yes |
| SSR calls | No | No | No | No | Yes |
| State across reloads | Resets with page | Yes (Node process) | Per client instance | Yes (Node process) | Yes (server process) |
| Extra deps | `msw` | `msw` + `@msw/playwright` (0.x) | none | `@connectrpc/connect-playwright` (0.x) | `@connectrpc/connect-node` |
| Maturity | MSW 3.x | 0.7.0 | Connect-ES 2.x, official | 0.6.1, beta | Connect-ES 2.x, official |
| Matches `docs/testing.md` wording | Yes ("Mock Service Worker") | Yes (MSW binding) | No (internal seam) | Not named (but network boundary) | Yes ("stub HTTP server") |

## Recommendation

**Use a stub Connect server**, started by Playwright's `webServer` array, implemented with
`@connectrpc/connect-node`'s `connectNodeAdapter` (or the Fastify plugin) and generated service
definitions backed by an in-memory fake store. Point the app at it with the same public env var
the production build uses.

Why this is the minimal *mock-minimized* choice:

1. **It is a real network boundary, not a mock.** The app runs production code end to end; only
   the server implementation is fake. No transport injection, no conditional imports, no
   test-only branches.
2. **It is the only option that meets the stated requirement** of unary **and** server-streaming
   with **binary Protobuf** without hand-writing Connect framing. The Connect router serializes,
   frames, handles errors, and supports streaming; the fake author writes only message handlers.
3. **It preserves the semantics the other options lose**: CORS preflight (which a service worker
   cannot intercept at all), real `Set-Cookie`/`HttpOnly` behavior, real HTTP errors, and
   SvelteKit SSR-side calls.
4. **It is explicitly allowed by `docs/testing.md`** ("a fake API at the network boundary (Mock
   Service Worker or a stub HTTP server)") and is the strongest form of "fakes over mocks" — a
   small real implementation with recorded state.
5. **It scales to the future seam.** When live sync lands (bidirectional streaming is currently
   out of scope and unsupported by connect-web anyway), a real server can model it; the
   browser-level fakes cannot.

Runner-up: **`@connectrpc/connect-playwright`** if the team wants zero extra processes and is
willing to restrict the seam to unary RPCs. It is Connect's official Playwright recommendation,
typed, binary-capable, and leaves app code untouched, but it is 0.x beta, cannot mock streaming,
and does not add CORS headers for cross-origin use.

Do **not** use `createRouterTransport`, a custom `fetch`, or an interceptor for this seam: all
require a production-app seam and bypass the network semantics the seam exists to test. Keep them
for component/unit tests if such a harness is ever justified.

MSW is acceptable only under narrow conditions (same-origin base URL, unary or fully buffered
streaming, no HttpOnly-cookie assertions). It requires hand-written Protobuf codecs and Connect
framing, cannot see CORS preflight in service-worker mode, and in `@msw/playwright` mode buffers
streams. It is more fake-specific code than the stub server, not less.

## What the fake must implement

For the chosen stub server, the fake API surface is:

- Every RPC in scope, with typed request/response messages from the generated Protobuf-ES code.
- Connect error semantics: `ConnectError` with the agreed stable machine codes and details;
  unary errors as non-200 + JSON body; streaming errors as an end-stream message.
- Server-streaming methods as async iterables of response messages (when such RPCs land).
- Stateful behavior across calls within a test (create → list → get), plus a reset mechanism
  between tests.
- Auth/session behavior faithful to ticket 04: login sets the real cookie; logout clears it;
  unauthorized calls return `unauthenticated`. This is where a stub server is strictly better
  than browser-level fakes.
- CORS and credentials configured to match production (exact origin, allow
  `content-type`, `connect-protocol-version`, `connect-timeout-ms`,
  `connect-content-encoding`, `connect-accept-encoding`, allow credentials) if the API origin
  differs from the app origin.

## Open questions / caveats

- **Origin posture (ticket 04):** Connect-web's default fetch credentials are same-origin; if the
  API is cross-origin, production needs a fetch wrapper with `credentials: "include"` and the Go
  server needs credentialed CORS. The stub should mirror that decision, not paper over it.
- **Binary vs JSON:** the map settled on binary Protobuf, but `useBinaryFormat` defaults to false.
  The transport must set `useBinaryFormat: true` deliberately; the stub supports either, so tests
  will not catch a missing flag unless the fake asserts the request `Content-Type`.
- **SSR:** if Connect calls ever move into universal load, only the stub server (or a Node-side
  MSW `setupServer` running inside SvelteKit) will see them. Keep this in mind when the first
  real page lands.
- **Playwright browsers:** Playwright's preflight handling differs by engine (the auto-answered
  preflight behaviour is Chromium-specific, per the issues above). The stub server avoids the
  dependency on that behaviour.
- **Test isolation:** decide between a reset RPC, a fresh server per Playwright worker, or
  serial tests. The reset RPC is the least machinery but is itself part of the fake API.
- **Dependency vetting:** ticket 01 must approve `@connectrpc/connect-node` (dev-only) for the
  stub, or `@connectrpc/connect-playwright` (0.x) if the runner-up is chosen. `msw` +
  `@msw/playwright` is the third vetting option.
- **`@connectrpc/connect-playwright` is unary-only and beta**; revisit if it gains streaming or
  reaches 1.x. It is the fallback if an extra process proves unacceptable.
- **MSW's service worker cannot intercept preflight at all** (Fetch spec §4.8); do not plan a
  cross-origin Connect fake around service-worker MSW.

## Sources

Primary sources consulted:

- Connect protocol: <https://connectrpc.com/docs/protocol>
- Connect-ES client transport source: <https://github.com/connectrpc/connect-es/blob/main/packages/connect-web/src/connect-transport.ts>
- Connect testing guide: <https://connectrpc.com/docs/web/testing>
- `createRouterTransport` source: <https://github.com/connectrpc/connect-es/blob/main/packages/connect/src/router-transport.ts>
- `@connectrpc/connect-playwright` repository and source:
  <https://github.com/connectrpc/connect-playwright-es>,
  <https://github.com/connectrpc/connect-playwright-es/blob/main/packages/connect-playwright/src/create-mock-router.ts>,
  <https://github.com/connectrpc/connect-playwright-es/blob/main/packages/connect-playwright/README.md>
- `connectNodeAdapter` source and README:
  <https://github.com/connectrpc/connect-es/blob/main/packages/connect-node/src/connect-node-adapter.ts>,
  <https://github.com/connectrpc/connect-es/blob/main/packages/connect-node/README.md>
- Connect Node getting started: <https://connectrpc.com/docs/node/getting-started>
- MSW browser integration: <https://mswjs.io/docs/integrations/browser>
- MSW streaming: <https://mswjs.io/docs/http/mocking-responses/streaming>
- MSW cookies: <https://mswjs.io/docs/http/mocking-responses/cookies>
- `@msw/playwright` README and fixture source:
  <https://github.com/mswjs/playwright/blob/main/README.md>,
  <https://github.com/mswjs/playwright/blob/main/src/fixture.ts>
- MSW Playwright discussion: <https://github.com/mswjs/msw/discussions/2404>
- Fetch standard, CORS-preflight fetch: <https://fetch.spec.whatwg.org/#cors-preflight-fetch>
- Playwright Route API: <https://playwright.dev/docs/api/class-route>
- Playwright web server config: <https://playwright.dev/docs/test-webserver>
- Playwright cross-origin mocking issue: <https://github.com/microsoft/playwright/issues/12929>
- SvelteKit load: <https://svelte.dev/docs/kit/load>
- SvelteKit page options: <https://svelte.dev/docs/kit/page-options>
