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


```dgm
%% An OIDC login through authena, as the shared engine actually runs it.
%% ---
%% `oauth-login.dgm` tells the protocol story in the abstract. This one is the
%% same handshake with the implementation left in: which process holds which
%% secret, what is written to Postgres and when it is deleted again, and the
%% four separate checks that stand between a token arriving and a session
%% existing. Every step's `desc` says what would go wrong without it.
%% ---
%% The flows carry no `label` — a sequence diagram already draws the message
%% text, and a label would print a second copy on top of it. Notes take
%% `side: below` for a related reason: an actor box sits at the top of the
%% stage, so a note above it has nowhere to go but over its name.
%% ---
%% A message an actor sends to itself is an edge like any other — `flow E -> E`
%% sends the particle round mermaid's loop. The four self-messages here are the
%% engine's own work, and they are the steps where nothing else moves.
sequenceDiagram
  autonumber
  participant U as Browser
  participant A as authena
  participant E as clusterlib/oidc engine
  participant PG as PostgreSQL
  participant IdP as External IdP

  U->>A: GET /ssp/auth/oidc/login?provider_id=...
  A->>E: FlowService.Login(LoginRequest)
  E->>E: rebuild OAuth2/verifier from stored config (no stale cache)
  E->>E: generate state, nonce, PKCE verifier/challenge
  E->>PG: CreateState (10-min TTL)
  E-->>A: authorization URL (+ auth_url_params)
  A-->>U: 302 to IdP
  U->>IdP: authenticate (+MFA)
  IdP-->>U: 302 to /ssp/auth/oidc/callback with code and state
  U->>A: GET callback
  A->>E: FlowService.Callback(CallbackRequest)
  E->>PG: FindAndDeleteState(state)  (single-use, CSRF)
  E->>PG: GetConfigWithSecret(providerID) → decrypt client_secret
  E->>IdP: token exchange (configured auth method + token_request_params)
  IdP-->>E: id_token, access_token, (refresh_token)
  E->>E: enforce configured alg, verify signature (JWKS or HMAC), issuer, aud, nonce, exp
  E->>IdP: (if userinfo_enabled) GET userinfo, enforce sub match, merge claims
  E->>E: map username/groups → roles
  E->>PG: CreateSession (encrypt tokens, lifetime/idle per config)
  E-->>A: CallbackResult{RedirectURL, Cookies}
  A-->>U: Set-Cookie + redirect into the app

%% The storyboard is the other half of this diagram's point. Eighteen messages
%% fly and the person signing in sees five screens; a `scene` says which one is
%% in front of them while each beat plays. Scenes are sticky, so the six steps
%% between `callback` and `signedin` — the whole verification back-and-forth —
%% all sit under one motionless interstitial, which is exactly what they look
%% like from a chair.
%% ---
%% Where a scene sits inside its step matters as much as which step it is in.
%% A screen changes when an arrow *lands*, not when its step begins, so every
%% scene below that follows a message is written inside the `seq` after the hop
%% that causes it — a scene consumes none of the chain, so it means "here" and
%% keeps meaning it if a duration changes. The steps that end on a new screen
%% then carry a `dur` long enough to look at it.
storyboard "What the person signing in sees" {
  frame app_signin  { img: "frames/app-signin.svg",    caption: "Acme's own sign-in page. One button, no password field." }
  frame idp_form    { img: "frames/idp-login.svg",     caption: "The provider's domain. The password is typed here and nowhere else." }
  frame idp_mfa     { img: "frames/idp-mfa.svg",       caption: "A second factor, entirely between the user and their IdP." }
  frame app_waiting { img: "frames/app-signing-in.svg", caption: "Back on Acme with a code in the address bar — worthless without the secret." }
  frame app_home    { img: "frames/app-home.svg",      caption: "Signed in, holding an opaque cookie and nothing else." }
}

scenario "sign in with an external IdP" { speed: 1.0 }

  step ask "The browser asks authena to start a login" {
    desc: "The request names a provider and nothing else. Which IdP that is, what it is called, and what secret talks to it are all server-side facts, because anything the browser carries is anything the browser can change."
    flow U -> A { dur: 700ms, msg: 1 }
    highlight A { style: active }
    scene app_signin
  }

  step build "The engine rebuilds its client from stored config" {
    desc: "The OAuth2 client and the token verifier are constructed from the database row on every request rather than cached at startup. An admin who rotates a client secret or repoints an issuer expects the next login to use it — a cached client would keep using the old one until something restarted."
    seq {
      flow A -> E { dur: 600ms, msg: 1 }
      flow E -> E { dur: 700ms, msg: 1 }
    }
    highlight E { style: busy }
    note E "no cached client:\nconfig can change\nunder a running pod" { side: below }
  }

  step mint "state, nonce and a PKCE pair are generated" {
    desc: "Three random values with three different jobs. `state` ties the callback to this browser, `nonce` ties the eventual id_token to this request, and the PKCE verifier proves at the token exchange that whoever redeems the code is whoever asked for it."
    flow E -> E { dur: 800ms, msg: 2 }
    highlight E { style: busy }
    set E { badge: "state · nonce · PKCE", state: minting, color: "#7c3aed" }
  }

  step store "Only the challenge is written down" {
    desc: "The row holds the state, the nonce and the PKCE *verifier* under a ten-minute TTL. It expires on its own because an abandoned login is the common case, not the exception, and nothing should have to come back later to clean it up."
    flow E -> PG { dur: 600ms, msg: 1 }
    gauge PG { label: "state TTL", value: "10 min" }
  }

  step redirect "The browser is handed to the IdP" {
    desc: "authena replies with a 302 and stops taking part. From here until the callback the conversation is between the user and their identity provider, which is exactly why authena can never leak a password it was never shown."
    seq {
      flow E -> A { dur: 500ms, style: response, msg: 1 }
      flow A -> U { dur: 500ms, style: response, msg: 1 }
    }
    %% No scene: the browser is still showing Acme's page while the 302 is in
    %% flight, and the provider's form is what the next step is about.
  }

  %% Two screens in one step, and the diagram is nearly still for both of them —
  %% which is the honest picture. One hop covers the entire authentication, so
  %% the only thing that moves while the user types is the storyboard.
  step authenticate "The user authenticates, and does their MFA" {
    desc: "Whatever the provider demands — password, push, hardware key, a conditional-access policy nobody here has heard of — happens entirely inside this hop. Adding a second factor is a change to the IdP and to nothing in this diagram."
    dur: 2800ms
    scene idp_form
    seq {
      flow U -> IdP { dur: 900ms }
      %% The password is submitted, and a second factor is asked for.
      wait 700ms
      scene idp_mfa
    }
    note IdP "password + MFA\nnever seen by authena" { side: below }
    highlight IdP { style: active }
  }

  step callback "The IdP sends back a code, not a token" {
    desc: "The redirect carries a short-lived authorization code through the address bar, where it lands in browser history and in any proxy log along the way. A code is safe there precisely because it is worthless without the client secret and the PKCE verifier that never left the server."
    seq {
      flow IdP -> U { dur: 700ms, style: response }
      %% The redirect lands and the address bar carries the code. This frame
      %% then stands for six steps: everything from here to the cookie happens
      %% server-side, and a storyboard that kept changing would be lying about
      %% how much the person can see.
      scene app_waiting
      flow U -> A { dur: 600ms, msg: 2 }
    }
  }

  step consume "The state is looked up and deleted in one move" {
    desc: "FindAndDeleteState is a single-use read: an unknown state fails and a replayed one fails too, because the first callback took it. That is the whole CSRF defence, and it works only if the delete cannot be raced — which is why it is one statement and not a read followed by a delete."
    flow A -> E { dur: 500ms, msg: 2 }
    flow E -> PG { dur: 600ms, delay: 400ms, msg: 2 }
    unset E
    note PG "single use:\na second callback\nfinds nothing" { side: below }
  }

  step secret "The client secret is decrypted for this exchange only" {
    desc: "The secret lives encrypted at rest and is decrypted per request into the exchange that needs it. It is the one credential that proves this cluster is the client, and it is the reason the code has to come back through the server rather than being redeemed by the browser."
    flow E -> PG { dur: 600ms, msg: 3 }
    highlight E { style: busy }
  }

  step exchange "Code, secret and verifier are traded for tokens" {
    desc: "Back channel, server to server, over a connection the browser is not part of. The configured auth method matters here: an IdP expecting client_secret_basic will reject a body-posted secret, and the failure looks like a bad credential rather than a mismatched convention."
    seq {
      flow E -> IdP { dur: 800ms, msg: 1 }
      flow IdP -> E { dur: 800ms, style: response }
    }
  }

  step verify "Four checks, and the first one is the algorithm" {
    desc: "Signature, issuer, audience, nonce and expiry each rule out a different attack — but the algorithm is enforced first and against the configured value, never against what the token asks for. A token that nominates its own algorithm can nominate `none`, and then the signature check it passes is one it wrote itself."
    focus E
    flow E -> E { dur: 1200ms, msg: 3 }
    highlight E { style: busy, dur: 1200ms }
    %% The note lives here alone and the badge waits for the next step: both
    %% hover over the same actor, and together they overlap.
    note E "alg from config, not from the token\nnonce must match the one we stored" { side: below }
  }

  step claims "Claims become a username, groups, and then roles" {
    desc: "userinfo is optional and its `sub` must equal the id_token's, or the two halves are describing two different people. The mapping from groups to roles is the point where an external directory finally becomes local authority."
    seq {
      flow E -> IdP { dur: 700ms, msg: 2 }
      flow E -> E { dur: 600ms, msg: 4 }
    }
    set E { badge: "id_token verified", state: verified, color: "#16a34a" }
    gauge E { label: "claims", value: "sub · email · groups" }
  }

  step session "A session row is written, with the tokens encrypted" {
    desc: "The session — not the id_token — is what the rest of the product checks. Storing the tokens encrypted alongside it is what makes refresh possible later without ever handing a bearer token to the browser."
    flow E -> PG { dur: 700ms, msg: 4 }
    gauge PG { label: "session", value: "lifetime + idle" }
    set PG { badge: "tokens encrypted", state: ok, color: "#16a34a" }
  }

  step signedin "A cookie comes back, and nothing else does" {
    desc: "The browser ends up holding an opaque session cookie. It never saw the client secret, the id_token or the refresh token, so a compromised browser costs one session rather than standing access to the IdP."
    %% The step outlives its last hop on purpose: the cookie arriving is the
    %% payoff, and a scene on the final instant of the scenario would flash the
    %% signed-in page for one frame and stop.
    dur: 2100ms
    seq {
      flow E -> A { dur: 500ms, style: response, msg: 2 }
      flow A -> U { dur: 600ms, style: response, msg: 2 }
      scene app_home
    }
    highlight U { color: "#16a34a" }
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
