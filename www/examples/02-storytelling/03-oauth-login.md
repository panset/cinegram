<!-- Generated from 02-storytelling/03-oauth-login.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# authorization code flow

OAuth 2.0 authorization code flow, told as an explorable explanation.

<div class="cinegram" data-cinegram="02-storytelling/03-oauth-login" data-height="1170"></div>

[Edit in the playground](../../playground/#doc=nFf_ihvJEX6VYoK55E6S5d3ECQoh7MERloTY2HuEsDJcqbuk7myre-iqkXZiDPdXHiDkGfJgfpJQPT808q7tJX9pflTXVH_fV1-13leHavViVlGU3FarankxZ0m5FQrBx93z5eU8YSNuHtLOx4Xd7atZlWqKT43d-kBcrW7fV_VTl0i1qoTupZpVtlpVz57Bq6tGHFwslqDBKft_ovgUwSRLsA3pOANJwQIyYAS6r0PKuAlULjGW4MU6PnsG8_m8_P5woNwCC9VgMGdPDAg_WWLzE4hDAcaW4duja78FcQR1TpJMCmATMRw1wku5mZV8MQn8o2HpXu3ToURRpgXcOAKMft-VzC4duaTck3EYveHfl9uIOZeQkm8oSt9g3jV7irJYR92rcZgF_vJmHQE2OR2Z8u2PTPkbhu-723f6Cuv69u-pyXBV18GbkvrdOuorbja7jLXTXR28pXx7bSmKlxZe909KCihw316dYf6W8mF8L-mOIt_-8kZ_4a2kTL8qryja7ltY-9s3xKnJhuDq9XVfQl84zOfrZrm8JC23r3ryrBFXHir749Pumw-Da6-p2VDE7BOsq89oZV3Be-CayK7gxWIJH3pQVAssiu26Us4arQ_5jkESsN9F8FHXdjtXqaz6SDwhDA4ZYgKTqUCKgSFtwQtDOkbNZByZuxlwGgRUtCO5XcC1gMNoO9YHhNJ2q8v0kR9YGngDjLYUzpAa0Q9pmEnxQJm7enRFptAu1lVXuEJwgv-EPryHgBsKuqk__XADz0tLrqsZ2Cav4OVyuWfFSnM4v3PB75z0C1naQCtAI_5AXdAU1UzWZzJygmt8xMPezsniIrLH0R6znRrXBE9RwNsZ4On9j2-uC0AIbFJN8PHn_8CmEaWHyWSSGWzIYMPan604H3fg4zSDZzh49uokfZmdKKLV-3HREcXoxQL-1htDkc02Zc1QU957Zp_irFBd55S253QUIU9EP-XicnkBH__1b3g-IEQjJ7_9DCddhq-QYlJkBW0q9kac6sWgKKzRDkH8GBOedXcKSoqhhX1SjwKEGpmPKVuge8_Cs5LIS38LG5IjUfwEzc5ie1GHFAffnHRWpANlYCIGLzM4Om-cVnB0LSBsMqFxQwtM1xmMCnsgvAMvI_AxSQ_2uip1aIuz9vg6FtXUWlBvwUVAPCytm8AndzoH1VKP6NBiJu2JYYPmrgi2VcFgCXxc3iMMpxYpi49enErZpSzz4A9kHxmGC7jqavAMloLfUEah0CrSgZghxcGLVuCjUDZUSxG-wKZpixl1mvbi1FOKoXTt1TVNNx4L5Z8i7VKw_Imyi3U_7jKq7D9qtX-42lzef_z5v6Owf6fCng0KzsS16vD_dB-61zm7o4n7SEbbM9uNhZSHsfKAk7-m44ONck14N1rXyFjHV2hnPd0OY6Qw692sjJFyVZyeaXD6UoKkHYmj3PGsHJ2hPuvbBAVq9Lkw1p9DJjIdt-oZdhQbH-npRvP61dsbeF5ggO_6zz5uNQ-Z7cCbZtt79ZZmubx4CZlMynbM9ZuOXUsB2xVcnjJPWfPMDVnlrEutbdTBmgqWjs5A_sxgNkZF3-0pkzQ5lhbAYdnZpD3q6C4mU-spIG2HKe2H53RfJyY7zixrs-bfYNYDKDhfDrczKHIqg2RLOVMGR2iV95viljQXv6exUTdNjmR19PRq0qKf3kb9Jr_Tr2Vi1233nLkn95JJIWXN-osXL_Hy17iuHpkcGMKkmfS2k9_V62tFd3D2b9TsHYbt4-RotORGZ4KuKFV3E7I0BYZAeaXwHzB4W0ZSsWK_iyhNpnFwdOZcbjuaTIqCps876VvFH2VsnEB9COO-_z4cU74DNDkxl2b1pnwW9WR2oD6_I8xWR003crRtv9BmtZ-ydXaYXsH3hKqPqfl9bqqXPF_xOpsiTSe654JXUdfXz67FkZoasD_woJHOGod0w1DoUMCD2pCjYKeUD-N_Mf2PhZt0oOEEIAn2eEf9_yyKQtGQSuGBWfkzzQ_XQ79OUL1YLuHVn0cIXyy_IPoP61h9ePfhfwEAAP__){ .md-button }

??? abstract "The source — `02-storytelling/03-oauth-login.dgm`"

    ```dgm
    %% OAuth 2.0 authorization code flow, told as an explorable explanation.
    %% ---
    %% Every step carries a `desc` that says *why* the protocol does what it does,
    %% not just what moves where. The animation shows the mechanics; the narration
    %% carries the argument.
    flowchart LR
      browser[User's Browser]
      app[Your Application]

      subgraph provider[Identity Provider]
        auth[Authorization Server]
        tokens[(Token Store)]
      end

      api[Resource API]

      browser --> app
      app --> auth
      auth --> tokens
      app --> api

    scenario "authorization code flow" { speed: 1.0 }

      step start "The user asks to sign in" {
        desc: "The application has no credentials of its own to check, so it does not try. It hands the browser off to the identity provider and steps out of the conversation entirely."
        flow browser -> app { label: "GET /login", dur: 600ms }
        highlight app { style: active }
      }

      step redirect "The app redirects to the authorization server" {
        desc: "The redirect carries a client id, a redirect URI and a scope — but no secret, because anything in a redirect is visible to the user and to anything watching. What it asks for is permission, not proof."
        flow app -> auth { label: "302 → /authorize", dur: 700ms }
        highlight auth { style: active }
      }

      step consent "The user authenticates and consents" {
        desc: "This is the only moment a password exists, and it exists between the user and the provider alone. The application never sees it, which is why a breach of the application cannot leak it."
        note auth "user signs in\nand approves the scopes"
        pulse auth
      }

      step code "The browser comes back carrying a code" {
        desc: "The provider redirects back with a short-lived authorization code. A code is deliberately useless on its own: intercepting it buys nothing without the client secret that only the application holds."
        flow auth -> app { label: "302 ?code=Ab3x…", dur: 800ms, style: response }
        highlight app { style: active }
      }

      step exchange "The app trades the code for tokens" {
        desc: "Now the application speaks to the provider directly, back channel, server to server. It sends the code together with its client secret, and that pairing is what proves the exchange is genuine."
        flow app -> auth { label: "POST /token + secret", dur: 700ms }
        flow auth -> tokens { label: "mint & record", dur: 500ms, delay: 300ms }
      }

      step issued "Tokens come back over the back channel" {
        desc: "The access token returns on a channel the browser was never part of, so it is never exposed to the address bar, to history, or to a referrer header. The one-time code is burned in the process."
        flow auth -> app { label: "access + refresh token", dur: 700ms, style: response }
        highlight app { color: "#16a34a" }
      }

      step call "The app calls the API on the user's behalf" {
        desc: "The API trusts the token, not the caller: it validates the signature and the scopes and never contacts the application. That is what lets the same token work across services that have never heard of each other."
        flow app -> api { label: "Authorization: Bearer …", dur: 700ms }
        highlight api { style: active }
      }

      step done "The user is signed in" {
        desc: "The application ends up able to act for the user without ever having held the user's password. Every step above exists to make that sentence true."
        flow api -> app -> browser { label: "200 OK", dur: 1000ms, style: response }
      }
    ```

← [cut over to v1.5](../02-storytelling/02-blue-green-deploy.md)  
→ [sign in with an external IdP](../02-storytelling/04-oidc-login.md)
