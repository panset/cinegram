<!-- Generated from 06-in-the-wild/07-architecture-canvas.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# architecture canvas

A persistent architecture canvas for an agent session, asked for in openai/codex#35100 (https://github.com/openai/codex/issues/35100): "keep one architecture view pinned that updates, highlighting the component being discussed", instead of regenerating a disconnected Mermaid diagram every turn. The diagram below is drawn once — it is the same unchanged Mermaid the agent would commit to docs/ — and each scenario is one conversation with that codebase. Every step is one turn: the component under discussion is lit, the file that implements it is the note, and everything else recedes. Nothing is redrawn, so nothing moves between turns, and the second scenario reuses the first's layout exactly.

<div class="cinegram" data-cinegram="06-in-the-wild/07-architecture-canvas" data-height="990"></div>

[Edit in the playground](../../playground/#doc=tFjbbiM30n6VQgeDsYG2ZCfjHDQXP5L8g2SCScZre7FYSEHCbpbUjNhkh8VWWzsIkKt9gMU-wz7YPMmiSHarZUv2zMXeuGUeisWqr6q-4rtsk80u8gyNd9tslp1_fqbMma_wrFNaTs-_OBOurJTH0rcOz0phNoImclVneWYbNB-5Zak0Ujabv8uaj9zps1nm8c5neSazWfbsGXwNDTpS5NF4GG-EuBGW1oEwIFa8gJBIWZODoDXKMKfMwjx7BnwNoaallXj3yWeXF-fncFJ539BsOl0pX7XFpLT1dLxsqohapGlYfTqDRbZGbKI0g_vKbBR20ChjUIKvhIe2kcIj5VCpVaXVqvLKrMBXCKWtG2vQ-CCpQB6XisqWCOUiy0EZ8igk2CU4XKFBJ8JmEZZZY7D0KOFHdLVQEqQSKyfqIA036LbgW2cmcFthPwcFatuBIpBOdAasKRHe__lvUJ4HWSsSNUJrykqYFcogrD-Ap6N5O9tqyReolQdvQdqSpkGQMBJQlBVQiUY4ZVmuNRgEldZs0JHwyhrolK-iidjIhSCcwKugNnls0rZwhdkBc7VGouvNxeIUgVY-D0sZd1G0qhuNNRpPoysa6zFnTXem8hUbFjUhOCxRIk3gJxtHFYHDYK8cyPLuMFzbDRIU6DvECC3WlYLgaEksrZE7QzhsCSkp6Mg_J9Bia1sPeCdKr7eTIOXs7Cx82W2_lspgcFxdNr8CodugA7xrLEtK21giIQPbBcsSnHROebZANDvHVA5aGQ9sIYfBdvFnY52HLpjKgzCqZrCehovu3F0KA0WUxYPsFz6hB_Jww66Khkeo2ppjkdbxvr-3SEG1ycIste3KSjgPb64XBqDUCo2ff-NsR-hgCt--ef3zwvAUtcXKiaYC0ai5aNT0Zx4FcLb16ObX4ZPGROur-detr6BWUmrshMM0ZZ1ER_P4gUoYqYdt0QI0T9-9WWSE7OlRWodz_tNrQpty_pbl3qDbqLI_sXGqVGY1v4rf61YjjQ6MW66xsaS8ddsjpymzdGIe_vbnyWJ-cmXJrxzSaRorRVnh_OQapRrGfm-xxfn0B1vEn_3-zro1urljaJbsrDiQZjXKFbr5qzuPzgid_oevr17vaxhdBmdni_b8_DNM_uCZ-Gs3w17hcf7uRqMnHo4nJ_BEctYwRZsy2GZT7saSlR-Ms5igDDZ2NyoLHkt7dsPBeg9EBKPxaPixG4_m4on4azcTbXVoZqwOQ-zexMIM8bPIbp0ocRYDshJNg4agq9CA6I1-9fbmlrjQBQv93yKDd0ANopzBxeQc_kgY4hSKhn2xyP5WoUOQFgkEOAzBmCZjYo3pN8hKMEMquc7FfNxv0cJIAmXCrujpnGO-rMAavYVaNASN8BVxTUiRNEqkRUvKINGZQy24cmnFKZS1ewlqCVvbgnAI2to1r-eqXWAlNsq2jjO7oj6Fd87yAqVxssiSzqoOcMoTdvrsRnkI2zyGU1zLOWhA8R6I4R1oUaDm27OpYZrAmuUgWzeDz8_Pa2Izs5yhpu92k99qnIEovdpgv44rTr9kkXEmi_9MPC0WRjTNpLHkT56nw57nezf5PlryNPpaSZylOh6kjz2-aoWTweMWJJZKYoCPr3pXC6059RMIrW2HEpQ54HYuPWI_lYJrDVe7pXUIwmx7907gtYcNOrVUqbYVKBwfZ9cMWyNBeM8xFmcbp0ypGqEZIgFHEVwvQcCL84sABqhRGAIzYCjUaHDCTOD1EgQU7SqAhEv-GuF5KEpsKugEgQmrS4eMsec5lBWW64ieUHf3EPOBWOnz2iitwbsIiBeXBwGRlhyHQ1gQwbAz85RHIyqCUbc_dP7kFN7_819sqElL6J5GQW81xgEHZ_-_7Ux0AhrZWGV85Coh2YTsoPgLyhPq5RFY9LI2QqtAbKPPrdxG3uOEIR3Gv7-9vQJlvOXcJbQGaxItCpVyAv-PWhXMWTBwGGVmrMDa2I5AFMyLyAvfUkhQFMT_cPP2p5zNlxZwNkfac-gTjowFZ68OPeHIYdFxV6Yl0Zl93kuxHJ35D8tEsMJaBF_Sppw0WpR4UtbyA-Ja25Uqh0yecqAofSv0kFXjogNuG_OTfu9ApkPS7ViqoJB0g2FDVIum0RzUfcl0zGLyvg2LcmLU-crZdlWleO5ZTUQXmlBCY-mJrEPFBuA3W8Tew0SuHmRN4Ko_DoUkWDpbg4BAbmKt7kuO4tS23dHppItYcqAKkNhou-VVpG13GCJBszwV7HxUwgEIf-_tmJDTM5IdIRmAMy4JaXVgE_s8pV_-2eXD5QMtGbOScS367tVtdMBs88VQi_YkPQBu1PA4anl-kQU2G-72Sx-ZAbERnqezGGMRtWITfySfPo3bhJV9DhLrNzcokeEcyTQ7IPWgDSSDk0HECzUo1gQ3f3kzgT2MV5GneEaBrYUyYIvfsPQBjyv0BIUo18yhlHwZJL_-6ebV9e3QuYUsxhazBgrbGincNjATvsRHQCnv_ZrvaOYhaI3BEkjrI8CKpHbgtGOMpFvcIywvHgVJOu4R1sIL9mCyc0xECjfyt3cnyhA6H_BhW1_YuyN5jQ3go-aFkCtkzWPofvLlly8uFtlDGAnampj-RrQ4BnpMOdRYQyG5ERp_BFAJ3uOM9Jst-n5d-JhRGKYjXsK1kkKaCmkzUsZKMLwIPj2_gGILxnbxlSUx_0aVqe_lFNc2wAWRCazQa-p5D-43WakUc1hQf6lW-4jUcX4N7zMjF8DtiBY33FY3aBuNUCtKvYOvcBvSaYwgvyvjh7H88aCNXdIIikODeRKcenoYjUlS6rH2WqyDRTmt7xusvc7rIXFH453CXSBcHoimtPfI2bwh72NjQNmjyqR4GlSphVsPhS8-631AVA56HI_LtGSRxfeB-C9Nh6NibDoMNpjBZR6gZJfLGeBd8-GhGfy3d4NRfB7qXId03fe7PdsMhfjRhpWx_Ut6zl1kN4z09HCZg1TLJTqOvv4x6WGU940mdwHpyW621zaG4EkHxG4Eubmk_ZfJllAOL2AeRf2coEbjheYWl2Up40XpY9DfezlKD5HxWK9qfDl-YVRmY_Um9F3AFTYwrtTLDy-Ph_oT2pT3A_Mj-cuT7S5TjGm6zVQKpbePhs4jLdEhhrNHvHubvfsQHrNb_HiV4jX3KHga5lh4kqsU20YQS3jtYcU8hbwTQYGhU-2T7nHCMgYCrVVD45YHtNgyNzWSPecU0j25IJXD8KTLBSfS3NgLM5nRZEcyEwQCxQ6vNKHH4s4elq3WQCXjijvh1IfAbSBNlXW-bIe3cGGoC9363jvt_wiCwwvYIa5z-VFc5-bVm1ff3sL7P_8D312__esVfPN3kGKH2K_On4JUHiUex5QsYJHxJdiUOXw6ufgRnO04rRoLyki84542PTT8IvyxpLoS7eq-_s1Xl6zrRug2pNkXky-BRrk1--PnP_4bAAD__w){ .md-button }

??? abstract "The source — `06-in-the-wild/07-architecture-canvas.dgm`"

    ```dgm
    %% A persistent architecture canvas for an agent session, asked for in
    %% openai/codex#35100 (https://github.com/openai/codex/issues/35100): "keep
    %% one architecture view pinned that updates, highlighting the component
    %% being discussed", instead of regenerating a disconnected Mermaid diagram
    %% every turn. The diagram below is drawn once — it is the same unchanged
    %% Mermaid the agent would commit to docs/ — and each scenario is one
    %% conversation with that codebase. Every step is one turn: the component
    %% under discussion is lit, the file that implements it is the note, and
    %% everything else recedes. Nothing is redrawn, so nothing moves between
    %% turns, and the second scenario reuses the first's layout exactly.
    %% ---
    %% The `cinegram mcp` server exposes exactly these operations (write the
    %% .dgm, lint it, render it, report what it animates), so the agent can be
    %% the one writing the scenario while the human asks the questions.
    flowchart LR
      client[Browser / CLI]

      subgraph api[api/]
        router[Router]
        auth[Auth middleware]
        orders[orders handler]
        reports[reports handler]
      end

      subgraph core[core/]
        svc[OrderService]
        pricing[PricingRules]
        repo[OrderRepository]
      end

      subgraph infra[infra/]
        db[(Postgres)]
        cache[(Redis)]
        queue[/Job queue/]
        worker[reconcile worker]
        ledger[External ledger API]
      end

      client --> router
      router --> auth
      auth --> orders
      auth --> reports
      orders --> svc
      svc --> pricing
      svc --> repo
      repo --> db
      pricing --> cache
      svc --> queue
      queue --> worker
      worker --> ledger
      worker --> repo
      reports --> repo

    scenario "Trace: what happens when a client POSTs an order?" { speed: 1.0 }

      step enter "Where does a request enter the codebase?" {
        desc: "Every request lands in the router, which only maps paths to handlers. Nothing business-related lives here; if you are looking for behaviour, this is the wrong file."
        dim auth, orders, reports, core, infra
        flow client -> router { label: "POST /orders", dur: 600ms }
        highlight router { style: active }
        note router "api/router.ts\napp.post('/orders', auth, ordersHandler)" { side: below }
      }

      step guard "Who decides whether the caller is allowed in?" {
        desc: "The auth middleware runs before any handler. It verifies the bearer token and attaches the principal to the request; a 401 here means no handler ever ran. If a bug looks like 'the order was never created', check this first."
        dim orders, reports, core, infra
        flow router -> auth { dur: 450ms }
        highlight auth { style: active }
        note auth "api/middleware/auth.ts\nverifyJwt() → req.user" { side: below }
      }

      step handler "Which handler owns the endpoint, and what does it do itself?" {
        desc: "The handler validates the body and translates HTTP into a call on the service. Deliberately thin: it knows about status codes and JSON, not about prices."
        dim reports, core, infra
        flow auth -> orders { dur: 450ms }
        highlight orders { style: active }
        note orders "api/handlers/orders.ts\nzod schema → svc.place(cmd)" { side: below }
      }

      step logic "Where is the actual business logic?" {
        desc: "OrderService is the component you were asking about. It applies pricing rules, persists the order through the repository, and enqueues a reconciliation job — in that order. Pricing reads from a Redis cache, which is why the first order after a deploy is slow."
        dim reports, queue, worker, ledger
        seq {
          flow orders -> svc { dur: 400ms }
          flow svc -> pricing { dur: 350ms }
          flow pricing -> cache { label: "GET rules:v7", dur: 350ms }
        }
        highlight svc { style: active }
        note svc "core/order_service.ts\nplace(): price → save → enqueue" { side: below }
      }

      step persist "Where does the write happen?" {
        desc: "The repository is the only code that speaks SQL. OrderService hands it a domain object and gets back an id; the INSERT and the transaction boundary are here."
        dim reports, queue, worker, ledger, pricing, cache
        seq {
          flow svc -> repo { dur: 400ms }
          flow repo -> db { label: "INSERT orders", dur: 450ms }
        }
        highlight repo { style: active }
        note repo "core/order_repository.ts\nwithTx(insert → outbox)" { side: below }
        set db { badge: "order #8841" }
      }

      step async "What happens after the response is sent?" {
        desc: "The service enqueues a job, and that is where the request ends — the client has its 201 by now. The worker picks the job up later, talks to the external ledger, and writes the result back through the same repository. This is the part people miss when they read only the handler."
        dim reports, pricing, cache
        seq {
          flow svc -> queue { label: "reconcile(#8841)", dur: 450ms }
          flow queue -> worker { dur: 450ms }
          flow worker -> ledger { label: "POST /entries", dur: 500ms }
          flow ledger -> worker { dur: 400ms, style: response }
          flow worker -> repo { label: "mark reconciled", dur: 450ms }
        }
        highlight worker { style: active }
        note worker "infra/workers/reconcile.ts\nretries: 5, backoff: exp" { side: below }
        set db { badge: "#8841 reconciled" }
      }

    scenario "Trace: why is the reports endpoint slow?" { speed: 1.0 }

      step same_canvas "Same diagram, different question" {
        desc: "Nothing was redrawn: this is the same canvas the previous conversation used, so the team's mental map is intact. The reports handler is lit this time; everything involved in placing an order recedes."
        dim orders, svc, pricing, cache, queue, worker, ledger
        seq {
          flow client -> router { label: "GET /reports/daily", dur: 500ms }
          flow router -> auth { dur: 350ms }
          flow auth -> reports { dur: 350ms }
        }
        highlight reports { style: active }
        note reports "api/handlers/reports.ts" { side: below }
      }

      step bypass "It goes straight to the repository" {
        desc: "The reports handler skips the service layer and queries the repository directly — which means it also skips the cache, and every call is a full scan over orders. That shortcut is the answer to the question."
        dim orders, svc, pricing, cache, queue, worker, ledger
        seq {
          flow reports -> repo { dur: 450ms }
          flow repo -> db { label: "SELECT … GROUP BY day", dur: 900ms }
        }
        highlight repo, db { style: active }
        note db "seq scan, 2.1M rows\nno index on created_at" { side: below }
        gauge db { label: "p95", value: "4.8 s" }
      }
    ```

← [pytest tests/ — one session, hook by hook](../06-in-the-wild/06-pytest-session-hooks.md)  
→ [map applies the function to every element, in order](../06-in-the-wild/08-scala-list-map.md)
