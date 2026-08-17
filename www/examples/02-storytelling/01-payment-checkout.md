<!-- Generated from 02-storytelling/01-payment-checkout.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# payment checkout

Checkout, told twice. The first scenario is the path everyone draws; the second is the one that actually costs money, and it is the reason `status` exists as a first-class attribute rather than a colour convention.

<div class="cinegram" data-cinegram="02-storytelling/01-payment-checkout" data-height="900"></div>

[Edit in the playground](../../playground/#doc=tFbBjttGD34VYvAbfwvIjp0W7Va9JYdeAmTR5GYF8HjElQaRZoQhZUddBOhD9F1676P0SQrOSJa8u226h55sDznkNx_Jj75XJ5XvMoWOw6BytX25JvZhYGwa66oX292600OLjtemRvPR97wpq1ZlynfonnXhzjZIKt_fq-5Z91jlivETq0yVKlerFbweHTJg35TAZ2twA-9rhDsbiIEMOh2sB0vANUKnuQY8YRi8QyiDPtOPYijcagWExrtychUHrjWDNtzrphnAeGKC1jscMtDiyZNzQE3ewYFYc0-HGA4_WfHXBDqhWZtGE4FmDvbYM0LQXGOQLA40GN_4PoDx7oSOrXebwt01_mxqHRje_Fw4gGPwZ8Kwf1f7rsPwf4JX6eSDWCe69hMt8A7DyRr8UDixU3-sgu5q6Kjb3yaO4Tb4ky0xUAwBUJ33t8G2Ogzwk2Y862E0HLX52Hf7V_HjyoauTAkaLCsM-6_ehhIDvIm_vh6Tj9BhvS767fYbvKBdIp-t1fnp84TiaVtKL-kudS9UrbtuiIUvFNwDdYhlDrvNFj6PrDB2Qk1rGQo1MjsejK0gz5HbiQgpyvycB6-Be2j0EZscCnX79t17eHF5p8qg7EMO3223LUl2CVbbqm5sVfMyAvHQYC6dZ0-YPJdgdc-1D_YXhEJJr3djvapUk8lOmOAbHcoH6GfuLnQvgc8J_nfz7Wa3vUD_XqBngJowB7n_6BWpbku0AY0P5Qg1UilTcw6WGR2wjxjH0l2jrM6P6b2u9RJ0Cn1zs3t5gfvDzPQSkfHuzoYWCvU6fdMybsBBn7Ch2GITLkrt8CX6pm5YwHm53cHrgJqxvK58NpU3IHXe0bLAq1UUL9-zrlBocp5BT8oUZTJbyA7pFtMp9I5tEw_HLkiSxr4j0I7OGKyrNnA46WC14wOQHgjI58C1pVkoA3aNWK4HJwbjOvi-qmOSQ5qPQyL0z19_g0NEcBBo1pmmJ2ldMYhSco0OSnvCUCFFgU7wah2wBFkg1lXLtvDOYAbkRRdr7SqUgtT-DNrNPZQwCMESzGgnbKEroe-gRDLBHiWsKHkUfkEiLnGmRXo3V0oxjU9iP6rFyFb-gI4s0Z2PCDK5YnyLOdxp21zPKjO2Hf_NpDrZRWN96N_MaLLDP47q5BNbbreLPTedpQWVcKazcYKd55ihUM7PnWkd3FBROOOdQ8PAtsUoZY-HnMMgAzWBlgQEXp43jlJSbujGffPFkUrui4mKOfJZ_Z5WJevWC2EqbTtK0lKk5iVytQOQuZk0dUx_kdROcCOB5YeLIDk-Q6jGWCX88Ts8IVq77XI_zA9YQk0tup6F7P2sVUCIC3kQHUIb_2iIKvz3QvYAs_r84fNfAQAA__8){ .md-button }

??? abstract "The source — `02-storytelling/01-payment-checkout.dgm`"

    ```dgm
    %% Checkout, told twice. The first scenario is the path everyone draws; the
    %% second is the one that actually costs money, and it is the reason `status`
    %% exists as a first-class attribute rather than a colour convention.
    flowchart LR
      browser[Shopper's Browser]
      checkout[Checkout Service]

      subgraph psp[Payment Providers]
        gw[Primary Gateway]
        backup[Backup Gateway]
      end

      ledger[(Order Ledger)]

      browser --> checkout
      checkout --> gw
      checkout --> backup
      checkout --> ledger

    scenario "happy path" { speed: 1.0 }

      step submit "Shopper submits the order" {
        flow browser -> checkout { label: "POST /checkout", dur: 600ms }
        highlight checkout { style: active }
      }

      step authorize "The primary gateway authorises the card" {
        flow checkout -> gw { label: "authorize $84.10", dur: 700ms, ease: out }
        highlight gw
      }

      step record "The order is written to the ledger" {
        flow gw -> checkout -> ledger { label: "order 8812", dur: 900ms }
      }

      step confirm "Confirmation travels back to the shopper" {
        flow checkout -> browser { label: "201 Created", dur: 600ms, style: response }
      }

    %% The outage is not a second story, it is the same story until the gateway
    %% stops answering. `variant` says so: this scenario replays "happy path"
    %% through the `submit` step — `until` is inclusive — and then diverges. The
    %% shared opening is written once, so a change to how an order is submitted
    %% cannot end up describing one path and not the other.
    scenario "gateway outage" { variant: "happy path", until: submit, outcome: fail }

      step attempt "The primary gateway never answers" {
        flow checkout -> gw {
          label: "authorize $84.10",
          dur: 1100ms,
          status: fail
        }
        note gw "no response in 8s\nconnect timeout"
      }

      step retry "Checkout fails over to the backup provider" {
        flow checkout -> backup { label: "retry: authorize", dur: 700ms, ease: in-out }
        dim gw
        highlight backup
      }

      step settle "The backup gateway approves it" {
        flow backup -> checkout -> ledger { label: "approved · order 8812", dur: 1000ms }
        dim gw
      }

      step outage-confirm "The shopper sees the same 201 either way" {
        flow checkout -> browser { label: "201 Created", dur: 600ms, style: response }
        dim gw
      }
    ```

← [ship a release](../01-basics/02-deploy-pipeline.md)  
→ [cut over to v1.5](../02-storytelling/02-blue-green-deploy.md)
