# websocket-handshake

## the upgrade

`main` · sequenceDiagram · opens here

### Scenario: the upgrade

`s0` · 3.2s long

#### 1. The browser asks to change protocol

0.0s–0.7s

A WebSocket connection starts life as an ordinary HTTP request. That is the whole trick: it travels through proxies and firewalls that would never have let a new protocol through on its own.

- **C → L** carries a message (0.0s–0.7s)
- **L** is highlighted (active) (0.0s–0.7s)

#### 2. The load balancer passes it through

0.7s–1.3s

A proxy that does not understand the upgrade header will happily strip it, which is why this hop is where most WebSocket deployments break.

- **L → S** carries a message (0.7s–1.3s)

#### 3. The server agrees to switch

1.3s–2.1s

101 is the only status code that means the connection is no longer HTTP. From here the same TCP socket carries frames in both directions, and neither side has to ask before sending.

- **S → C** carries a message (1.3s–2.1s)

#### 4. Frames travel both ways

2.1s–2.6s

Ping and pong are not application messages — they are how each side finds out the other is still there, on a connection that may sit silent for hours.

- attention narrows to **S** (2.1s–2.6s)
- **C → S** carries a message (2.1s–2.6s)
- **S → C** carries a message (2.1s–2.6s)

#### 5. The browser closes it

2.6s–3.2s

A close frame is a request, not a fact. The connection is not gone until both sides have sent one, which is why a well-behaved client waits rather than dropping the socket.

- **C → S** carries a message (2.6s–3.2s)

#### Standing state

State that outlives the step that set it, with the window it holds for.

- **S** is badged "upgraded" and marked *open* (1.3s–2.6s)
- **S** reads open = "1" (1.3s–2.6s)
- **S** reads open = "0" (2.6s–3.2s)

### Interactions

- Clicking **S** opens the **internals** view — labelled "What the server is doing"
