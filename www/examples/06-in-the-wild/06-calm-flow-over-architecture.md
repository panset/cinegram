<!-- Generated from 06-in-the-wild/06-calm-flow-over-architecture.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# conference signup flow

A CALM flow playing over the architecture it runs on: an attendee signs up for a conference — the worked example CALM itself uses — and the request travels the very nodes and relationships that architecture declares: an actor, a webclient, a load balancer, an attendee service, a notification service and an attendee database, the last four `deployed-in` one Kubernetes cluster. Asked for in finos/architecture-as-code#2999 (https://github.com/finos/architecture-as-code/issues/2999), which proposes picking a published flow in CALM Hub and watching it as a live diagram with play, pause, step and scrub.

<div class="cinegram" data-cinegram="06-in-the-wild/06-calm-flow-over-architecture" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=rFndjttGsn6VAg-M2DmSZsbJcWwdHBw4jhcONpsE8QDBwjKgFrsoNtTqpruaQwuGgb3aB1jsu-z9PkqeZFHVTYrUaCa_V_ZQ3cXq6q---qr4obgpllezAl0Mh2JZXD6ZGzePNc47Y_XF5ZN5qex-Xlnfzf0NhrkKZW0ilrENuNDbfTErfIPuN26tjEUqlm8-FM1vtBCLZRHxfSxmhS6WxYMH8BxePP_mL8C7oLHqYNwWeDfEGmFsAUyE0DoC75agHKgY0WlEILN1BG2zcg8eQOUDKCi9qzCgKxF--ts_xVTnww414Hu1byyml5pIaCtoCUnWKadlbcB3LVIUgzGoG7Qkz28wHMB5jSRLA1oVjXdUm4YXqDh1WGNpVUBid8WWKqMPM1DQ4aa0Bl3kP6xXGjbKKlci_zo-GoYbUyKvcj6aypTyQjGWfxNPxnu0imqjCGfislUUofJtgLXGxvoD6rlxa_AO4c_tBoPDiCQGS9tSxLCA58SR4kgaB5Vxni7G55ormpde4389fvbsmex8WMfY0PLiYmti3W4Wpd9f3L3vwhC1SBe8_dEMutqUNTTBN56yJ40pd4wDBU27sYZqdocBYly6uFftRs7dqVjWvNJEUMSxNDcI2qhtUHvoTKyTPasOM2hUy1GhiI1spjK0m4UsmM_n8u-P9UEWS-jkjZ8yFj89A0ZHEZUGX8EGyWiG5wKeg8aoyhp1vqJ3rYCw9yigDqpLYGpUiKY0jXKR2PmNCgilt-3e0QzIg4KASmOAWqWwRA-1txpi5zlE7AdxSGKNJkDN7giCQ0sxwTHWeACKxlpQ24C4gFcYMBmrkY9BgoRsbQkoEK99w78oB6i3ePvwyrJnhwHgCdwc_3XpncMy0nqSHTNxbIpAk8KQUQcb_57_7nOuqkwJ6CIGgnSQaZ5wrGNQjgy_I522VM75CBsEjrKTI_T2-oVd7QknvoH2SJxegO8NxYEw-jsrayx3NCBiCpjXERuCPSoX5XpU06CD6LfI8V3AC-NQrNQqRTqaPeM1tDZRkyrFDzFmnABJJYhSVCEOphJuiRlQ7KiKg6ac5x_FkiCmUSaAr05OncmpMnx3ETxD0hCQ2Tf2IObkjQwudo7xxUelBVzzPpPcW5d-vzdxvUwcyelI0AXDxDNQJ5Nx2zCOHG8cJbDcEgKpfbrkKqg98hvy2zdY-cTyGRkardlgUJHTwsWgKC5hLdexhi6ohpjA2dtkjz3mdFCwJny3lhwS6DrqMPToSGFtXTRWfn3XCuIVgVVOoz65X-aIDQfFIWri3U5iRFFFnAlsnApB0JQOk7LTWupP3OGGTEyH5hxfb1W7xbVkt3FbGhCXyw4YnfIlMeMIvJKtzFuIjnclk8z_IDblBikfLgWS6SvfkQREB980cslMQVYdMm0curpnhsQXpA6UPej9irWhxAvsm9CHOzLlwbdyvIfrz-Gnv_8D_mct1viUdBuyM1g_gf-GL9Z5QY_zRwu-PRUJLFZxnWMolvrC1uODkQQxmHIntYofYVVhGZmUJfDB8B1xGWBQaLTqsFwD-UyAmVdK37oogSGWB2W0Bz5eOtrX375--cO1YIMgKMm2WCuXVpgIhDGlh6-qZb6GzjjNYORwiMXM51JxGIjgmGiBat-xGmA_kz-MQ-eT8khkghoOGBO0GgxzyZW1RiqXa7DGYXoN-5q4oFa26iMgKB2Qsi57NkqQxTU0wXD5iTXuhcAd-KAxoAbLXOirRDt9lU6RT-mmrN2ocgc3Bjt5mZR1ULQjuY9cFoW5R1V5sXIMl7LmNLz-auVgkC4P3zzP_3v7iJ8fVdw859CbF0dh92N69HbleC21m21QTQ27pzTPNeXNUeDAi_ToLa8FKSXzvpR8-PANV5Yv858fP6Y1vVc0eAWvk-TKRsaKjN58O_rrZOFgaU7RB3zzcDD4VYb0I1mJTqfDDFpuPl-1l5ef4ZlQnA_QccfkiLx48uC4bnBu_GI683ty_o5Vk1jwmsmD29b4nFSiU8F4WBUjvZ4LCGNkVcAHoAZRL-FqcQkf81VLdWw3exNhVXBeHPWyPKVxJap82LOhdBWSN7Aqxl3HBrfGicpgGZ0kOvM6P-hVdsq_4T2V8HvPfj7sM5EMp-ivg6ssW2Ax6zQ0niKJUnzJQiuKdk28KOTaC6-pQLmlv5glTjSYHqVoV48ELLPNJlE9F2GmHZPOZu1iVaSwyMoj6u4EHXwAqzZoOYRO7XEGuFfGziCacocR4qHBVTED3YYlfHF5uSfI2VSbbW3Nto7nrVI8WFyKGrrBfk8i0_u96Fu1YgY3yrbIz8hs55-Xj6tV8WssHevsxNgVk-DVZW9rjEG-zYzA3l6prJ3o2k8IquC5vngfbuPw-k7YjJpEFn1OupwEuP6iPiHYBN-RqEM_lGe5ehMoCpykrFhUN5h-2SvumERniRQSZbaAr7l3DYZXnZPbqZn0zqb-t5eqYwEvC9oozzsfrJZTWOs71KyNldYBiaaYO8dg5whsfFHff_f6Gi6OJPTvf8Gr6-vvX_8c8E4N_mbM3YGUx3cjJfg2IqyKr8-HbRB-Ujk7FTTq81iRTj5NIDIYuG8SVTlprviXUfXLL0uysm-iY9L2p2xjzY418yHJiQX8yZct9Yqx97qX6LlHojS3Oaqyuv9z7zVaabGWE4prgrlhbcIOlTWqZpYlZcIZjtqBMngiTO2F4Y7QYTyiiL0bF_0RuE4q3mlB-wWYguXTy6eX54FFGCe2NkpvBQe1ctoat5VtojmnG38HxD67G2LSF8Gq-O5UaS9ZkqXbGk-GHNSoArCEOw-1fr7UcTfA9TDg1gyQlWsW8ofYyYiKfJrC7Gh0dyJtiZt16clgr3bCQ4wAblWJ9TsL7lK1hHP2a55F_LEbNm5rc5PYt0rTnk4q-D093ag2HgV-pjmH72OybUR6D8gifNeH5aRAEtwhjsY3-PrlNy9fXMMV_Pjq5Q8vc6j-D_5_ANOTMSZO35Dt3YvZS-7BBarj22HmGL9h1vNcQGq8o4HpJmi85yDHbmxaF6-e_iH1NTeL9wBbpg4Mytwi3jt_uD14OItuwtQ1dX4yJsltU7qLZPEgC08nPP0UpcdjnyzDLKlTJg5tqXGEMsvpD3NMkN5NWsCXPtZ5iiH1emJ4ND1JkxPYoORMHtmlOVIafxzHR8e--n9Tw8-5nBpu6b6zh4rho8i7AfzHqjmG3tmK-etTI_fUg958ntVahu0V_YzpaW8xMpyDmaX_ogyo4igfruj3g_6LEatfTWrBqVdDPXjXYpu9ON33O_Im3d89WeMqE_aZyx9fXkGn7G40OJl-vJBOvm3ONUoBG3vI5AgKHHaTvTkbe3IZfy4ZDYNRb1my9CKTNeeMwRoYvVyRONtElTgPe76MNG_jBqz0TkuEfcXbOT_k20RMg5ixDuH0S61aFsapVerbEyMzNZsyKzEy_54GXqmQoANCF893RnSHOP1lDRNfwosESSbtU9QvHt_N1c7Hs9ZXxV99-0kY6MEOA-x04P4dnLtG4xI2yOf5OQXzhxD7UyH2Z3dDNF1AAqh8phpNOtO8TVCZjyaN-XmlMs68o2zxYSfjUhMJfOegtL7cTQdZ0NXeIjTeOEFX5o_UrfelZPShhy3I90knPfx1XyqQnc0TUVEjyuWx47hdm3xifNUPAn0nGaYO-cuUD3lMKCJJUnX4jKS2it0ZRIwsmXxlncD2ZAZzQszjK8xskSIo9zLg8tn5Zmpk5WxVuJcROcP-mOb88ha8io9vP_4nAAD__w){ .md-button }

??? abstract "The source — `06-in-the-wild/06-calm-flow-over-architecture.dgm`"

    ```dgm
    %% A CALM flow playing over the architecture it runs on: an attendee signs up
    %% for a conference — the worked example CALM itself uses — and the request
    %% travels the very nodes and relationships that architecture declares: an
    %% actor, a webclient, a load balancer, an attendee service, a notification
    %% service and an attendee database, the last four `deployed-in` one Kubernetes
    %% cluster. Asked for in finos/architecture-as-code#2999
    %% (https://github.com/finos/architecture-as-code/issues/2999), which proposes
    %% picking a published flow in CALM Hub and watching it as a live diagram with
    %% play, pause, step and scrub.
    %% ---
    %% Why play the flow *over* the architecture instead of beside it. A detached
    %% sequence diagram redraws the participants as bare columns, so a reader has
    %% to hold two pictures in their head and trust that they still agree. Here
    %% there is one picture: every hop is an edge the architecture already declares
    %% as a `connects` relationship, and `deployed-in` is the cluster box the
    %% traffic enters at the load balancer. A transition that cannot be drawn is a
    %% transition whose relationship does not exist — the diagram checks the flow.
    %% ---
    %% Steps meant to happen together. Cinegram has one timing rule — actions
    %% inside a step start together, steps run one after another — so a pair of
    %% transitions that fire at once is simply one step holding two flows. That is
    %% `commit`: the row is written and the signup event is published on the same
    %% frame. The step before it is the deliberate contrast: `check` wraps its two
    %% flows in a `seq`, so the answer cannot start until the query has landed.
    %% ---
    %% Scrubbing needs standing state, not narration. The two pills on the website
    %% are `gauge` readings — the request id, and which transition is on screen —
    %% and a gauge holds until it is overwritten, so dropping the playhead anywhere
    %% still says which request this is and where in the flow you are (`4 → 5`
    %% reads one after another, `6 + 7` reads together). `seats left` on the
    %% database is the same trick for the effect of the write, with a `delay:` so
    %% that the count drops exactly when the INSERT lands rather than when it sets
    %% off: gauge windows are exact, so a scrub can never show a write that has not
    %% happened yet. The per-step `desc:` lines are the other half of the state —
    %% `cinegram narrate` prints them as an ordered list of steps, which is the
    %% fallback view the issue asks for beside the live diagram.
    flowchart TD
      attendee([Attendee])
      conference-website[Conference Website]

      subgraph k8s-cluster[Kubernetes Cluster]
        load-balancer{{Load Balancer}}
        attendees[Attendee Service]
        notifications[Notification Service]
        attendees-store[(Attendee Database)]
      end

      attendee --> conference-website
      conference-website --> load-balancer
      load-balancer --> attendees
      attendees --> attendees-store
      attendees --> notifications
      notifications --> attendee

    scenario "conference signup flow" { speed: 1.0 }

      step submit "The attendee submits the signup form" {
        desc: "A CALM flow begins at an actor, not at a service. The attendee fills in the form the conference website is serving and posts it. Everything after this hop is a relationship the architecture has already declared, which is why the flow can be played over it at all."
        flow attendee -> conference-website { label: "name, email, ticket type", dur: 700ms }
        highlight conference-website { style: active }
        gauge conference-website { label: "request", value: "sig-4c2f" }
        gauge conference-website { label: "transition", value: "1 of 10" }
      }

      step post "The website calls the cluster's front door" {
        desc: "The conference website is a webclient running in the attendee's browser, so this is the first hop that leaves the machine it started on. It arrives at the load balancer, the only node inside the cluster the outside world is allowed to address."
        flow conference-website -> load-balancer { label: "POST /attendees · HTTPS", dur: 700ms }
        highlight load-balancer { style: active }
        gauge conference-website { label: "transition", value: "2 of 10" }
      }

      step route "Inside the cluster the request is forwarded" {
        desc: "The four nodes in the box are `deployed-in` the Kubernetes cluster, and in CALM that is a relationship like any other. Focusing the cluster is the diagram saying the same thing the model does: this hop is private and cheap, where the one before it crossed the internet."
        focus k8s-cluster
        flow load-balancer -> attendees { label: "POST /attendees · HTTP :8080", dur: 700ms }
        set attendees { badge: "handling", delay: 700ms }
        gauge conference-website { label: "transition", value: "3 of 10" }
      }

      step check "One after another: ask the database, then hear back" {
        desc: "The service will not register the same email twice, so it asks before it writes. A `seq` makes this pair strictly cause-then-effect inside a single step — the answer cannot begin until the query has landed, which is exactly what the next step is not."
        seq {
          flow attendees -> attendees-store { label: "SELECT 1 WHERE email = ?", dur: 600ms }
          flow attendees-store -> attendees { label: "0 rows · not registered", dur: 600ms, style: response }
        }
        gauge attendees-store { label: "seats left", value: "118" }
        gauge conference-website { label: "transition", value: "4 → 5 of 10" }
      }

      step commit "Together: the row is written and the event is published" {
        desc: "These are two transitions of the flow and they are meant to happen at once — the service does not wait for the insert to commit before it publishes. Both flows leave the service on the same frame because every action in a step starts together; the pill reads 6 + 7 for that reason."
        highlight attendees { style: active }
        flow attendees -> attendees-store { label: "INSERT attendee A-4c2f", dur: 1s }
        flow attendees -> notifications { label: "publish signup.created", dur: 1s }
        gauge attendees-store { label: "seats left", value: "117", delay: 1s }
        set notifications { badge: "queued", delay: 1s }
        gauge conference-website { label: "transition", value: "6 + 7 of 10" }
      }

      step confirm "The 201 walks the same relationships back up" {
        desc: "A reply is not a new relationship: the response travels the `connects` edges it arrived on, in reverse, and CALM no more needs a second set of arrows for it than the diagram does. The browser has a ticket id while the email has still not been sent."
        flow attendees -> load-balancer -> conference-website { label: "201 Created · A-4c2f", dur: 1.2s, style: response }
        note conference-website "You're on the list — ticket A-4c2f" { side: below }
        set attendees { badge: "" }
        gauge conference-website { label: "transition", value: "8 → 9 of 10" }
      }

      step email "The last transition lands back on the actor" {
        desc: "The notification service works on its own clock, which is the whole point of publishing an event instead of blocking on it. The flow ends where it began, with the attendee — and the Hub can now replay, step or scrub this same sequence against exactly this architecture."
        flow notifications -> attendee { label: "confirmation email", dur: 900ms }
        highlight attendee { style: active }
        set notifications { badge: "sent" }
        gauge conference-website { label: "transition", value: "10 of 10" }
      }
    ```

← [claude code tool call](../06-in-the-wild/05-claude-code-tool-call.md)  
→ [escaping the self-loop note hack](../06-in-the-wild/07-flowchart-node-notes.md)
