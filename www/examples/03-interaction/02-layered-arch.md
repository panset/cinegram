<!-- Generated from 03-interaction/02-layered-arch.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# a priced order request

A layered service, walked one layer at a time.

<div class="cinegram" data-cinegram="03-interaction/02-layered-arch" data-height="900"></div>

[Edit in the playground](../../playground/#doc=jFfRjtu4Ff2VCwEDZAHbmeyiffCbNwmKAEGSZlwUqLTAXpPXItc0qZLUaN3BAP2IfmG_pLiXki3PzKZ5kiWSh-Q59xzSD9V9tX6zqMjneKrW1e1PS-szRVTZBv_69selwxNF0kuMyqx0e6wWVejIf2_fvXWUqnX9UHXfOyRX6yrT77laVLpaVzc3sIGxGySK91bRAgZ0B9IQPJU2wAwI2R5p1fibG1gul_LcgLbYRjxCNjZBsv8isAl6Hwk17hwBOseDgxdYY5XhDtkQdMH6vIZf90H16VeBy3igBDjN6TU4ygnonuIpG-tbIJcIIinStIAUgFAZSJk6MJiAfkeV3YnXXfBkTA7gQjgA5hVsDYGKIaWl6nPmVsVLiz4BRkZGB7s-8wJP8sUHeRG4SP_sKWXoMBuZXXqljDHDPjhNGnDAE-zIWK8BQRnbrRq_d2FQhnttf248QOp3bcTOAOmW6ve6JfjIO_6FGwGU9vXbd5_Gt3aoN18-wF8w04An-UheN_4KCLuu3nSdswpZ-iu4EDXFVH-WB9wVice2LlplfVt_Kc-r1uez6HBE6-t38riaI_aO0hnlK7-NLTtMB8r1z_KATdtGajH_4QyYsX6HGa_Qu7Z-9SWk3Ea6--vHHyaaUBmqX30lbdMPf4AnUtdvrwR_Owo-wmCfja83fTaf4DXw8x8TcZlcvY3Iezp31jbXrzb8gI-hfTKv0h6Wy6a_vf2JoB34UztcvhQl-Gv5dWkZdeCm8eelTbh9aVChllvKrxncy0jC2NNFyf5fguftvwQuHPB-p6yBB-52c1PMNdlJTDs6WVNHXicIXkztwy7oEwzocwIdcfCQTQx9a4rVBOxotXYEYS9hMbfe6GJjO0h4SmDCAMe-BMtoPR6inFWHlcjCv8RscCaV7tnqUh-Nf-TtJEUeow3QVCjccQAyJ9PkTQUPkDoivYY3q1t4HIuN0wdjtPcETbWdLdYhbxolQGR6RiiFpCmpNTTV-0u0TZ0mzgIl-O-__wOZ4tF68fUCYui5iBcQMRM4e7RS09zPYNeRZwr2IRKgPzF5p9BHUEFzHfm0aqoyv4RuWVN5d2Eo5XuuXngAhztyvMwvn--28Hqs32oBuo9r-PPt7TExCwzgQ5ZBTbX9eHdeM-mm8ZelQjgUEq2mNUTbmlzGz6kMURlKWUYVPnGWbIWcWZ_0nNQtn0WlowohallJktILg0_gQzEV82Nz4o8r-CBcwZ7LZ9cn6yklcKG1CgxFWoDN5YxJCjvSsI_hKJqVVBzn25ELwxOWsetmJLP3rvJgzrOKhJkkqs80_2lOs0BMTr0OjzmOlG_J3JdxRK4Rp6l8uGy5MMNb_v9SaVJWTypd8VBaXtRmTDTIBjMc8UDl7oDnfaRTynTkCjcUuZsHhLdf__ZuanFstSIKFrOfuLphsNmEPsPBh4GBBkMCgWdHBn7ZYVYGfgs7UOgcCcDxiWZlMzPOz2E6T-U55VykJ9A2qdD7nL5fvjFfZ1CR-CLGnvGUvqFfWUNTdX0k2Pdern1piszxeLiI6Gifr8aPMzeV9fcYrcQx-X2IHH3fVwEdxWRTnkqAT-5SACmHSGnM7tS7_LwUPnt3kg5lhNSDQg87gjRwmGnYs2A-iIoMvsNEkNizOgxe1riCjR8TFHfhntinAuXxKJfJLPdQtq6kPV_ShNmnimPGmUjTqXc-UecCffh09_7rtgj5DaGf1oycv1d2lw9i1m_I3LXQVMxTYMKGaDPPelZGIue5MoMJbrJm-V2urcSOCXJneabI3202PKewKbSQXlwkSqDCkctGHaY7vZw8owkhGezkxB6Y_9_6lMdDifR06Z_-LnDUysWZtNj2IsYLB9GTvPxG7vEflvNpPRL65scLo4-Nrx5_efxfAAAA__8){ .md-button }

??? abstract "The source — `03-interaction/02-layered-arch.dgm`"

    ```dgm
    %% A layered service, walked one layer at a time.
    %% ---
    %% A diagram this size is unreadable all at once, which is the point: `focus`
    %% takes a layer and lets everything else recede, so each step has exactly one
    %% thing to look at. The cross-cutting concerns are real but they are not the
    %% request path, so they start folded away behind a chip.
    flowchart TB
      subgraph edge[Edge Layer]
        cdn[CDN]
        gw[API Gateway]
      end

      subgraph app[Application Layer]
        orders[Orders Service]
        pricing[Pricing Service]
      end

      subgraph domain[Domain Layer]
        rules[Pricing Rules]
        basket[Basket Aggregate]
      end

      subgraph data[Data Layer]
        pg[(PostgreSQL)]
        cache[(Redis)]
      end

      subgraph cross[Cross-cutting Concerns]
        authn[AuthN / AuthZ]
        otel[Tracing]
        audit[(Audit Log)]
      end

      cdn --> gw
      gw --> orders
      orders --> pricing
      pricing --> rules
      orders --> basket
      basket --> pg
      pricing --> cache
      gw --> authn
      orders --> otel
      basket --> audit

    interact {
      %% The concerns every layer depends on and nobody wants drawn through the
      %% middle of the request path. The chip says how much is behind the click.
      click edge -> reveal cross
    }

    scenario "a priced order request" { speed: 1.0 }

      step arrive "The request lands at the edge" {
        desc: "Everything the edge layer does — termination, routing, rate limiting — happens before any of your code runs."
        focus edge
        flow cdn -> gw { label: "POST /orders", dur: 600ms }
        note gw "TLS terminated\nrate limit ok" { side: right }
      }

      step orchestrate "The application layer orchestrates" {
        desc: "This layer coordinates and owns no rules of its own. If you find business logic here, it has escaped from the domain layer below."
        focus app
        flow gw -> orders { label: "createOrder", dur: 500ms }
        flow orders -> pricing { label: "priceBasket", dur: 500ms }
        note orders "no business rules here" { side: right }
      }

      step decide "The domain layer decides" {
        desc: "The rules that make this a pricing system rather than a CRUD system live here, and they run without knowing whether a request or a batch job called them."
        focus domain
        flow pricing -> rules { label: "apply discounts", dur: 500ms }
        flow orders -> basket { label: "reserve lines", dur: 500ms }
        note rules "pure functions of the basket" { side: left }
        note basket "invariants enforced here" { side: right }
      }

      step persist "The data layer stores the result" {
        desc: "Only the layer that can be swapped for another database sits down here. Anything above it that names a table has crossed a line."
        focus data
        flow basket -> pg { label: "INSERT order", dur: 500ms }
        flow pricing -> cache { label: "cache price", dur: 500ms }
        note pg "the only writer" { side: below }
      }

      step whole "The whole path, end to end" {
        desc: "With nothing focused, the layers come back at once — and the shape of what just happened is the diagram you started with."
        flow cdn -> gw -> orders -> pricing { label: "one request", dur: 1200ms }
      }
    ```

← [the cascade](../03-interaction/01-incident-triage.md)  
→ [a row's journey](../03-interaction/03-data-platform.md)
