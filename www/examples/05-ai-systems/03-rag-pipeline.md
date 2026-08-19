<!-- Generated from 05-ai-systems/03-rag-pipeline.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# rag pipeline

Retrieval-augmented generation, told twice: once with a warm cache, and once with the cache missing and a live web search standing in for the corpus. The second telling ends with an answer whose citations do not come from the documents anyone approved.

<div class="cinegram" data-cinegram="05-ai-systems/03-rag-pipeline" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=xFndjlu38X-Vgf4w_gmiVdZZZ2OoF4UbpEWBJEjdtLlYGdjR4UiHFQ95zOHRsWAYyEP0GfoKve-j5EmKGfJ8aFd14iJAr-wlD4ecr9_8ZvR2cVysny4X5FM8LdaL68-v0F7xiRM1_On1zVXE_VVrW3LW08rsm8VyEVryv_DTnXXEi_Xd20X7C0-kxXqR6E1aLBdmsV48eQIvKUVLR3RX2O0b8okM7MlTxGSDX0IKzkDqbUVrCL4i6G2qAaHH2ECFVU1LQG90b-OfPMn7qaa8CY1ltn6v3yA4eyToaQtMGKsaOKE3sm097ELM50JsO17B93UWyFQFbyCRc_IlecPlER7Qc08R-jowQWWTPprBBPAhQRUagl0MjchVWSZUnSjJgP4UPAG2bQxHMivdvrq60n9_F5K8LURLDFxjJJFgI4hzrN8vgQPYBJahjzYl8qq_vhmOGC36BJFahydWgaLXFpmAK_IYbYBUx9Dts6XuXQiHrr0HTtTCTz_-He47n6xb38sN1leuYzGc7KA3g0QPxh4p7omX0Ne2qqEh9AwIVY1-T5AC1KEHhNcdxZPIomZLxpCBCj14OlIUe6rArgVDXEW7FStPDqxtUueJQc_dem6z-9AlMfgadmjdPQSvnw_WsPouh7YB3IYuy8ruW6rscVWlDXG7BnnkCZji0VYExhqwieFvYZvjTsRE4s7pFZysc9CHyOoMlMXRAdYnisRJ9JM3dlHWAOHliz9Azpsss3i2PumxHTq3xeoALaY6RxHJc0g0Gv0Zdvqu0HuImGqSWJYAhV0IyYdEq43fudBXNcYEX7_ceMhu-ejuL0wR_iT_f_WxLKuT7r7KroqvZCnmLKW7v1KVQoQ_a_rolvrj7qPvkBn3BF_Knx_rTk_bux9oWz6G3xc9isCI_nD3ZQzMV1_5KhiK8FIXy5XITM3W0d13MTRtghfl77xbIILuXuQM_CYYcvmcLpT1Vxs_KgpXV5vu-vqGsoKjptP6oOVc42lXFb281dP28kZWc1J42hnUm6s67Q7qzVWdnVXdRLXR_ZvFBIibBbwFbonMGp6uruFdNoImN_IBNguBidedhGLwsCVJGwmmozpXjsv3oPm4hs1ixOiSskzEGpp9iCYj5ZDZEVIXfd4db7A-BUBog5VEzHnJ2BBwixXNQBd6FLwx9IaMnsr5oCmYaskbE3rPKRI2kiJ7Cg2leMogW1MkCH61WeTnS7gPrp97Ht6Cwy05UU2STHK6qqk6CADsKQG70FP87WaxBNPFNXxxfd2wmFGk1nZfO7uv0yiM08nRGrYdn_JHc4NnaC02HwIk5vt4Bmhb2oUM89kAj73w_WC1wa6sj0U-kIEgQrUOHgXQtNr54hStbwoWCSXKDnRawdchHErhmx6xs5EVe3SpJmyJBRMzIlseYVHx24ShIuT6EtFXUlAwgaHKGmKF_4KQKUMU7APxuYtKFj5IwrmXnn5-c3tlxvgsbrmdu0UqgCGHp3uogzNZhWJ8RU-taLkkZAyrkQGr1KFzJ8AY7ZGMKDTKk29TLzWsZUCNLVG8Y3RS4axfjtVX-IBED8I90-t7ySnsmEZRPmQjkmMaM0DCwxMZBqbXHfnK-v1qZpcJTuYINLfLzvo9xTZKXj3f3VQ__fiP0Tqfi3WWoDY5s9U8OqW45sgqxRaz4dqM5gzoIqE5PY7GF3qWD7blKWiBfLKR3Gkta0xAmiijtF4ydFDLaORwFSIZJV8InjBeWSNSKnQTgDTWd0lesy-IMNhziN6MHvL6TD0MsBVCdBZnWcX3xdnz6an_-qdSzPNgWw65Honb4JmG6GNKo3e2aPYk0mqb5DgnTEJeD0uoggtRtv7v6S3ePMPNYhCwx25_-U3Di0TUEV1Ha3j-2JGlvAwwk-uokryeoQk5-1JNDWCPF9xZ6nqhxZYhUoXO_SabV6s0lSotMBCpsmyDX8EfhWiiYSCs6sF8gHu0ngvJ0hybnCn-OxC1DLvQxeWQK1IgcqXn1O12ZDLJ1pAQtifu17rHmV8N_IbrEJOk5rmzHyVPsdCZtyth_wZTtu5jUBm88vDkgdo088dm8Qw-heeDN8_rbSnt2TNFQ8uw7azTYhiioXgJ7YV659wS3NmT76wndypgkSw62BKOBXVirGcJXONRifh2xJ3yhlnBKSuC6VsCJp9W8KIgGeOJgYMwfCqfD2CuFUF6QHCUFEu1KHEShpnCnoSGjl558kR5glBRYKWqXWJrsshylzY13GIhEH0dHE1NifX6vU3LUSLm7Np1rlwPVeA01C-0Hp6rP0uN8tq9zYk7ivEHaX3onAEXOD8qhQOJwM5fspZQFTVVPq56zbyd-Tx8As8mX3wyJsFm4z9bPT_kK3gwEdPrIQzGKM6s8QFpnMdiCi08O0f9IXyLkIlePmCXczESY1mzy7IeRfYoJEd2IwR8zFDlY732P9lSe3sk_zjKv9LmKrdlJTxLSy2hH7EiZSwpaA5kCFPYGI1aGKKAz9B_g8emENSh35ZnVFi68dKZFfHW2XQaMi3XL4kLDtNEoMGqtl5e2rWZfI5RPbHBmVEfEcLiiInJnxH5uRvKioTNOE4YHfL8PTVIXVNybGqRB70y205BVBh5UGFHmf0rR6KmTafVWVtRBWeGOUtP27Eb1SajNNfrB-3HMl-yLjcs4aw1P4sifeRDBlKq--Nw-Vb6dA8UY4iFewIdrWZ-aUO8tAN9pjFRYiWmUxnhKH8Yc3-ksj70mfcEIIz-HD41KN605HUA0uNpqaOiGLxCIELHef6jDDuTJejRJuVyH8Y-fMjPf8w5MHU8GW-EmyzyIg3UkcYudN5sNl5NRAaeXasJJri5QFrEITPWInfOeYupPrv97PZSpRvHFNmbmQ_qSAQd6cBKIZ22eegmfd7lHmfWCjqUdqT0gzMIHuZLksZKHrJLpW-TBlFvO54xmki5L62C3ynBXIKNkRwd0U8ENcfQqEkUWKHzSjUM7GajPAnEPP_SLssUblTGSurq08-QEzHLLBJ0TJlffrkJVSly6H3xdAvtQBwnCWNH8MV7Ce0H8dHbx9HQ0_bqMifN1C_VkUZAZ_u-dnc4uQQszK8Nwa3g29IDaJ4Ld0nkXI6J3HyMSTwiPyBsXdifTciUB3R-G8Lh4SQiczQ9KD7Wp0q7wgpsQG-wStI7MuxDMB_MP29_Pf55A5_CxawUP_wMB821mraQ6E267IdBQpQXS1Q3GA-gieDRVzRU-Twe0pG9jlPHtJ_6Se2z1X7iO2cPVCasQV1SPpREzPw300700PkqNALnyghKnRxnVx6wS3WIWjOPdMUCfXLDvDH4jxztRvVvL5G0p78aSbv5H5A0dX82VfHkyK5ahx3bbMs8055-v8CoRB1CF_lyRGjZ2yvtn34FkX5tFDIb3c3IW5OngAwo0CgWBx-2wZwy_bOC0E2L_gSRjpZ6Miv4BuNh-E2gXDVP4NqaYXdE7rFxcieog5eGhTQeHrYjhgRr40kMQG4HJhDn2inFdsj6cmsOXl7BDwIy8gXrfH82L8pU9oxfTEiSAjR4mBVDjH4czuXg9OZM3JGisdU4kht42f_nQf-DnzxW_yXVzNH_YXRzSKYhtsbTaxUmht9sFCqG37_GOp4BNvvh3cYv3r169-8AAAD__w){ .md-button }

??? abstract "The source — `05-ai-systems/03-rag-pipeline.dgm`"

    ```dgm
    %% Retrieval-augmented generation, told twice: once with a warm cache, and once
    %% with the cache missing and a live web search standing in for the corpus. The
    %% second telling ends with an answer whose citations do not come from the
    %% documents anyone approved.
    %% ---
    %% Both stories share their opening, so it is written once. The variant replays
    %% the base scenario through the `lookup` step — `until:` is inclusive — and
    %% then diverges, which means a change to how a query is embedded can never end
    %% up describing the cache hit and not the cache miss.
    %% ---
    %% `outcome: fail` on the variant is a claim about the answer, not about the
    %% pipeline: every service did its job, and the result is still worse. That is
    %% the interesting failure in a RAG system, and it is why the fallback path
    %% deserves a scenario of its own rather than a footnote.
    flowchart LR
      query([User Query])
      embed[Embedder]
      retrieve[Vector Search]
      cache[(Passage Cache)]
      web[Web Search Fallback]
      rerank[Cross-Encoder Reranker]
      assemble[Prompt Assembler]
      generate[Answer Model]
      answer[Answer]

      query --> embed
      embed --> retrieve
      retrieve --> cache
      retrieve --> web
      retrieve --> rerank
      rerank --> assemble
      assemble --> generate
      generate --> answer

    scenario "warm cache" { speed: 1.0 }

      step ask "The question becomes a vector" {
        desc: "Retrieval never sees the words. The embedder turns the question into a point in the same space the corpus was indexed into, and everything downstream is geometry from here on."
        flow query -> embed { label: "why did checkout get slower?", dur: 700ms }
        highlight embed { style: busy }
      }

      step lookup "The retriever checks the cache before the index" {
        desc: "The same questions get asked over and over, and an embedding is a stable key. Looking in the cache first is the cheapest thing this pipeline can do — and the branch that decides how the rest of it goes."
        flow embed -> retrieve { label: "1536-d vector", dur: 600ms }
        %% `delay` holds the lookup back until the vector has actually arrived —
        %% the two hops are one causal chain, written without a `seq` because
        %% nothing else in the step needs sequencing.
        flow retrieve -> cache { label: "fingerprint 8f3c…", dur: 500ms, delay: 600ms }
      }

      step hit "The cache has the passages already" {
        desc: "A hit skips the index entirely: these eight passages were retrieved and scored for a near-identical question minutes ago, and nothing in the corpus has changed since."
        flow cache -> retrieve { label: "8 passages · warm", dur: 600ms, style: response }
        set cache { badge: "hit", state: ok, color: "#16a34a" }
        gauge retrieve { label: "passages", value: 8 }
      }

      step rerank "The reranker throws most of them away" {
        desc: "Vector search is recall; the cross-encoder is precision. It reads each passage against the actual question and keeps four, because a prompt stuffed with near-misses answers worse than a short one."
        flow retrieve -> rerank { label: "8 candidates", dur: 600ms }
        gauge rerank { label: "kept", value: "4 / 8" }
      }

      step assemble "The prompt is built in order" {
        desc: "This is the one genuinely sequential beat in the pipeline: the passages have to be in the prompt before the prompt can be sent. A `seq` says so, where the rest of this file lets actions start together."
        %% The note sits outside the `seq` so it spans the whole step — inside it,
        %% a stateful action costs the chain 800ms and then ends, and the reader
        %% would lose the token count before the prompt was sent.
        note assemble "system + 4 passages + question\n2.8k tokens"
        seq {
          flow rerank -> assemble { label: "top 4", dur: 500ms }
          flow assemble -> generate { label: "one prompt", dur: 500ms }
        }
      }

      step generate "The model answers from what it was given" {
        desc: "Every claim in the answer is traceable to one of the four passages, and each citation names the document it came from. That traceability is the entire reason for the machinery upstream."
        highlight generate { style: busy }
        flow generate -> answer { label: "answer + 4 citations", dur: 800ms, style: response }
      }

    %% The cache miss is the same story until the lookup comes back empty.
    scenario "cold cache, web fallback" { variant: "warm cache", until: lookup, outcome: fail }

      step miss "The cache has nothing" {
        desc: "Not an error — an eviction. The entry was there forty seconds ago and the pipeline now has to earn the passages the expensive way, in front of a user who is already waiting."
        flow cache -> retrieve { label: "no entry", dur: 600ms, status: fail }
        note cache "fingerprint 8f3c… not found\nevicted 40s ago"
        set cache { badge: "miss", state: fail, color: "#dc2626" }
      }

      step fallback "The index is stale, so the web stands in" {
        desc: "The corpus was last indexed before the change that caused the slowdown, so vector search returns confident, irrelevant passages. The fallback reaches outside the approved documents — which is a decision, not a retry."
        flow retrieve -> web { label: "live search", dur: 700ms }
        flow web -> retrieve { label: "6 pages", dur: 700ms, delay: 700ms, style: response }
        gauge retrieve { label: "passages", value: 6 }
      }

      step web-rerank "The reranker keeps three of the six" {
        desc: "The same reranker, a worse pool. Nothing here can tell that these passages came from a blog rather than the runbook the corpus was built from — the scores look exactly as good."
        flow retrieve -> rerank { label: "6 candidates", dur: 600ms }
        gauge rerank { label: "kept", value: "3 / 6" }
      }

      step web-assemble "The prompt is built from web text" {
        desc: "The assembler cannot mark provenance it was never told about, so the passages arrive looking like every other passage. This is where an uncomfortable answer becomes an authoritative-sounding one."
        note assemble "system + 3 web pages + question\n2.1k tokens"
        seq {
          flow rerank -> assemble { label: "top 3", dur: 500ms }
          flow assemble -> generate { label: "one prompt", dur: 500ms }
        }
      }

      step web-answer "The answer is plausible and its citations are not ours" {
        desc: "The user gets an answer with citations, and every one of them points at a page nobody in this company reviewed. Marking the answer rather than hiding the fallback is the only honest ending."
        %% The delivery itself does not fail — the answer arrives. What fails is
        %% the claim the pipeline was built to make, so the warning is a note and
        %% the verdict is the scenario's own `outcome: fail`.
        flow generate -> answer { label: "answer + 3 web citations", dur: 800ms, style: response }
        note answer "citations: web only\nnot from the indexed corpus"
      }
    ```

← [fan out, barrier, synthesise](../05-ai-systems/02-multi-agent-fanout.md)  
→ [launch, negotiate, call](../05-ai-systems/04-mcp-handshake.md)
