<!-- Generated from 01-basics/02-deploy-pipeline.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# ship a release

A commit's journey from a laptop to production, and back again when the error budget says otherwise. It exercises the attributes the compiler emits and the runtime honours: scenario `speed` and `autoplay`, per-action `color`, and `ease` on the promotion hops.

<div class="cinegram" data-cinegram="01-basics/02-deploy-pipeline" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=rFVNb-M2EP0rAxZGt6jlWl6n29WtSBdF0AANsr2ZC5iSxhK7FEmQQydG4P9ekJLljzStD3uyxRkO35s3j3xhW1bkU4aa3I4VbJ5npfCy8j_NF1mNVpldZqVFJTXO6qZjU2Ys6usyN1KhZ8XqhdnrNhArGOEzsSmrWcEmE_gVKtN1kr738LcJTuMONs50IEAJS8YCGbDO1KEiafQUhK6hFNVXEI2QGp5a1EAtcj2ZADpnHJShbpDAi50HQy26J-lxBncE-Iyukh593AGCyMky0PBZmc5KhQ6wk-RTvXhWDLmgSXYIrdEmOF-Ar1ALJw2svUWs1ylzLQIZq8RuPQWLLhMJcSq0rowybt2jX6PwuAaTcEdunYmJ0BrrZ1xvlHmqWuEI7h-5Bqhxu_oNt6iMRQf3qSlfYqCRtHr3uyR4RGu8JON2P6RAJVe3d_AwdD4tOWxW726NJiE1OnjERnrq82PYh7JxwraAeutXf4QSncbYmE96K53RHWryqRCAJ9FI3aw-979wq4IndEM0SrV6GPU6i6Ku--Nq3EKW8TCfv8dIY2BzXKtkz-O44rAZeBzXBiSJwABmjEUcMRB_IZuBM0qlsZmdVeR6lJIz30oLAhyqqBBn8AJJ3gLy2c0UDvIWQC4g7IfWEdphhIGzo1A2-BY9iCEWq_Udivr2HTg2AF5AiRJVAZzFz7iZsynUwRVwM593fgpphGLCd4tFdXMT8e37iq1sWiWbloZSb2WeIi6DVDVwdnvX__X9sKOnwQ2t0A1eoE4ajRKdgn7CsjXm64h5-X-YbVB-KPJ2kjaUcjgLWhL8CFITNk4kX3G97CBfeM4uudlQKulb4OyvFkF2okFQQtceZO85N4z_Bb04b8fhOKXnW5F93CyqXIwMf75elb7YNarEMUbg7CFdChjvvsOQn0NNPji3wSEOR9gtqg6CbZyokbPpIZ7wf0j4D0uvwY2h6IUCpM5MoH7tFcERwVUk7UDujObpFe-VeVI7EAQb6fyldUarnzj9X7hXQgu3g3w-iZJryOfzyWUP8vwbNqHHcVUHTOnRbWMHPqUXy4k450p2pQcrPKUh7Z-xI_n_OAk3y-VyOY5mkvbMQ2kHZzfPz7Gry9liwrn-fP8nlMHp_vh8-fzaSeOtydmjUap_eMkkfCoCbYype4ddqNTfu6dueqXQWDxOeSuy9_UvefnhUqOPb0g0cj6X6G19Ll14KHDIrGV3-qLsuWb7L_t_AgAA__8){ .md-button }

??? abstract "The source — `01-basics/02-deploy-pipeline.dgm`"

    ```dgm
    %% A commit's journey from a laptop to production, and back again when the
    %% error budget says otherwise. It exercises the attributes the compiler emits
    %% and the runtime honours: scenario `speed` and `autoplay`, per-action
    %% `color`, and `ease` on the promotion hops.
    flowchart LR
      dev[Developer Laptop]
      git[(Git Repository)]
      ci[CI Pipeline]
      reg[(Container Registry)]

      subgraph envs[Kubernetes Environments]
        staging[Staging Cluster]
        prod[Production Cluster]
      end

      dev --> git
      git --> ci
      ci --> reg
      reg --> staging
      staging --> prod
      prod -. rollback .-> reg

    scenario "ship a release" { speed: 1.5, autoplay: true }

      step commit "Developer pushes a commit" {
        flow dev -> git { label: "git push", dur: 500ms, color: "#22c55e" }
        highlight git { color: "#22c55e" }
      }

      step build "CI builds and tests the change" {
        flow git -> ci { label: "webhook", dur: 400ms, color: "#22c55e" }
        pulse ci { color: "#22c55e" }
        note ci "unit + integration\n4m 12s"
      }

      step publish "The image lands in the registry" {
        flow ci -> reg { label: "sha-9f2c1a", dur: 600ms, color: "#22c55e" }
        highlight reg { color: "#22c55e" }
      }

      step stage "Promote to staging" {
        flow reg -> staging {
          label: "helm upgrade",
          dur: 700ms,
          color: "#22c55e",
          ease: in-out
        }
        highlight staging { color: "#22c55e" }
      }

      step promote "Promote to production, slowly at first" {
        flow staging -> prod {
          label: "canary 10% then 100%",
          dur: 1100ms,
          color: "#22c55e",
          ease: in-out
        }
        highlight prod { color: "#22c55e" }
      }

      step observe "Error rate climbs past the budget" {
        highlight prod { color: "#ef4444", dur: 700ms }
        note prod "5xx at 4.2%\nSLO burn rate 14x"
      }

      step rollback "Roll back to the last good image" {
        flow prod -> reg {
          label: "rollback to sha-3d81b7",
          dur: 900ms,
          color: "#ef4444",
          ease: out
        }
        highlight reg { color: "#ef4444" }
        dim staging
      }
    ```

← [GET /api/orders](../01-basics/01-k8s-request.md)  
→ [payment checkout](../02-storytelling/01-payment-checkout.md)
