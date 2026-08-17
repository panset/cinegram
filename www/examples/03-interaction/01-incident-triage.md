<!-- Generated from 03-interaction/01-incident-triage.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# the cascade

A service map for the twenty minutes when it matters.

<div class="cinegram" data-cinegram="03-interaction/01-incident-triage" data-height="900"></div>

[Edit in the playground](../../playground/#doc=nFfbbttGE36VAX8YfwtIsuykSay7JA6KFEGSJilyIeZiyB2RCy932d2hZcIw0IfoE_ZJitmlKMqHIG4uInJ2Z4bzfXPydXaZrU5mGVn2fbbKlk_m2jJ5LFk7e7w8mWtbakWW5-w1VrRQVZPNMteSfcT1jTYUstX6OmsfocXZKmO64myWqWyVHR3BSwjkL3VJ0GALG-eBawLekuUeGm07pgDbmixohgaZyYdFbo-OYD6fx983l-T70YrR9iKA6xjYiZ4n6F0HW9cZBVhyh8b0UDmwdMXwz19_g-YACkNdOPQqWnQ-Cn1nC-cu4qWCSuwCAYLSWHlsoHQ2dIZJgeq8thWghV3QoANgNGXxUlcooEDo_AZLmoF1HC8b0wX28XABX2qCUJJFrx14ag32IUJRYihREQQXDWKMUIJzlqBxjbijq9agtkE-m8xmkduNcduyRs_w7lNuAbpAPqz_kP-_yTupitZvVEVwDK_P33_LrUhDV1Qe2xpK52n92nmCzwnXpAWArV6__PgWfkWmLfaD1Hkl5j_En0Gm7SVZdr5fv909Jc9W3XKmqA3rc2rJKrKlHn2pYv1TMgnnr34ehH921NH6N1fA7_J0aLLFivz6g52XaAx8lLchshg_zOd5t1w-oRj-Doa9FFstQmz1XpZCuyMeo5OTdGd_qAqRjlf2B_Hj79OIHy5fuqsiuJZrpdHlRXSc_u2ud95AntXMbVgdH9MVNq2hRema4zGRwzG2el4lmvIMrsFgQWYFeTbhb5L4GdzsXQ7f90MuhzIJxwNUB74G_oY7h04mAD0mrj30B57GLHsoJlXAf4ExRTVXxb2Bnb96yF2k9AfdOSsJe-jga-2kjTgLw9lNbm8kRcYukWeT_hCVQ0ukVnCyWMLNUGRMLVjnGxTfXzxuNroUu0kmWkOxUSjF7SsMZLSlBXx1nmtAKAgZkGMzCiwtJbj0UmNL4DZQExque-C9eel3pSeyUNBG-gjanmtpk5WThu6drRZ5lnxLs9pV6KRAYVKXU2ROTi_AtyHPZtJ6V3C2XDYhQT_YiqV6UMBT_bKm8sJ1PBr4ZW9gClsQU3kmnXkwopCxwDDgEIB1IyFFW7eBFLWN9oHBExoIurJoZoBWySSLDMQxQN47Dx6ZhmkkZwaZbNkv4L1LqNUYYIPakIKeeAbbWpe13KQrLNn0sK37QZcw9DIfGh3CHmJXdiF22glMuy40tq0pSu3ZGZwunt6CeQalM87Lhf-ps-fPl892WQ9QYVfdY0UMXKLpSN6TxZ2GdRwV8qx01lJcHqB1zuS5DcidoKJSZmtFKyhIPvsOUZ7Y95NmQ-w1hYR1gxckcxG2zge6S9OndBmwkgHKgDA6FrjSTOoBlQLjUAmwcUGJrHCNadYbT6j6eFI4ZkOWygsZ6jrIuciHUS2P45oQ2LUBCorrQxGXlprGLDusj--SlRC4ejKS9SKRFRi5C6uYOo-h6cXiZEJTIN6XkZikFSiqPCpSDyfEQSm1AtBQFcNQ-r_gEsXCOGy0MeEuP_u2rsPYaITYwNoY6KwnLGssDK2G_I9DVkFBtbYKhpzQFjD54Zp6aV2eFvDW7ijm3QIZZnGxihcUeAqu87K0ibAwKOWMSne3K0sWpj1YqWEVqKqI5iSVZ9_Frzx9djopqDu9bD8xJ9yleEfmnz_MfCw3sZdnQ3wTAiZlhoW7pLskykQbKEzDLbZyt9ncM0VSLbBuKD6gIc_SDmm31MoyrVMGENRSBhvvmjRX-qZl1yzgdVp_7H67l7W3jX8LTHb2pCmr_ncrJn3zBLjP7z5A0XkLJ0-vRvie3elyt0g5KIaR4oiNRx0OSXZb-7Ctm9xmN99u_g0AAP__){ .md-button }

??? abstract "The source — `03-interaction/01-incident-triage.dgm`"

    ```dgm
    %% A service map for the twenty minutes when it matters.
    %% ---
    %% Every service links out to where you would actually go next — its dashboard
    %% or its runbook — because a diagram consulted during an incident is a
    %% navigation surface, not an illustration. The scenario replays the cascade so
    %% a link to one moment explains itself.
    flowchart LR
      users[Users]
      edge[Edge / CDN]

      subgraph core[Core Services]
        api[API Gateway]
        orders[Orders]
        inventory[Inventory]
      end

      subgraph deps[Dependencies]
        db[(Orders DB)]
        queue[Job Queue]
      end

      pager[On-call Pager]

      users --> edge
      edge --> api
      api --> orders
      api --> inventory
      orders --> db
      inventory --> queue
      orders --> pager

    interact {
      click api       -> url "https://example.com/dashboards/api-gateway" { label: "API Gateway dashboard" }
      click orders    -> url "https://example.com/runbooks/orders" { label: "Orders runbook" }
      click inventory -> url "https://example.com/dashboards/inventory" { label: "Inventory dashboard" }
      click db        -> url "https://example.com/dashboards/orders-db" { label: "Orders DB dashboard" }
      click pager     -> url "https://example.com/oncall" { label: "Who is on call" }
    }

    scenario "the cascade" { speed: 1.0 }

      step normal "Traffic is normal" {
        desc: "Baseline. Worth a beat at the start so the shape of healthy traffic is on screen before anything goes wrong."
        flow users -> edge -> api { label: "12k rps", dur: 900ms }
        flow api -> orders { label: "checkout", dur: 500ms }
      }

      step slow "The orders database starts timing out" {
        desc: "The first real signal, and it is not an error rate — it is latency. Nothing has failed yet, which is exactly why it is easy to miss."
        focus deps
        flow orders -> db { label: "p99 2.4s", dur: 900ms, color: "#d97706" }
        gauge db { label: "p99", value: "2.4s" }
        note db "connection pool\nsaturated" { side: below }
      }

      step retry "Orders retries, and makes it worse" {
        desc: "Retries against a saturated dependency add load to the thing that is already the bottleneck. This is the moment the incident stops being about the database."
        flow orders -> db { label: "retry x3", dur: 800ms, status: fail }
        gauge db { label: "p99", value: "8.1s" }
        set orders { state: degraded, color: "#d97706" }
      }

      step spread "The gateway's thread pool fills" {
        desc: "Inventory is healthy and still unreachable: it is queued behind Orders in a pool they share. Independent services, one shared resource, one blast radius."
        focus core
        set api { badge: "saturated", state: degraded, color: "#dc2626" }
        flow api -> inventory { label: "queued", dur: 700ms, status: fail }
        note api "shared thread pool" { side: above }
      }

      step page "The pager goes off" {
        desc: "By the time the alert fires the cause is three hops from the symptom. Click any service to open its dashboard from here."
        flow orders -> pager { label: "SLO burn 14x", dur: 600ms, color: "#dc2626" }
        set orders { badge: "page raised", state: down, color: "#dc2626" }
      }
    ```

← [sign in with an external IdP](../02-storytelling/04-oidc-login.md)  
→ [a priced order request](../03-interaction/02-layered-arch.md)
