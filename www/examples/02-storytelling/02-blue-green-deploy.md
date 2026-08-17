<!-- Generated from 02-storytelling/02-blue-green-deploy.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# cut over to v1.5

A blue/green deployment: how traffic moves to a new version with no gap.

<div class="cinegram" data-cinegram="02-storytelling/02-blue-green-deploy" data-height="900"></div>

[Edit in the playground](../../playground/#doc=pFbRjtu2Ev2VAS8WNwFkx1azLaC3pAWKAgFatMmTFWApciQSS5MKObJiLAL0I_qF_ZJiSNm7WW-Kpn1Zr6Th0cw5Z2Z0Jw6i2VYCPcWjaMSmXiUK8UjonPXDi0296tyEqyEi-pXG0YXjWg97UYkwov-6E711mESzuxPj1x0k0QjCjyQqoUUjrq7gFXDsixwLJXaPnhowYQaKsu-tgn04YAIKIMHjDAeMyQYPsyUDPsAgx3Xrr65gtVrl37fGJsCPcj86BPxoEyXoQ4SbZMJ8U8GNsRr5d5aWbkB6DTcJP9ys4Z139hZBQsQDSpfRns3GKgMSlLPqFsLs0_MKGCqfZCyQEYHsHp31CIkkIfz5-x-QVJw66KS65ezJYAbsbUwEiXDMAGQQSv1j0ClDDcEjyEFaX0GHSk4Jc5hyYcmAL_fr1vcuzMrISPDm19YDTAlj2r3jv-_52nW7Z2-C1PBaOukVxuf5ttK7HzLZ8H3wFINzGN-3nh-lqRuiHE0WZvfaTaWUw3b9Mh8F6La7X4KGbnu6rst1na_R60dAubjdj7nEBep6OToUqOEENRSo4RFUrgpWq3babL5BcF0p7f5Ot724Uz--M1zEDHXh4nHMZ3dqTiEp9DLaAK1QE0E4YGRFuZJWwB2kEVE3sF1v4NNSPcubCKU-QisyjQkj-xgPGI9krB_4aKlbY1INxwUqxBcvkLwtwjt7wFM7rKFQqQMm8IGKw-GI1LSt5-hsSROcLrYnbgfOp8p-s5QA9YAJVPAKpcteSug4udxTltatKIllqCxgBcO2WigDYN-dZDmrAnegp9jA9WazT0zEOZJZP0sFd-Bkh44Lvt5ctaIqx77lYxVodPLYwMu_Bam_GuShLk5OXhloxVtuqnMHcOdGSo868lKln33WBySBzG1fFd4kjLlVmXSUykBEqa3HlGCMocN12_pXkPADKCOtT1kKqcgG_t9nt0DoSxrWD1kYoDAgGYxnTfLoeUITBl4yXRhjI5-d_ZCx_IbFvtUTogHwaCz0fRGx_m-IywMfCBm2Ffds1S_q0lZWYwMd8psvRFTSy8jN9QqSswqZudPGoGgxLRrmcftFESvoubdGjAo9reGnPhcBe5s6NPKAqSoNZBPMBiOCpaxAqtrWn-Z352QiiFLbKUeGAv7_BHOIZDi1iB8mTJTOMj729OcKXd9b-rtiaRVciPzof3Wtrq-xFScKjR2Ms4OhgpHo6LDJxjrgE7xNlOdXK8oc4SFTPN9PzoELUl-yxY3SLSsEemfHcmJGfm1aQxlwZJ2DOPmU5zwZSUwGB47BempbH_oHC7_Jj4LT55VutWPCYx6bvDLzAFxE5T36T9m7nAhfpu8Cqv7XUNruodtWy_Z5yHpEshFPq0DH3P95HPNsDuOI-imPqtMkMjKBQVfsxhpVZU_wcSUdapCzPK7ht_MnxyyjTotHLXECU1yUufx46aPcI_v2_jujystFo8M8oD5fCfzu6kGpSx-7Dlqx3WyuGKkMg4UI8en9p78CAAD__w){ .md-button }

??? abstract "The source — `02-storytelling/02-blue-green-deploy.dgm`"

    ```dgm
    %% A blue/green deployment: how traffic moves to a new version with no gap.
    %% ---
    %% This example exists for `show`, `hide`, `wait` and `seq`. Unlike a reveal
    %% (which a click owns), show and hide are timeline state — scrub back to the
    %% first step and the green pods are gone again, because the clock owns them.
    flowchart LR
      users[Users]
      lb[(Load Balancer)]
      cd[Deploy Controller]

      subgraph blue[Blue — v1.4]
        b1[Pod b1]
        b2[Pod b2]
      end

      subgraph green[Green — v1.5]
        g1[Pod g1]
        g2[Pod g2]
      end

      users --> lb
      lb --> b1
      lb --> b2
      lb --> g1
      lb --> g2
      cd --> g1
      cd --> g2

    scenario "cut over to v1.5" { speed: 1.0 }

      step steady "Blue serves everything" {
        desc: "Both blue pods take the live traffic. Green does not exist yet:\nthe hide holds for this step, and its edges conceal themselves with it."
        hide green, g1, g2
        flow users -> lb { dur: 500ms }
        flow lb -> b1 { label: "50%", dur: 600ms, delay: 400ms }
        flow lb -> b2 { label: "50%", dur: 600ms, delay: 400ms }
      }

      step launch "The controller starts the green pods" {
        desc: "One pod at a time, with a pause for each readiness probe.\nA seq chains its actions instead of starting them together."
        show green, g1, g2
        seq {
          flow cd -> g1 { label: "start v1.5", dur: 500ms }
          wait 400ms
          flow cd -> g2 { label: "start v1.5", dur: 500ms }
          wait 400ms
        }
        note cd "readiness 2/2" { side: below }
      }

      step canary "A slice of traffic tries green first" {
        desc: "One pod, five percent. If v1.5 misbehaves, this is where it shows,\nand the blast radius is one pod's worth of requests."
        flow lb -> g1 { label: "5%", dur: 700ms, color: "#22c55e" }
        highlight g1 { style: active }
      }

      step cutover "Green takes the full load" {
        desc: "The balancer flips the weights. Blue still runs — that is the point\nof blue/green: the old version idles, ready to take traffic back."
        flow lb -> g1 { label: "50%", dur: 600ms, color: "#22c55e" }
        flow lb -> g2 { label: "50%", dur: 600ms, color: "#22c55e" }
        dim b1, b2
      }

      step retire "Blue drains and is stopped" {
        desc: "Once green has held the load, blue is scaled away. Scrub backwards\nand it returns — show and hide are frames on the clock, not deletions."
        hide blue, b1, b2
        note lb "100% on v1.5"
      }
    ```

← [payment checkout](../02-storytelling/01-payment-checkout.md)  
→ [authorization code flow](../02-storytelling/03-oauth-login.md)
