<!-- Generated from 01-basics/01-k8s-request.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# GET /api/orders

API request path through a Kubernetes cluster. Strip the scenario block and this is ordinary Mermaid.

<div class="cinegram" data-cinegram="01-basics/01-k8s-request" data-height="990"></div>

[Edit in the playground](../../playground/#doc=lFbvbhu5EX-VAQEBuYMkS47TpvqWM9w2OB9ixAYKVJsPXHKkZcUleSRXimAE6EP0CfskhyFXu6s_iX1fBO2QHP74m5nfzDPbssV8zNBEv2cLNptPSh6UCFez-WTzPkw8_t5giFO5rtmYWYfm5V0rpTGwxfKZuZc3R7ZgEb9GNmaSLdhoBB8ePkK7BxyPFcTK22ZdAYdfmxK9wYgBhG5CRD8tzGgEj9ErB7FCCAIN98pCqa3YADcSYqUCqADWS2W438Nv6Guu5LQwK213ouI-wv3nwgAIrdDE5d3XiN5wDbfp-wst6XL55lbbRsK95RJ-4Zobgf6nL4Wh5dCUa89ddcC1HEC9zabkBkCZ9fKjWXsMAW6tid5qnRbzcu_ILUVeBqe5wfY4AHdquWlKnHCnAvpt5xkAo5DLN_T7U2tDI888m7A0vMbguMAFOG9l5yBsxbKF-_EBHtFvlehvdlbOlw9Wwoeh6TqZfhleOLg3UwqTSdHMZm8RdJnZ7C3KrMmkzLq3ha1IrG5Fb6PbLxmvycid6o1EAN2-VbijHR-gYB9NUBIhoS8YrLytoWDT6ZWzcsIpGQtGZ5SJ6LmI8ExuR6N8gvKHg7TWL-hFYkNoVQQqiJASTyq-9rwGu4JdxSNU3KU1le6dtlSITXoGHJD2CJ9B8xL1Agr2b2trUCbaDu23zOVoBE8VwlFaEDJjI1iTYAwLZwzBEsgQ-T6AbSKBixW2rnZ8D42JSkOwNVqDwMMmwMp6UHEAuM3oDrPHLXINwnWgUvwJCCFAI51VJkIkGnY8wxOVDWjg___9H_ynqR1Em8s1ossbA2HcVftjpq67W9POMlddYRIhXa0X7B93T3DFnbqyXqIPBYNnCA5RLmA-nY1BW-sWEH2DBy6TP5RrhIJ9bknzyEWF-Rmaqry9z5O_nN6kGF1Odyk9jN4ZlDHIxi_gr7NZHeh2clOpdaXVuor5cIh7jQvgIqot5j1DmBF9rQyPhPXp_rH_DsBjQqtaQRGdoJxApoLr622I959PTw9X8-m8A_qXy0DzsReQetsklAeB841GqHlMtFY2xKTIlJw9PmNjBlUw2rGgWp7iV147jVNh66IwdGCRWf2ZyrR7VhKNXjPgOT_hXf-EIbo2nFCwpJ_O2697cEpsqLgr5DpW-y5_TwhMqjNQoiGF89l0Pr2evl28n72fXeZRqrrTqiGknfUbKFgWmYobqdv8awu5R9FHor2-DUXZhH1749-GNyZa09aCXc9m8OlXUAbmN3XIDB6FDYOzRqZSoH8BIXq-RR2g5GJzqNac9yfEHOnZkKVhbAbp1xbP86GDdCxmkAUbH1bSm-Y39KjO1j7atzCz-RBo9m2cZ45O0y8OGP8ivcmMS4sBdirScHGQTkspMkiQioeUJCizLo5G8DkJhYRy33eDtk8Y-NHAkwSQSmBwMnlMYpiaeXJxM2n7ySTuHSZPOyyDFRuME0qTUPENkscx7ColKlLfXbVPw05ySEMYBBU7ifDWRvA8VuhJcU3bmQBVMq2sluin8OCRmhIvNQLX1uCFMUmZ5UEyUx_t54SjUchZeTQtJDqXd2Zr9_CoJAp-mFy4c8tPJJdp-KOpiCvTzTWCyF6-ubeCRjL6yNNNN2PIcvkmHQ_wYEMk6TnMZcoMZgW6n4w5rp2dO5dHCNfb0p1nVlme9J0VRlHBn2s6XNYqQsFaCvL3d4q-FTk4esJrus27yyJ-OP-CjK-s33FPevCQDrTf4SAERIo4ROlUDzK5PbdDuJpimFT-SCpvzoUreSlYfdTvZFEYT31QK-LQbs6lTFu7aRwU7ANhrJDEnehNN-ewwkr5M5JTnIfBH8LO5C6am-sO8tt3p5DzqYLVKlyQ2N8b9PSglL9Ae9J4dmCTlhX2-fsDdPJo3Hi8u7-7fYKf4e-fP_0Gr5s5ZHncPi718V1IbSmjAY-x8e2kS2sn8GT5vYDPb9r9w6Y4PpXxc4jZzXmL-05nXymt25jT3ww0B4TUlpuwayv0tSF_vHuCS2E_y1Ty8poO6_R-2F818m3b6p2VP0B2XE_KvLpzvv8zjfPLtz8CAAD__w){ .md-button }

??? abstract "The source — `01-basics/01-k8s-request.dgm`"

    ```dgm
    %% API request path through a Kubernetes cluster.
    %% Strip the scenario block and this is ordinary Mermaid.
    flowchart LR
      client[External Client]
      lb[(Cloud Load Balancer)]

      subgraph cluster[Kubernetes Cluster]
        ing[Ingress Controller]

        subgraph cp[control plane]
          api[kube-apiserver]
          etcd[(etcd)]
        end

        subgraph ns[namespace: prod]
          svc[ClusterIP Service]
          pod1[Pod A]
          pod2[Pod B]
        end
      end

      client --> lb
      lb --> ing
      ing --> svc
      svc --> pod1
      svc --> pod2
      api --> etcd

    view podA "Inside Pod A" from "../pod-a.dgm"

    interact {
      %% Pod A is a door: clicking it opens the diagram of what happens inside.
      click pod1 -> view podA { label: "Zoom into Pod A" }

      %% The control plane is not on the request path, so it stays out of the
      %% way until someone asks for it.
      click cluster -> reveal cp

      %% Pod B is the endpoint that was not chosen — jump to the step that says why.
      click pod2 -> step balance
    }

    scenario "GET /api/orders" { speed: 1.0, loop: true }

      step edge "Request reaches the load balancer" {
        flow client -> lb { label: "GET /api/orders", dur: 700ms }
        highlight lb { style: active }
      }

      step terminate "TLS terminates at the ingress controller" {
        flow lb -> ing { label: "HTTP/1.1", dur: 600ms }
        highlight ing { style: active }
      }

      step route "Ingress rule matches host and path" {
        note ing "host: api.example.com\npath: /api/*"
        flow ing -> svc { dur: 500ms }
      }

      step balance "kube-proxy picks a healthy endpoint" {
        flow svc -> pod1 { label: "10.1.2.3:8080", dur: 600ms }
        dim pod2
      }

      step work "Pod A handles the request" {
        highlight pod1 { style: busy, dur: 900ms }
        note pod1 "200 OK in 14ms"
      }

      step respond "Response travels back to the client" {
        flow pod1 -> svc -> ing -> lb -> client {
          label: "200 OK",
          dur: 1400ms,
          style: response
        }
      }
    ```

→ [ship a release](../01-basics/02-deploy-pipeline.md)
