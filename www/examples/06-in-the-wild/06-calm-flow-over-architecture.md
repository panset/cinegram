<!-- Generated from 06-in-the-wild/06-calm-flow-over-architecture.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# conference signup flow

A CALM flow playing over the architecture it runs on. An attendee signs up for a conference, and the request travels the very nodes and relationships the architecture declares: an actor, a webclient, a load balancer, an attendee service, a notification service and an attendee database — the last four `deployed-in` one Kubernetes cluster. Asked for in finos/architecture-as-code#2999 (https://github.com/finos/architecture-as-code/issues/2999), which proposes picking a published flow in CALM Hub and watching it as a live diagram with play, pause, step and scrub.

<div class="cinegram" data-cinegram="06-in-the-wild/06-calm-flow-over-architecture" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=rFnhjtvGEX6VAQsjdirpdE7q2CqK4uK4cNA0CeIDjMIyoCU5FBe32qV3lkcLBwP91Qco-i7930fJkxQzu6RIne6SuP5j41bL4ezMN998s7zJrrPV-SxDG_w-W2XLJ3Nt56HGeadNebZ8Mi-U2c0r47q5u0Y_V76odcAitB4X5XaXzTLXoP3IRyttkLLVm5us-UgLIVtlAd-HbJaV2Sp78AAu4PnFd38Dfgoao_baboGfhlAjjC2ADuBbS-DsAi4sqBDQlohAemsJ2mZtHzyAynlQUDhboUdb4AyULcWWx3ctUoDg1TUakrVr9HuwrkSSbR6NCtpZqnVDYu6WEyUWRnmkFSgLqgjOz0BBh3lhNNrAfxinSsiVUbZA_tWKpYO76K-1OAbWBV3pQt7Zr4sjanS8UgWVK0L4-R__Zn_EmlEUoHKth02JjXF7LOfabsBZhL-2OXqLAQkK01JAv4ALusJSgqOjO5W2js7GR5srmheuxN89fvbsmex5WIfQ0OrsbKtD3eaLwu3O7n7uTBO1SGf8-KMZdLUuami8axxhDGajiytOr4KmzY2mml3ivGsbQfCyzeX4nQpFzTt1AEUcUn2NUGq19WoHnQ51tGfUfgaNaglnQAEbeZgK3-YL2TCfz-X_1_VeNks65Y2fM8Q-P4ExSwFVCa6CHEmXjLoFXECJQRU1lmKOGEm2OHjksfSqi5BqlA-60I2ygdj5XHmEwpl2Z2kG5ECBR1Wih1oljDmonSkhdI5DxH4QhyTUqD3U7I5g2LeM3lrxP7gHCtoYUFuPuICX6LEHLB-DBAnJ2gpQgF67hn9RFrDc4u3DK8Oe7QeMR9xy_DeFsxaLQJtJjcTimiJQxzAk5EHu3g-gDV5VlS4AbUBPEA8yLReOdfDKkpaakNMWyloXIEfgKFs5Qm-v39jVjnDiG5QOiSsM8L2m0FfPkLOixuKKBkRMAfMqYEOwQ2WDpEc1DVoIbosc3wU81xbFSq1ipIPeMV59a2KdqkL8EGPaCpBUhCgF5cNgKuKWmNjEjqo4aMo6_lEsCWIapT246ujUFONTac5dAMeQ1ASkd43Zizl5I4OLnWN88VFpAZf8nI7ubQq32-mwWUWW5HIk6Lxm_hnIkzm2bRhHlh8cFbBkCYHULia58mqH_Ib09hwrF8k7IaNEo3P0KnBZ2OAVhRVsJB0b6LxqCHQg9jbaY4-5HBRsCN9tpIYEupY69D06YlhbG7SRX9-1gnhFYJQtsTzKL3NEzkGxiCXx01ZiREEFnAlsrPJe0BQP02hjqD_tRU_OKpXdZqvaLW6ktLXd0gC3vuvoWMSRFUfAlUplzkK0ETlM_2JTTEr2KB0sBpGpK-VHglF61zSSYHbTqH2ijH0nXCA8EXlL7Sl5MDTDWlPkBPZNqMMeWHLvWj4hPNx8CT__81_wh3hAgXwkhzFcZ7B5Ar-Hr_pNPcYfLThzKhAYrMKmj-HQ17hQh2bLOJI-xX9gVWERmIz5Lz5zQlaDfi7o2pRIxWoDRlvu4R4HsokVVCtTMbr6eo1JxQ00XjNBhxp3QnEWnC_RYwmG2cJVsTBnYivGLOG3UsbkqriCa42drEjjA0VXJJ5bfC-sIeQ2alyLteWoFjUj9fKbtYWhyT980yPq7SNeP-iXeYc56YBvng9L8DouvV1b3kttvvWqqeHqKc0T7b4ZaYDncekt7wVh23nPtjc33zH5fp3-_PAh7um9osEreBXFSTIy1i305vuxipluHCzNKTiPbx4OBr9J2X8kO9GW8TCD6pnP1-1y-QWeCMXpAB2emByRN08WDvsG58YvphO_R-fv2DWJBe-ZLNy2xuekAq3y2sE6Oxyk51jGyDqDG6AGsVzB-WIJH1KqpYG0-U4HWGdcCAdVKas0JuvK-R0biqmQQoF1NtbbOW61lUZ8ELNMfbzQ69FYcMN7KqHBniSc38VufzhFnw5uRGyB9Z4toXEUSMTUC9YiQeRdpA_hoF6bTHv4LYnChH4kU8rZoT67eqTxCmVZNDAjcp_imULHsxmzWGcxLLLzgLo7QQc3YFSOhkNo1Q5ngDulzQyCLq4wQNg3uM5mULZ-BV8tlzuCVE213tZGb-tw2iqFvcGVCIZr7J-J1H-_F4nC-aXXyrTIa6S38y-Lx9U6-y2WDu1oYuycWfB82dsaY5CzmRDY2yuUMRPp9xlB5Z0NUDrnb-Pw8k7YjMYp1kVWBoEIuD5RnxHk3nUkAsoNXUxSrz0FgZOII4PqGuMvO8VDhUgRUQsiXhbwbQDlveZdpxTpTJacNXFQ7NXcWOPKhjbIeue8KeUUxrgOS24Eqiw9Ek0xd4rBThHYOFE__vDqEs4OJPTf_8DLy8sfX_0S8I4NfjTm7kDK47uR4l0bENbZt6fDNsgjaZ2d8iWWp7Eiw24c1RMYeLTgjj-dP_iX2xNwnFT6OTNE-XvMNkZfsazcR-2wgL-4oqVeWPVe9yo2jREUbywG6RKJTeDmSjQiblYTimu8vmbxyw4VNapmlpRXxBmOFHPhHRFGBa55aLIYDihi78ZNfwSuo4533NB-BaZg9XT5dHkaWIRhYitX5VZwUCtbGm23n4R-vrgbVDIswDr74XhiWrEKm6hLKV_LU7QHVm2nwdXfvXQ8UnMH9LjVA0glsUL3EDq5uiEXryauaJQt0afEE6wMKrBTV8I8nHOe3yh4XQSzh0K1hHP2a54U7mFE1HZr0uTUjxDTQUd69j2Dzqgb4nsl7-vSnUFUpmJby2g8YInwXR-Wo5ZIcIccGmfw1YvvXjy_hHN4_fLFTy9SqP4Efx7g82QMn-M3JHv3onTJg6mAc5wd5orxG2Y9s3mkxlkauG2CxnsOchhTpp3w_OkngXSaou4BtoziDMo0O907lN-exk-imzAORp2b3B2kqSrmIlrcy8bja4_-aqHHY18swwVLp3QYZjZtCeWCoz_MoUB6N2kBX7tQp9FeOvTE8OhKIV4nQI5SM-keK16uxDuBw53KYeD8Y5yEuZbjJCpjafJQMXwUOTuA_9Anx9A72SN_e2l8-_2rFz9dHhTmRdJnCbZPx4Vxh_XpQDGyneKZ9P6i8KjCqCQmtj8e-l8doM_Ef-zNQP7vWmwFgf9_ncR83VMlttJ-l7j78fIcOmXSXZ6AZnKPL7QPbXNqFPLYmH0iQ1BgsZs8m6qvJ5Pxl4PRjSiWWxYlvYxkVTljcHpGK6UvEKI7rIMdhz1eOvGIVThbSlhdxY9zPcglvVz02onS4HKLw1iSvnEY6gcQLZdLJlZSZGD-Pd4Sx8aBFghtOD370B3y89eNRJyE5xF_TNJHKD9fPL6bm60LJ62vs7-79jM_0IEZbnHjgft3cK3qEleQI5_nlzTKJwHoUyHyZ3dDNCYgAlS-1Yyu_LhPJ1Smo8nofVqZnPxE1Dl_JfeGPPG7zkJhXHE1m95TdbUzCI3TVtCVyCLO433rGH3tYAvy7c3KlH7ZtwZkZ9PVoKgPZWfyCWYykA1Xl7z4ss1l-LaukwpT-_R5xvn4dSaKIinV4VuK2ip2ZxAtsmXyBXEC26NbliMiHqcwsUWMoORlwOWz0-PSyMrJLnAvDXKFfZrxe3kLXtmHtx_-FwAA__8){ .md-button }

??? abstract "The source — `06-in-the-wild/06-calm-flow-over-architecture.dgm`"

    ```dgm
    %% A CALM flow playing over the architecture it runs on. An attendee signs up
    %% for a conference, and the request travels the very nodes and relationships
    %% the architecture declares: an actor, a webclient, a load balancer, an
    %% attendee service, a notification service and an attendee database — the
    %% last four `deployed-in` one Kubernetes cluster. Asked for in
    %% finos/architecture-as-code#2999
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
    %% Scrubbing needs standing state, not narration. The pills on the Attendee are
    %% `gauge` readings — the request id and which transition is on screen — and a
    %% gauge holds until it is overwritten, so dropping the playhead anywhere still
    %% says which request this is and where in the flow you are (`4 → 5` reads one
    %% after another, `6 + 7` reads together). `seats left` on the database does
    %% the same for the effect of the write. The per-step `desc:` lines are the
    %% other half: `cinegram narrate` prints them as an ordered list of steps,
    %% which is the fallback view the issue asks for next to the live diagram.
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
        set attendees { badge: "handling" }
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
        flow attendees -> attendees-store { label: "INSERT attendee A-4c2f", dur: 800ms }
        flow attendees -> notifications { label: "publish signup.created", dur: 800ms }
        gauge attendees-store { label: "seats left", value: "117" }
        set notifications { badge: "queued" }
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
