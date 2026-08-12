# Diagrams inside a document

This file is the one to open with `Cmd+Shift+V` when checking the VS Code
extension. Everything below is ordinary Markdown apart from the fenced `dgm`
blocks, which the preview renders as animated diagrams.

## A request through the cluster

The block below is the body of `k8s-request.dgm`, and its `view … from
"pod-a.dgm"` resolves relative to *this file* — clicking Pod A drills into the
other diagram in place, with a Back button to return.

```dgm
%% API request path through a Kubernetes cluster.
flowchart LR
  client[External Client]
  lb[(Cloud Load Balancer)]

  subgraph cluster[Kubernetes Cluster]
    ing[Ingress Controller]

    subgraph ns[namespace: prod]
      svc[ClusterIP Service]
      pod1[Pod A]
      pod2[Pod B]
    end
  end

  client --> lb
  lb --> ing
  ing --> svc
  svc --> pod1
  svc --> pod2

view podA "Inside Pod A" from "pod-a.dgm"

interact {
  click pod1 -> view podA { label: "Zoom into Pod A" }
}

scenario "GET /api/orders" { speed: 1.0, loop: true }

  step edge "Request reaches the load balancer" {
    flow client -> lb { label: "GET /api/orders", dur: 700ms }
    highlight lb { style: active }
  }

  step terminate "TLS terminates at the ingress controller" {
    flow lb -> ing { label: "HTTP/1.1", dur: 600ms }
    highlight ing { style: active }
  }

  step balance "kube-proxy picks a healthy endpoint" {
    flow svc -> pod1 { label: "10.1.2.3:8080", dur: 600ms }
    dim pod2
  }

  step respond "Response travels back to the client" {
    flow pod1 -> svc -> ing -> lb -> client {
      label: "200 OK",
      dur: 1400ms,
      style: response
    }
  }
```

Prose between two diagrams. Editing this paragraph must not reset either
diagram's playhead — that is the whole point of the block identity being a hash
of the block's own text.

## A second diagram on the same page

Two players on one page must not fight over the keyboard: pressing Space here
scrolls the document, and only a diagram you have clicked into responds to it.

```dgm
sequenceDiagram
  participant B as Browser
  participant A as Auth Server
  participant R as Resource API

  B->>A: POST /token
  A->>B: access_token
  B->>R: GET /me
  R->>B: 200 OK

scenario "Token exchange"

  step get "The browser exchanges its code for a token" {
    flow B -> A { dur: 600ms }
    highlight A { style: active }
  }

  step issued "The auth server issues an access token" {
    flow A -> B { label: "access_token", dur: 600ms, style: response }
  }

  step call "The token is presented to the API" {
    flow B -> R { label: "Authorization: Bearer …", dur: 700ms }
    highlight R { style: busy }
  }

  step ok "The API answers" {
    flow R -> B { label: "200 OK", dur: 500ms, style: response }
  }
```

## A block that does not compile

Deliberately broken, to show what a failure looks like: the message and its
hint appear where the diagram would have been, rather than the block vanishing.

```dgm
flowchart LR
  a[A]
  b[B]
  a --> b

scenario "broken"

  step s "This step names a node that is not there" {
    flow a -> nope { dur: 1s }
  }
```
