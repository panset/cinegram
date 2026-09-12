<!-- Generated from 03-interaction/04-manifests-in-the-margin.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# one request, three manifests

The manifests behind the boxes, in a drawer on the edge of the stage.

<div class="cinegram" data-cinegram="03-interaction/04-manifests-in-the-margin" data-height="720"></div>

[Edit in the playground](../../playground/#doc=nFVfj9u2E_wqAwKH34ut-C4_tIH6FKRBETRFizZAUVgBjpbWInESqXIp_-nhvnuxpCz7zkEK5MkWyR0OZ3dnH9VOlbcLRS6GoyrV6vXSukhB19F692r1_2Wvnd0SR15at4yGlr0OrXVF0_ZqofxA7hvCtrYjVuX6UQ3fEB1VqSIdolqoRpXq5gafDGGOwYaMdQ2iIWz8gXgB66DRBL2nAO_SDjUtwW_Tf466paJyNzdYLpfp9y0aq9ugezmj8fO4oeAoEiPQ3yNxxKCjAesjC0QgWIbGHxR2tia5cBu8i_DbBKfxIw2dP_bkYoEPEY0nhvMRbPw-kfjr7S8fEY2O6PUDMcgKLuhgOS6g84MS2ECBvUMg3VjXpuAT2ZFH3XVH7LWLjOjBRLCxwFuHezoYu7HxHsMYeUaTXAjfBPNMoiTL_xgdbWPSq4TxOwpp02jXdIStDzNSrUPDC9SdrR-g06dQ-Mf7PoXITfO2O-6TbNELHdgIvdfHAu91bRLaxFZ0vd_6cJ-z1pFIKMfpMHTaOl5gb2xt0NnWRMY4yLc8KXKikMAsZ-rUwIfEiJoF2M-8TgJLxUAHSupCM7wjRGNdW1Ru2_l9bXSI-Ph75SBPIRfX79LPZ1mxrl1_cG0g5s-VkxUeN23Qg4HjtdM98aBrKiXtQ4oAeFevp7op4UNDgaedhob1uW5ebA6-uV3_5pvz5938Sa7Jt2eGWC6rcbV6LXluJ5rnNd7VieiuPq8J-JcW72SxoQHLQhpOt8QoXgZ9Zf9OaM2ZzUKhUtO_4qj7rlLynG3wPSo1N_Wr6yOPUnxlesvTGZSnDqzU9O9roNdHJlB59wVok5IAVKo5t_EUdAV6fWQCFV2eRACuyelgPSol5TU5ygLRBLowshTJA1FT4rZY5ViAIw0wniMqJcY31Rt6HWtDnLvTc5TwUx1xXU6nw5jb_VJPaY-9OE_wo1icFGdBB90PHRW1dG9ulFx_J5Mr8OtAX_KN5BfST7nVZenkBamrbPwhLZ6Iv-zcox_R-KJSmb103VzI5zrGIzq9oU4e9tP7T3iV2VVqgWYMJb5frXoWzQTD2Naka6ZIjseOSsjE2VE-dKkuU0f1Sd-Tp-dFxuaYL77W97KcYLQMhBzkwyKZvUZnWYaC9AIX-NPoSOKotQ7BEkMPw6nN00Bx0sqDty6ebC6l6pj0O1vD7F4nri6hypb1DqRrAy_j5LmoyQbOLoDHrNx3X1Yun_gP5U50eRLvguMD0cCIez-N3R5hdE4c6UrIFy0EzQ-cRk2gobO15hJ32Nto8sBk3VPOSYF3RruWkhhu7Dd5WgW6Umiw9UOquXSS9ikjCdOPkqg64_htGiR-755Ll4zxwvcuq_F2VdwWd8Xr8s3qzWqux2eqNraf7fRSY7GIWePNyMdZYfX0-enfAAAA__8){ .md-button }

??? abstract "The source — `03-interaction/04-manifests-in-the-margin.dgm`"

    ```dgm
    %% The manifests behind the boxes, in a drawer on the edge of the stage.
    %% ---
    %% A diagram of a Kubernetes request path says there is a Service in front of
    %% a Deployment. It does not show the YAML that makes either exist, and the
    %% person reading the diagram usually wants to see it. An `exhibit` puts the
    %% file in the drawer on the stage's left edge: hover the handle for the
    %% cards, click a card to zoom the file, click anywhere to put it away. Each
    %% exhibit is `for` the element it explains, which lights up while its card
    %% is hovered or zoomed, so the file and the box are read as one thing.
    flowchart LR
      client[Client]
      ing[Ingress]

      subgraph ns[namespace: shop]
        svc[Service: orders]
        dep[Deployment: orders]
        pod1[Pod]
        pod2[Pod]
      end

      client --> ing
      ing --> svc
      svc --> pod1
      svc --> pod2
      dep -. manages .-> pod1
      dep -. manages .-> pod2

    exhibit ingress "ingress.yaml"    from "manifests/ingress.yaml"    { for: ing }
    exhibit service "service.yaml"    from "manifests/service.yaml"    { for: svc }
    exhibit deploy  "deployment.yaml" from "manifests/deployment.yaml" { for: dep }

    scenario "one request, three manifests" { speed: 1.0 }

      step host "The Ingress matches the host" {
        desc: "The rule in ingress.yaml is what routes shop.example.com to the orders Service. Open the drawer on the left and click the card to read it; the Ingress lights up while you do."
        flow client -> ing { label: "GET /orders", dur: 700ms }
        highlight ing { style: active }
      }

      step select "The Service selects by label" {
        desc: "service.yaml has a selector, not a list of pods. Whatever carries app: orders is an endpoint, which is why the Deployment and the Service never mention each other."
        flow ing -> svc { dur: 600ms }
        highlight svc { style: active }
      }

      step endpoints "The Deployment keeps two of them running" {
        desc: "deployment.yaml asks for replicas: 2 with that same label. Change the number there and the Service picks up the new pods without a change of its own."
        flow svc -> pod1 { label: "10.1.2.3:8080", dur: 600ms }
        dim pod2
        highlight dep { style: busy }
      }
    ```

← [a row's journey](../03-interaction/03-data-platform.md)  
→ [the upgrade](../04-diagram-types/01-websocket-handshake.md)
