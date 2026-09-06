<!-- Generated from 06-in-the-wild/09-animate-the-flow.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# order placed: payment approved

A single POST /orders call unwinds through a service layer — client, controller, service, payment gateway, repository — one message at a time, with the reader holding the scrubber instead of a playback timer deciding the pace. It answers mermaid-js/zenuml-core #268, "animate the diagram": https://github.com/mermaid-js/zenuml-core/issues/268

<div class="cinegram" data-cinegram="06-in-the-wild/09-animate-the-flow" data-height="1170"></div>

[Edit in the playground](../../playground/#doc=rFhtb9vIEf4rAxZB7gDJb73mrgLyIXByVwNtYsQB-qEq6tFyJO55ucvsLMXoggD9Ef2F_SXF7AtNyXLSNP1iwdzZ4c4zM888y4_Vtlqczyqywe-qRXX2bK7tPDQ0H7SpT8_-OEerWwwUn62NG07qTVvNKteR_Rr7tTbE1eJvH6vua7aFalEF-hCqWVVXi-rJE3gBrO3GEFy_uXkHp87X5BkUGgO9HbStGULjXb9pAIHJb7UiMLgjD__-579AGU02zJb2yRNQzgbvjCE_K5Yz6HDXkg2wwUAD7mbgqXOsg_O76MBZgpaYcUOAARCCbin5G3RoIDQEnrAmD40ztbab-IiV71cr8qAtB8Ia3BoQOoO7Faq76MRDTUrLjuhNdnWo6ASuAqDlQeJsybeo6_mvfPob2b41c-U8we8unv00g2WVUYx7a40bj-2yWkR3TQgdL05PNzo0_epEufb0uLNTzdwTn148-ylunM_n8fddPg9oBrJBezK7SbRPeSH_-bhuHWAfnIQHHpWAgE2JOnq7rYnVLaxoLefXARpkWBHZ6G4GaOuMWUTQAdodtNoYzaScLDZu4OiKPqAK-SwcUvgY9ox12xlNDB7liLJuAS1oG8h3zmDQzibQ3YC-Bh1OHsReaklOlsIe6yJ4rOWtnggYW5rX2pMSp6VUGNyW_JhXo9ckWEgxxQqejW5z3YGKR5Ssp7oiHY_u1hAGFx15SkE5m0LHVhKkU51PQVHIBLctbxa3QB80B4a1k5p3QFvyu5QQ6bpbiBlk3DEMjVYNoPdukPy0hJan1WsIt6W6lWs7bQTZdLRNT8yw9q6FCJJ29iSCGIOdpyzdbrDf0O14_oyvnL4jK30wK4BZKafGec1Uz9ID5exa-5bquEFajQGVd8wpKOBAHWgLKwoDkZVwoztMdQUG4zvKusAKg9eBGDhoY2Ihckq13jSlB2dgXchFvDJo7ySJJ0vL9L4nq-hlbjsL0KEPWukObYBLQIbLSD4PloI3svpGsLkcOenQ7GarRqubhNWhyTXuxOQ6UdgvqZIOjd5S50ZHb8ciXloxvJwv-7Oz31P6Kydb7BGt2KAKeisJlOW4KXizt-9mqxbCbYriS75T6MP3e1tvtkr-v9mqvY1ynAUwbum73BZSKj0_v371-uXV61-iEzGaP3xf3HBVP__hhx_Pj_m-xt0CVIN-k51n2xlg63p7cMDrBByaME4E7DrvtlTLcxCDI4coNrNYsJeupucvztOG47H2XY3h8EA56Ms3r3--evuXVy-_Tx4eC_zu_gVH0ndxdg6XnjBQDXvviV7JMI0R1qSMtp-PsNjIYER29rm23K_XWklp_2Pd25r_13h_fnH1528M9oezi1L-8Jbe99qnaMjGn5oOEzx5kmsyVvO-6y-BOHGSemJpWZFFrx0sq2ie2qFePKymCj4CdyRr5ydn8Ck1YmQvL5zCAZaVcGeSLsD9qtUhEVN0LR4SKDJTF_vW0rwMCJaGZB0HjXDfODimMig6tUUwaTHQwodRo8hU5Y7wjuFP795dn8ArIdrQCIvWbrAcPGEbR5TskjIBZ80uEjIwESfJo2UshsbVUbbNwMZ1HMN1q19JhZNllaKSwQSXkFOS6PIj1L1fwLOzs5YFsmjnVM8jKU1xrDV3GFRToBkDhpoMycRlkRiTKfQIppONjqLMWfWsrQw74zZagQ5MZh1Hkg4xPJ4Q4cGom-UZK9Ddp_OpjF5nkkhQO2UoDdI4mdMc1SwiK3nCFhB6qwMEwa516o7B9YfwCWgFQZkkjwOY22CKX0ee9ViHZVB7Us7Xk5MXNRdcr2JRtM7S7jiUaYPmOHKDDHiGTPKjm8m7NgI32t0QUbCEUi9KpJrt2xUlMYOgPHIzTvRoW3RV6bt4U0gDXhRMrEmJZEseV4Ygyp2pRk8jA6xbuTrLMqVkYIiIOknD6ntoYkclLV_Cu6qzFDEYyKc3l5aKWRnkFAq9343ZYnpf4Mqpk2yVzMXZnVP3B0ndDETWwXlJYd4T7Y6kO-_hsDO0AE_cOct06CT_RHmWdxtckVmMZBYF3LKawRZNT_I8K7ZltV9McpDDasqA7hcT8h3v6d_gimGSsL5-WEh_FWGMElpMnSs9JsmY7dVP6kTXh9LnB9e8_a4SBmxcl5JFuR0LO0UazKUTPXmniPkpSx9D8D0HWLne1uh3sfqkZNaojdRLHpq8d7jMJaHwy37rTvMvAi-n8seHnZtH2hTrMmUy2gXc_DgdI8F8vE3LBk-h91ZasCjx3-KlCZSr6eENBItMgMH1poYGtwQ9Z7E-uZ1C7XHgfB40Ea-yFTlqcvQtl7uSXJBZO5tEuKwydehl8MYM7yMnaD3GeYc1_1XFfn8ZOaz3I-SZLyoH9d6iv5tS53id-W_pEvLVNn5-AOuGeMEoinHEOebk_qo6u38oxDXnBjsqczjO6nKL7hmUM31rpT7shni_RYQmEbjvOucDkN1oS-Rh46mLt0sY4q0NVM_BtTLepcGHhvIFvIQ0SBPmDzbfSIIX_w8SvPgWEpwm8Ys0mF5cPxiqAigD90pYJU2UvitpUXfH6-N-e-rTe4B1Gn9ZDsaJI3LI4C59c7AuCThcRXacUGNkq_EbR2JQE9WftpFFNY_ojfU2kUgH0i8eqQ_KtfQ4wT0Qd4d9-rjIU67tDAV6qNXKh5JM_QmKLyq8eyAzuE9juL0JJX6S62KPJt4ORiTuvw-leZG_Sma1XG6Ywp0r9OA6slQXBhWqFSUdv4ah8YQiOYwT5lyl3pR-L9ijagqHx5g-o_kuPyeZM5TVp79_-k8AAAD__w){ .md-button }

??? abstract "The source — `06-in-the-wild/09-animate-the-flow.dgm`"

    ```dgm
    %% A single POST /orders call unwinds through a service layer — client,
    %% controller, service, payment gateway, repository — one message at a time,
    %% with the reader holding the scrubber instead of a playback timer deciding
    %% the pace. It answers mermaid-js/zenuml-core #268, "animate the diagram":
    %% https://github.com/mermaid-js/zenuml-core/issues/268
    %% ---
    %% The pace is entirely the reader's: there is no autoplay racing ahead of a
    %% `desc` before it has been read, and scrubbing to any millisecond shows
    %% exactly the state that millisecond implies rather than an interpolation
    %% toward it.
    %% ---
    %% The service and the repository trade three same-direction messages over
    %% the life of one order, and the gateway can answer with either of two
    %% replies on the same pair — exactly the case `msg:` exists for, so every
    %% `flow` here says which arrow it means instead of leaving the compiler to
    %% guess from position. The order-state `gauge` on the service — pending,
    %% then authorised, then confirmed — holds across every step in between, so
    %% a scrub landing between two writes still reads the right answer, not a
    %% blank one.
    sequenceDiagram
      participant C as Client
      participant Ctrl as OrderController
      participant Svc as OrderService
      participant Pay as PaymentGateway
      participant Repo as OrderRepository

      C->>Ctrl: POST /orders
      activate Ctrl
      Ctrl->>Svc: placeOrder(cart)
      activate Svc
      Svc->>Repo: save(order, status=PENDING)
      Repo-->>Svc: orderId=4471
      Svc->>Pay: charge(orderId=4471, amount)
      activate Pay
      alt payment approved
        Pay-->>Svc: approved, authCode=A1
        Svc->>Repo: update(orderId=4471, status=CONFIRMED)
        Repo-->>Svc: ok
        Svc-->>Ctrl: 201 Created (orderId=4471)
      else payment declined
        Pay-->>Svc: declined, reason=insufficient_funds
        Svc->>Repo: update(orderId=4471, status=FAILED)
        Repo-->>Svc: ok
        Svc-->>Ctrl: 402 Payment Required
      end
      deactivate Pay
      deactivate Svc
      Ctrl-->>C: 201 Created (orderId=4471)
      deactivate Ctrl

    scenario "order placed: payment approved" { speed: 1.0 }

      step request "The client submits the order" {
        desc: "The client POSTs a new order and lands on the controller, the one layer in this stack that speaks HTTP. Everything downstream of this line only ever sees a plain method call, never a request object."
        flow C -> Ctrl { dur: 600ms }
        focus Ctrl
      }

      step dispatch "The controller delegates to the service" {
        desc: "The controller does no business logic itself — it calls placeOrder on the service, which owns the order's whole lifecycle from here on. This is the seam a unit test mocks out."
        flow Ctrl -> Svc { dur: 600ms }
        focus Svc
      }

      step persist "The service records the order before touching money" {
        desc: "The order is written as PENDING before the service goes anywhere near a card number, so a crash between here and the payment call still leaves a recoverable row instead of a charge nobody can account for. save() hands back the orderId every later call in this flow will carry."
        seq {
          flow Svc -> Repo { dur: 500ms, msg: 1 }
          flow Repo -> Svc { dur: 500ms, style: response, msg: 1 }
        }
        gauge Svc { label: "order state", value: "pending" }
        focus Repo
      }

      step charge "The service asks the gateway to charge the card" {
        desc: "With a durable order on hand, the service calls out to the payment gateway. This is the one hop in the whole request that leaves the process's own trust boundary and can fail for reasons the service does not control."
        flow Svc -> Pay { dur: 700ms }
        focus Pay
      }

      step approved "The gateway approves the charge" {
        desc: "The gateway returns an authorization code on the same pair a decline would have used — the diagram draws approval and decline as two arms of one decision, not two separate calls."
        flow Pay -> Svc { dur: 600ms, msg: 1 }
        gauge Svc { label: "order state", value: "authorised" }
        focus Svc
      }

      step confirm "The service marks the order confirmed" {
        desc: "The order is written a second time, now as CONFIRMED — the same repository, the same save-shaped call, only the status column changes. This is the row a support engineer greps for when a customer asks whether the order went through."
        seq {
          flow Svc -> Repo { dur: 500ms, msg: 2 }
          flow Repo -> Svc { dur: 500ms, style: response, msg: 2 }
        }
        gauge Svc { label: "order state", value: "confirmed" }
        focus Repo
      }

      step respond "The service reports success back up the stack" {
        desc: "The service returns the order id the client will display, and nothing about the payment or repository calls leaks into this response — the controller only ever sees the outcome."
        flow Svc -> Ctrl { dur: 600ms, msg: 1 }
        focus Ctrl
      }

      step complete "The controller replies to the client" {
        desc: "The controller turns the service's result into the actual 201 response and the request unwinds. Every activation bar opened on the way down has already closed by the time this reaches the client."
        flow Ctrl -> C { dur: 600ms }
        focus C
      }
    ```

← [two questions, answered in order](../06-in-the-wild/08-overlapping-activations.md)  
→ [replay a finished run](../06-in-the-wild/10-agent-run-replay.md)
