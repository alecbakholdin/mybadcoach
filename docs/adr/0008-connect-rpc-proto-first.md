# Connect RPC transport with proto-first contracts

Backend and client speak Connect RPC (connectrpc.com): protobuf contracts compiled by buf into Go
and TypeScript, served by connect-go over plain HTTP POST, consumed by @connectrpc/connect-web in
the SvelteKit client.

The product is mobile-web-first and used for hours at a time, and the core loop is real-time
event streaming. We chose binary protobuf encoding over JSON REST for payload efficiency, and
Connect over raw gRPC because browsers cannot speak gRPC without an Envoy/grpc-web proxy in front
of the server — and we wanted no intermediate layer at all. Connect gives gRPC-shaped unary and
streaming calls with protobuf bytes on the wire and no proxy; the future live-sync MatchActor
becomes a bidirectional stream on the same transport instead of a WebSocket.

As the schema of record, `.proto` files replace the hand-written `events.ts`/`state.go` mirrors:
types are generated in both languages, so the wire schema cannot drift between ports. The
hand-authored parity fixtures remain the behavioral contract and stay hand-written, human-readable
JSON under the existing canonical-JSON equality rules. The OpenAPI/spectral plan is dropped in
favor of buf lint + buf breaking on the proto tree.
