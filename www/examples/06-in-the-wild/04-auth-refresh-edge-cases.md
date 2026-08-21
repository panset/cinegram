<!-- Generated from 06-in-the-wild/04-auth-refresh-edge-cases.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# auth refresh edge cases

One diagram, three scenarios: a token refresh that works, a refresh token that was already rotated by a concurrent login, and an identity provider that does not answer. The happy path is the only thing drawn at full strength; the two edge cases are `variant` scenarios that replay it up to the exact hop where reality diverges and then tell their own ending. The shape of the question comes from r/webdev (reddit.com/r/webdev/comments/1u87ptj): "the happy path is clean but the edge cases are killing the diagram."

<div class="cinegram" data-cinegram="06-in-the-wild/04-auth-refresh-edge-cases" data-height="1260"></div>

[Edit in the playground](../../playground/#doc=zFrdjhu5cn6VggDDXhxprLE9a0eLXDg5C8PIzzHWBnwRBWOqWermiiJ7yWq1BWOB8xDnXXKfRzlPElSRbHVrNLP2bgLkxh5JJJusn6---thfZofZ6no-Q0fhOFvNlt8vjFtQg4veWP10-WKhOmoWAbcBY7NAXeOiUhHjla73s_nMt-h-x7StsRhnq__4Mmt_x2yarWaEn2k2n-nZavboEfzFIWij6qD2c6AmIEKs0KlgfFyBAvI7dJAXBGoUQe_DLs5Bnb7lMWv36FH-XUVQNqDSRwieFKGGzREUVN5VXQjoCKyvjZuDchqUA6PRkaEjtMEfjMZwWkx7jOA8gXKxx3AFHxqERrXtEVpFDZgI1CB4Z49AjXE16KB6B4pg21krC0UK6GpqfpCh1Htgw4AYBlRA-HRQwShHn05nT08P2Fp1BEPQtUA-bwsBP6uKoPEt9A0GhIDK8v61OWCoeVWneaADQmv5LxPA9w7QaeNqOUXaWqNaBL-VVX_pMJLxDiq_xwjb4PcQnva40XiAJwG1NnRV-f3T8uXTyu_36Cg-ve5evWzp5-9Wsup6RnesVFlUDjYdpQNMDbAz1rLt-KccDVfrmay1WCzk_3_3ybyboFzVYATvZHil3EHFOUQvH7fK2C7w10FHiKSObBytNhahb4xFHiUL7pVxsLW-l1HJjVHtEdDiAR3sMUaVbAneLSplLaCrjUMMQ3ztnO_j1dpF_KVDV-Gf0-bXDqBVgUxlWuUIXrctqAj_5je8g9dtez7gzUf-_fW7t_BGEfbqeGeFjhoZwv-_x3AwFZ6Peavf8ZC3JZrfDdE8HffPqmqQR77HGNnf8sXa8bjXbbtYd8vlc0z_vvm4gjc_foCnPmgMEZ78E6rABqDbF9_xhDcfF4vPr9t2BS-W1ykVb_FzawLqS-vx_lfw7i_vP8DTkr9PAt2-lMX418lw2dkKrPe7roWt2ht7hO3tP1zPCzLQ7UueKQMXFx7FAyCn_bysoCoyBxzN-zwafI4dTwJ2ES_v761-V06T9vOkDsrRLR1b_Md8vlv5Rea_1e8u7fHZcgmqqjBGtusN_GmAtkC3r4aJeZPOQ8DYehfxIZtF8owMdPtqDgHJpA8vhynTfbADeRsO-7KVDKtl-MnJxh2UNfpWjvqNQXMzBM3Fx6cJazdJfYbcjYooRzpewY8HDEeIhC0I-pkIxjUYTEb6grIZVSNs0PqeIWIA0NiogBq4CDKomAh9METoErTaI3hXocBopRzjvw5mS5zpGaFhPTsB3GpiM0jxH-fFjeKAcFzP4AvEFlGv4PpqCb-mlJODCLysZ3xU1aaPCZIYE3rDyZ8XN8TnMXjAyPuOZKyF2nvNy_N6ABpjtYL1rGCmyVBpDdc-wSz5YrLphstmzUbp6AreNz7QwpoD6smwhNcardlgUIT2CFWDiqsTSFQoSmYz-9bHaBh3iQP24HcoKC3GOV4onJGUVCfYIPWIDhRYVDvUeYNSrIG3BDEBl9QIPrDAOINsjirG0y-gu7CCl8vlPs5hH-sVXLPJeXzEhMlfYKN0jWwrBrT1rAyoVVdjHmLVBq0MGefFbM7n7S7MbUzdWFM3lHYR6WhxlSEnjRk7PmNl9n2d0B8C_owVxeTsLScyGyuglGiofCSuS0cx3F2_j1c6YDBbg7m-mdop4vqYCQI_HiqrzB6s56A7wt__-jeGGIfERItJxpw_54yJjInMHxRJ1jRMrBgUSiZuVSRZnX-i0UYcHjCAQ9RsQonC4QigNr6jKaGLU----Tg4N7lFvPt98i7vqosrqf7nvnaeJB7WMz7r9YvV8tnq-nq9dryB9PHFMj1q7Jeyl1NO5m847VwkVJp508Z3rirUpYsY-Gwq8UuIVUAOlbv-MbEkQN94i9B646jwsIkZVtm-JjLj9R0Zh8AEheaJl5ZkGLjda6h8CFhRSfjTxhnU5gwH3iG0VvEn9tUvHXYYedlwBE8NBjBusU1RXKJug43hxKb7007YyUOJt_VVl0jMKBHTpCET83ZTZN_JF4nnY3ZLOXoljKbybmvCPqYynu2bKv7jWDhASd9zn_w0iT1BOamgmomSysusxHDB99BiKE-fQ-OtLjGQsIpyfpSHBrFkCnmd7ZxRLbON2KKj1F5ktpM6jiilrFBzJgipNHGayhFNHJokZZM7jQBHqgy8J4c9O5D3bmLsUMPWhzT1HEkj_lIsU_zL7ikOTtQxe_jm5oKH86w08FJc5FkZFguTOV9mgsPlqQMSCzUzforCL08QzGFVJg1xNWKPhQDOLuFx1ShXY0l83nlMfBtir9p4N0XZJ_zlW_3uMhZPFkk4uDdMTHKwGYpotz-w0yTNc95mOLjTnYI1O0xBxC5FGwdEdtpw5nNOCDorgr3aZfR3-JmGDhOG8syRxFHTKuO2nf3aUOBu46FUz3N42APw8HVhkGCD7XuO0sLRs5mnTkk_xXnugzOI1yhVNQrRbZUJdz1W0COR5lQm934vaUy3rwA_m0jxCn7KQThYmrrACcq4YceihWxHumnFuLFXjtfaqGqnvWeklZIxAo5Ivo2icxRUEdeR2RceoOwdcP_DKfzszHOTWeOa-3-VwK-mCXyXoN38AYI2mtu5UnbuFn0Kpbb4YGrjlB3qn4REa9WRa4LTELuqYjZzOeeFC6Rcj0qYTspnE6_gfZc2Kpxpj8pNQaVXTDEORtIzlZwXy-t5qj2lo0gfKBznA5cT9aL4IxU_E5gucm_Bi8aubX0UQs2J8FDEXOTS3y_vj5bf4GeXYuXZWax8JXEedYY_FhmJzXEFb4tiluwpPSM1wXd1A5-y5T4J3n3qHBn7KbWOle0iP-Lvf_1bEsV8yTItVT7DZ8LhStmhbWTD5wYGHfSKRJhKqhQHfC0iw6NHoECb7RaFCqSyzi4YfFaq-7S9PKsx56rEuZoJT_Phv5M-Mze_q9_Vp85B7LMahVtHld9jotjTnFEVLjJlyayMz5S_CZ0TrXYbVCpLfiuUtfIctN6DVYSXM4jUxiKdiLXFLUkHg5opbAG-UewXOvicow5U7VPPOGZYwg2v4EPvQSPX4yidDXuTMypjdWIKPfMnPl4quMKX-gZTUm88cabuEJiqUabW_DyHRhh0Bmht9KnP6YN39Rl__jZ-lUphVuzO0auLpRamAAzY-kCZEI9kcTHFKk_QSFgRXtAPXp_VsDZgRMfRp7bEPQKJarDBodwm7um0iWRc3ZnYiPKai18ujGzoDbI5Cp4m3ptZfNJbRLhWENlEbcNT0iLMtkhVOwyD5tsGT77yFiJnvoqxy3Wy9yGedSv389KhrN1pJZ9NWsm0QqYJKaZSF_n81XrtNsccVzl8F89OIHuBk557ID1_lGbnHj74XXExo1E4C1mpUTxGp-7w5y6mkL_U8Bz8LsV9QrBTdUtL9r6zGiyqg0jnVSNxX_lWHlOj64xDewTVcC_MqJopZVqtuOlxlN9SN5I5zRX8y0jxHxPzylur2lj0ipzQ3K6kpikbt3RcXFazLOa4M4X12hVtQrqxoHTOXI2MHZkOewdtF7gYcn8duNvbKhu5EY9G6o31tUj_HHqREgL5jvKdUaJg1DAkodNRJLrcV5ecMY4wHNSJUEuYXV-9il_ZpGRPXoyKb2NWw0qTSJboLz_KI1frteNomWfdmIlDhskhiyYd_Dg4NbrjSC6RvtQK14ikAoE_4AW2PRGTJ6pgtnyul4z-AhNJZhKFaipV85Ma9g4z64TE_gzAeh-ogR1ia1w9PxX6TJkYzMjDBoEw7BP5U4Ll1CiXiiNn9QMAfokn3w8oF2huLnC-owf9_rXM98zzUzBJvGHEVwPKBW7iOdI8eTdJvIe1rA2m61lqIGCDKsQk54qAKcWibTFpGM_VnjNPGGng1qbaITGA84zHRjhNjRr2yKbgbUjB3R9zMWD37o9g3in9uDweP7cCozyj7aR_HuMrA0npqbJCZo8JJpH7aldhYiebrj7P2e_jOG9aZjrbjtlBMth67VoVY--Dhj8VdrNVFfkwLGT28Obj6e9xM3uR0VLvf4PRJiksEdoT6YzKaDjmxlfWnOaAibA17tT2n-6MKeceys1yloeSaDFcg_v-jKXeVSe0793_IgVNp3yIgRrdLkb3J2XhrOPUvHWOocy4L6o0cqWUryE8bFWQfkkk8dVYxX4cRxp61WC1m49Mnzhv-qbgMGy832XEYfw6ZUtqzpKpFaVCm91sFZdtpU2XyxiqPVTeUfA2PoA-v6XJ3KujkNkj2-h0Z5TeAWAovcQLb2Kht2VmogQYfBelXBdBA51O4jbHWuiM4wGdExnLK73K8HISfcvCXOJjemvD5ezlvksW36ZLIE6zWPCFw6KgunJiik3wvRPXizKThBY3leN8R6pG7lCwH5bkpiRM7XyfjnV9df01rJGnr2fjq9u1yycbTHgTx1wx-XKoC50LqCqh079JE7fKWimCF-RHjTVzosklRjbd5TascLByiSiRPlw8SvFO9z0qlssmJkhZMVO1H-rsVEs1dRLL7rloTHSfGh_zM3W6p4rnJbkL0ovfLJ9n9ldYn4mMLtZUhlYpYvbGdcRosJW9-GCkQKlasTHK-pKNlXJMH6x3NTd7WMgYdG2kgGov1ygDyKSIL2fLIh6jdhelASqYnQMzBd0VlEuHMJISY8dNhWYGv8FKcf1KP3P7aKjx3aA2Z4quMR5d1QTvTNF9hEVOi9izeKf4XL5_yRHyMO_MkwYCMoTchw__OiEgN2z1y8RT-Li4OTs4h5rczQ02mReuF-juLV3Z64h3isKr5G0CQ9NuslW8xnmQ_3l8mX3SEiSjofIapwKbvDIlt2wZANnZG2wEvQeZR94gAHXa4NAVjAT-9Loap1GaP5lunLya5iqpsPPUgvH0NmBlIm-2b9DB0XfQC3cm2PhgHtQXvlnGvUeIhf_-Lzjz7LcrsvCkWOe78wARhvVsuZzLG2McDXTL7XWmC2yc_Pj7ImIx1nLT30WwnSc6GYLvudk1e7xfwn0cB_1XiERO5HzP5qa4QnduigVaruDtdkjZATmZJp2iasifLZ_vBzDbnMj5602na6SEaY3qIonuZbj6pUrNo2-Wz9O7Ij_xeRevRaSRIDt7Hy9JWLnd4Xq_Szfw_98U4QlWfSPwvBiH569rN_v1P3_9nwAAAP__){ .md-button }

??? abstract "The source — `06-in-the-wild/04-auth-refresh-edge-cases.dgm`"

    ```dgm
    %% One diagram, three scenarios: a token refresh that works, a refresh token
    %% that was already rotated by a concurrent login, and an identity provider
    %% that does not answer. The happy path is the only thing drawn at full
    %% strength; the two edge cases are `variant` scenarios that replay it up to
    %% the exact hop where reality diverges and then tell their own ending. The
    %% shape of the question comes from r/webdev (reddit.com/r/webdev/comments/1u87ptj):
    %% "the happy path is clean but the edge cases are killing the diagram."
    %% ---
    %% Nothing branches on the canvas, so the failure cards stay readable while the
    %% main flow stays the same eleven messages an on-call engineer already knows.
    sequenceDiagram
      participant App as Mobile App
      participant GW as API Gateway
      participant Auth as Auth Service
      participant IdP as Identity Provider
      participant Cache as Session Cache

      App->>GW: GET /orders (Bearer at_4)
      GW--xApp: 401 token_expired
      App->>Auth: POST /refresh (rt_7)
      Auth->>Cache: lookup family f_91, token rt_7
      Cache-->>Auth: rt_7 current, family active
      Cache--xAuth: rt_7 already rotated (reuse)
      Auth->>IdP: POST /token (grant_type=refresh_token)
      IdP-->>Auth: 200 access at_5 + refresh rt_8
      IdP--xAuth: no response
      Auth->>Cache: store rt_8, retire rt_7
      Auth-->>App: 200 new access token
      Auth--xApp: 401 invalid_grant
      App->>GW: GET /orders (Bearer at_5)
      GW-->>App: 200 orders

    %% ---
    %% The base story. Every step here is inherited by the two variants below, so
    %% the shared opening is written exactly once and cannot drift.
    scenario "happy path: access token expires, refresh, retry" { speed: 1.0 }

      step call "The app calls the API with a token it believes is still good" {
        desc: "Nothing in the client knows the access token has aged out. Short-lived access tokens are deliberately cheap to validate and impossible to revoke, so expiry is the only thing standing between a leaked token and a live session."
        flow App -> GW { dur: 700ms, msg: 1 }
        set App { badge: "at_4" }
        gauge App { label: "access token", value: "at_4" }
        highlight GW { style: active }
      }

      step expired "The gateway rejects it before the request costs anything" {
        desc: "The gateway verifies the signature and the exp claim locally — no network hop, no shared state. That is why a 401 here is fast and why the gateway never needs to know anything about refresh tokens."
        flow GW -> App { dur: 600ms, status: fail, msg: 1 }
        note GW "exp 14:02:11\nnow 14:02:40"
      }

      step refresh "The app refreshes instead of bouncing the user to a login screen" {
        desc: "This is the whole point of the refresh token: a 401 is a routine event, not a session ending. A correct client refreshes once, in one place, and queues every other in-flight request behind it."
        flow App -> Auth { dur: 700ms, msg: 1 }
        focus Auth
        set Auth { badge: "refreshing" }
      }

      step verify "The session cache confirms rt_7 is the family's current token" {
        desc: "Refresh tokens are stored as a family: one row per session, holding the token that is current right now and every token already spent. The lookup answers two questions at once — is this token real, and is it still the newest one issued for this session."
        seq {
          flow Auth -> Cache { dur: 550ms, msg: 1 }
          flow Cache -> Auth { dur: 550ms, style: response, msg: 1 }
        }
        gauge Cache { label: "rotation", value: "7" }
        set Cache { badge: "family f_91 active" }
      }

      step exchange "The auth service swaps the refresh token at the IdP" {
        desc: "The auth service never mints tokens itself; it is a client of the identity provider like everyone else. That indirection is what makes the next scenario possible — and painful."
        seq {
          flow Auth -> IdP { dur: 700ms, msg: 1 }
          flow IdP -> Auth { dur: 700ms, style: response, msg: 1 }
        }
        focus IdP
      }

      step rotate "The refresh token rotates, then the app gets its new pair" {
        desc: "rt_7 is retired the moment rt_8 exists. Rotation is what turns a stolen refresh token from a permanent backdoor into a token that stops working the next time the real client refreshes."
        seq {
          flow Auth -> Cache { dur: 550ms, msg: 2 }
          flow Auth -> App { dur: 650ms, style: response, msg: 1 }
        }
        gauge Cache { label: "rotation", value: "8" }
        set App { badge: "at_5" }
        gauge App { label: "access token", value: "at_5" }
        unset Auth
      }

      step retry "The original request is replayed and succeeds" {
        desc: "The user never saw any of this. Success here means the refresh was invisible: one 401, one refresh, one retry, and the same response the first call was supposed to get."
        seq {
          flow App -> GW { dur: 600ms, msg: 2 }
          flow GW -> App { dur: 600ms, style: response, msg: 2 }
        }
        highlight GW { style: active }
      }

    %% ---
    %% Edge case one. It replays the base through `refresh` — `until` is inclusive —
    %% so the reader sees the identical opening and only then watches the cache give
    %% a different answer to the same question.
    scenario "refresh token already rotated (concurrent login / replay)" { variant: "happy path: access token expires, refresh, retry", until: refresh, outcome: fail }

      step race-lookup "The same lookup runs, a fraction of a second too late" {
        desc: "The tablet the user left signed in refreshed the same session 300ms ago and already spent rt_7. Two devices sharing one token family will race like this whenever both wake up at once, and neither client did anything wrong."
        flow Auth -> Cache { dur: 550ms, msg: 1 }
        focus Cache
      }

      step reuse "The cache reports rt_7 as already spent: reuse detected" {
        desc: "A refresh token presented after it has been rotated is indistinguishable from a stolen one being replayed. The cache cannot tell a slow phone from an attacker, so the protocol says assume the worst."
        flow Cache -> Auth { dur: 650ms, status: fail, msg: 2 }
        note Cache "rt_7 spent 14:02:38\nby device tablet-2"
        set Cache { badge: "reuse detected", state: fail }
      }

      step revoke "The entire token family is revoked, not just rt_7" {
        desc: "Revoking only the replayed token would leave whichever copy is genuinely ahead — possibly the attacker's — still working. Killing the family f_91 collapses the session for every device holding any token in it. \nThat is the trade this design makes on purpose: a rare false positive logs an honest user out, and a real theft ends within one refresh interval."
        dur: 1.8s
        set Cache { badge: "family f_91 revoked", state: fail }
        gauge Cache { label: "rotation", value: "revoked" }
        note Auth "revoke f_91:\nrt_7, rt_8, all devices"
        focus Auth
      }

      step deny "The app is told to start over" {
        desc: "invalid_grant is the only honest answer left. There is no new access token to hand back and no refresh token worth keeping, so the response has to be terminal rather than retryable."
        flow Auth -> App { dur: 650ms, status: fail, msg: 2 }
        set App { badge: "signed out", state: fail }
        gauge App { label: "access token", value: "revoked" }
      }

      step relogin "The user re-authenticates, on every device" {
        desc: "This is the beat worth rehearsing before it happens at 3am: a support ticket saying 'it logged me out on both my phone and my iPad' is the expected output of reuse detection working correctly, not evidence of a bug."
        dur: 1.6s
        note App "full re-auth\npassword + second factor"
        dim GW
        dim IdP
      }

    %% ---
    %% Edge case two. It replays the base through `verify` — the cache said yes, the
    %% refresh token is fine — and diverges at the one hop this service does not own.
    scenario "identity provider down" { variant: "happy path: access token expires, refresh, retry", until: verify, outcome: fail }

      step idp-call "The token exchange goes out to the IdP" {
        desc: "Everything so far was local: the gateway's signature check, the cache lookup, the rotation bookkeeping. This is the first hop that leaves the blast radius the team controls."
        flow Auth -> IdP { dur: 700ms, msg: 1 }
        focus IdP
      }

      step timeout "Nothing comes back" {
        desc: "A 5s client timeout is generous for a token endpoint and ruinous under load: every refreshing client holds a connection open for five seconds before failing, so an IdP brownout turns into an auth service outage a few seconds later."
        flow IdP -> Auth { dur: 1.1s, status: fail, msg: 2 }
        note IdP "no response\nconnect timeout 5s"
        set IdP { badge: "unreachable", state: fail }
      }

      step fallback "The auth service degrades instead of failing" {
        desc: "The session in the cache is still valid and was verified a moment ago, so the auth service signs a short-lived access token from those cached claims rather than returning 503. \nThe trade is explicit: five minutes of authorising against claims that can no longer be revoked upstream, in exchange for a service that stays usable through an IdP outage. Refresh rotation is suspended, because rotating without the IdP would desynchronise the family."
        dur: 2s
        dim IdP
        set Auth { badge: "degraded", state: fail }
        gauge Auth { label: "fallback TTL", value: "5 min" }
        note Auth "sign from cached session\nno rotation, no new rt"
      }

      step degraded "The app gets a 200 it cannot tell apart" {
        desc: "Deliberately the same status code and the same shape. A client that behaves differently on a degraded refresh is a client that will behave differently in an incident, which is precisely when you want it boring."
        flow Auth -> App { dur: 650ms, style: response, msg: 1 }
        set App { badge: "at_5 · 5 min" }
        gauge App { label: "access token", value: "at_5 (degraded)" }
        note App "200, but no rt_8\nexpires in 5 min"
      }

      step degraded-retry "The retry succeeds, on borrowed time" {
        desc: "The user's request goes through, and in five minutes the app refreshes again. If the IdP is still down the same fallback fires; if the fallback budget is exhausted this becomes the 503 with Retry-After that the happy path never has to think about."
        seq {
          flow App -> GW { dur: 600ms, msg: 2 }
          flow GW -> App { dur: 600ms, style: response, msg: 2 }
        }
        dim IdP
        gauge Auth { label: "fallback TTL", value: "4 min" }
      }
    ```

← [one polling cycle](../06-in-the-wild/03-poller-sequence.md)  
→ [claude code tool call](../06-in-the-wild/05-claude-code-tool-call.md)
