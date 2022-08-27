# go-http-streaming

Proof-of-concept for file streaming via http in go.

Rationale:
- build an API that can serve dynamic CSV without having a fixed datasource
- ditch writing to disk which can impact performance
- serve files using complex business logic and low-memory requirements
