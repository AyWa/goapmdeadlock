## fix

it has been fixed in https://github.com/elastic/apm-agent-go/pull/997

## APM deadlock

This is an example project to reproduce deadlock issue that we have on production.
We have our server that use APM wrap (server) and APM wrap for (http.Client)

In some case a `http.Do()` can be stuck forever (at least more than few minutes). Without APM it is working correctly. That is why we suspect `APM`

### How to reproduce ?

We have our main server `main.go`. The handler of this server will generates N sub request with a `10ms` timeout. The sub request are made to our `fakeserver/main.go` (that answer after 20 Milli).
The main server will log every X seconds the number of request it processed, and the number of sub request that are not done.
We can call our server with `curl localhost:8082` (we do not need to call concurrently).

At beginning everything seems to work correctly, the request answer correctly. We can see on the main server log like that that show that everything is fine:
```
Open request 1 Done request 217
Open goroutine req 132 Done goroutine req 43468
Open request 1 Done request 221
Open goroutine req 134 Done goroutine req 44266
Open request 1 Done request 225
Open goroutine req 139 Done goroutine req 45061
```

However at some point a request is stuck and will not reply (even if the main request has a timeout and all the sub request too). Some goroutine are stuck forever. (deadlock ?)
```
Open request 1 Done request 322
Open goroutine req 134 Done goroutine req 64466
Open request 1 Done request 322
Open goroutine req 134 Done goroutine req 64466
Open request 1 Done request 322
Open goroutine req 134 Done goroutine req 64466
Open request 1 Done request 322
Open goroutine req 134 Done goroutine req 64466
Open request 1 Done request 322
Open goroutine req 134 Done goroutine req 64466
```
