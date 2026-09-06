<!-- Generated from 06-in-the-wild/10-agent-run-replay.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# replay a finished run

A finished multi-agent run, replayed. A main agent spawns three subagents — research, design doc, test runner — one of those spawns a subagent of its own, the test run dies on a fixture database nobody migrated, and the run recovers and says so. Scrub to any second and every agent shows the context, tokens and cost it had consumed by then, and how many subagents were alive at that instant. Drawn for anthropics/claude-code#24537 (https://github.com/anthropics/claude-code/issues/24537), the Agent Hierarchy Dashboard proposal, which asks for historical replay of a multi-agent run with transport controls, a timeline scrubber, per-agent resource numbers, and an interactive HTML export.

<div class="cinegram" data-cinegram="06-in-the-wild/10-agent-run-replay" data-height="1260"></div>

[Edit in the playground](../../playground/#doc=xFpfj-M4jv8qRO4KmME6qSSVrq7OPizm32Efdu8O0wXMQ2XQpVhMLMSWvJJc6aDRwH2Iu7f7dPtJDqQk20lcXf9usS8z1XZISxT5-1Ekv4weRstZNkLt7WG0HE2vx0qPfYHjvSrl5Ww6FlvUfmwbPbZYl-IwkdtqlI1Mjfolv9-oEt1oefdlVL9EzI-WI4-f_SgbydFydHEBP8BGaeUKlFA1pVdBEmyjMwiiKCfwA1RCaQjvXC322oEvLCK4Zs1PHfz9v_57pS8uwKJDYfMiA4lObTVIk2fg0bFWjZZ-CUYjmA34wjhMGkWrjV4p71if2esMfIGtCpAKHRgNAjbqs28sghRerIVD0GZt5AEqtbXCo8xAaMnCttFxdbl5QOv4hRMHB85M4GNumzV4A0IfwGFutOQf4APaQ9p2YfaOdeVGkxEzVujNDnVQlxvnQXkoBP2tXVOhhPWBZHRYSWH2UPE3WrPt0SKIUj0gqxMefCE8KO280H4CP1ux17AxFoT2hTW1yt1lXopG4jg3Ev9lvnh39Z5lvyu8r93y8nKrfNGsJ7mpLoeFLpVzDbpLlv0-mPcHWg_r-bNCSyd4gJ-FK9ZGWAm1NbVxosxgX6i8AOF2jldVKOeNVbkoo7_Q2QnWc-JQsFe-AG-FdrWxns1oTekyEOBVhaXSCI5OYo02gxptkE1eZRqbI-imWqN1wZ5Cg9Ierci9ekD48-1f_wL4mbRPWGo8HocdGU3O43JT4xJ8oRyocJZxzYUoNxlo4_mhTNuewE9K49aKCiwK6UCbcOhYYoWefENL2GnyDG18ofQWxNo0nnwZnVNGx-N04LwqSw4BpbcT-I0fh83lpqopoOlngl3USjYjm40ViJIWcADUEiVHkNLecBg5LDdjMqZQGmXYLxmiFlsMNhc9swpwHmsolfMZ7PAQjrc7FdqQFtYKr4wOVvyFoyAYHtZYmj3shYO9Vd6jDgshs90TytyTxxekZWNNRUdLqnOrav9H1qZj6KtgFVHVJUqgAOiiNS69LClE-YkTFUIwsDeVsdbsj4_4lsSSywAdjsod1EJjSd-534pmi_cT-FltNmjpN_wESrHG0kFu8LNyHkyACVqjNhIzcIZMpvS2xAgEubCWEOg-4sB9BvcBBe55A_eEA_cUyUbnGFCCwUTkBRsNoTCldNBor0rencbPPr6KtuT98triYedlI3n3RQCKSskxn2RTS-HRQYEWKToZT-BBlA0p0NIFTEGoTEXLpz_3xu74ZfQXK3yBljxNkwYdlkDa17gVegI_Yi4ahxGv90pLcnlhEfCzyH0G0ortVultCI-iC2RYi3y3F1Y6kAY5TKBCi-UBLJIi_rXQqmKPy8gvLBKmoGs3S76v9NaB4GcHcN4Y2YJlRGzmE4b6Cdw79PewFnKLjk-MZIRvAqd8V5eC4xAuU0TCJWyEIle8hFo4h_L7DO6lqu5hh1gHtNDGjw_ox0xY0XYRxx-UU-vyABT8ZgOEKp0_ry35B_ytoUDiIL8Pq1nyR-9hU5p98BQXzv_v__s_8fSMjpgUKS7AQe4bUZaHoPno_OLvO-hkAQpYo8sDSMMg5cjO9NCbUk5WmhaQF8J6uP1xpSGh1913vzIaoUxPvv-dXlM6cPfXNif4faVZqFlvragLiPa5-9jSXHwS6bCXTrA6aLOGu_THWDS-iC9DInEX_jeWJo_PKSNwd_TfcUgt-DlqGZeTC31HdEd2G9O_-H1MGtzdd7dECimH-PnHsDXXVJWwh7tfG53-TtuLiD4er5rp9CrsIlmje5p2cP4mbOD8Oe-DHifR7hWtmt7wT7rHaQ_nuuKSacWJGuEL_SwvVb5LetKPKcS9FTnCl4A2S1iNfisOIJUMPBkwj7z0T6sRfF3pr6Ta5aiFVQZWo8igokskKdcawRdwNaJcwmwyha_RgvQ9ynVhNZpOl9Mpg1sMWg6MSHOUo_3xhKjJD0lv6xM5LfbfI-0y7inHREIJ62I2rpRuPLbHFqhTeJgtltM5x-aeQoOWz59R3vXYKmU6ysG-OIS1FML1yPQo7Ii3NxZdQU9pBT4uPyRyJ17Pmmirge6sqepAvfQ8xUrMKCarUdgyxWjnhD0f7B9eOI4lLGZVBosu0cxgxqfYWFyNMpCNXcL76bRydDZsUFWlL2fsdlnrZVnnVfTLQm2LUm0Lnz7u_KHEJcQsLOpz2L5nGKbFJdilFRAA4hLWjTv0RdL-elLh4PjUeoJml8QCkXeCrS2wFLVDSULMh8vkdsGTvylLDklEfiT8r9PJdHEqfHoCMS04Ery5eEoqGq89r-NFJ-l-HPFhfWohYzWazpezLqI2ylLKm25UqvWr8xi6PfZNiSVuOacIgcfcS3zREviePFppjhiz1-lGFBODCdzuDadPLnGyp7sipQva7DlEEzUqS7kaXcgq84CgtMSazlv78hAySA5sSkX7URAw7wRw--a8FW635PsWL_rXf_sJbq7nNyDxQeUYlIi6Lg-s__GICJAd7q_uibgY8vi9UP7bDt9bfSsVc5JHpF7m7-QUb_D32ew1_j5bvM3hzz46dMZDH76-eI5kSNePBN_tnvfJAUS4GopOibUv5nQA75aLEJW9-sZZxeM8KI8SIY649sZqnPLGHgKB0E0MyzKmxkS6Od9aVPuVoxQI1qhR-IJiF8vNBG57JMuL5jsbIjwo3NOLjbHLWO2JyZxIJYsuw6UL08E0XWIaOA0ladBttQeDI9N1_Digu8ynl_icB_TWUsJC6S0ZpVJSlrgXLwjgJymtd-qR1s4YKqzsHxKs5CtvCNb59Zuibv7aqJvPXxt28-tXx93sw5mlTnxmaKkfLp6Wem6Mp5V-kuTgq9H0w3I2bwl4bYXOC7DoG6tjwYoIM_Kd86aO1_V0pa3rIXY-Dl-h3Z7ESVMtuJBBio_RYsP1hVhITGWi4O9ctDtORifwo6Gw74I7pfIZrBvfVrbyEoV1kbXZdm5J-bTQByiFRwu1USEx6JXWCBVo4bH-RZDFaTevLtqITN7iAcfx3PVzXpL_Ftu_By6IZzDvYYJrQeGaQCFL8WzR1Ua7NlEdxp_TwCFcbW34B7jiCq-3QsUg6tAnowSKUvBvf_b54DQEOuRxg8nwYynFowIvAyjy8DcA1Lub12QT89mb0-f2UGZXfYZ4Icx9eC3MLWavhrmrxWtgbiABexbO3QznMtzIWY1m8-VVd8PAzzVqR2Y3upcK9PP5VKU8R7WumBMKnyEB2RemRJAmb7hWqTRDV7pecFU5QBNXNWc3y8XiDFhFQFQQdhu0EOJ1peHYQFjCbHaz67dvQorzt0ZYVsP193ChAae4I6Mk9pM4vtiYelziA5ZtFGh8QAsO0aW-UlsgD9gUMiHuCGDNXw4Q6XNKygjU_T5UvS1msG4rr5jqnBTiZhNPNRqPEjhVoUz1416xl0wcar2xwxErzF5Y32ZlKE4B-Ibi5LErV_SIs_ws1LClyd2lISYah_vWmDRMKjmcprH6qPEx-O33EmeL6RT2xsrHgJd-8ATyPo247Ra7u5x96i53LnMGu-0i57NzIHomCnMYPgOFB2UpZnp4-MxlDAP6bPJ-9oSu5-D7FYHVi7S87PL4arIYWsq56w_i7zllDQkOIPCH58idf_D9kzY81zNAVASKrzDAuWNMrx9R0ycWjsJPG6FKZvnl9F1LLv2BgfOG_0mf_5xeek2BXukro_wzLyLZnE0PEBZvkXJgDWitsfRRYpZQS-P2zTIi-TZ0SEMam1ts62UR8ShrBy_WJXKbJbCCqOtSoQzYPuGe5eONnseaOsecEEt2SoZOXHkI2XXs7gi3C5wgzQm8z78F76FLcIbuoRtNgUMoL7ZCaedjA3MP1jT-NOk-gvjj3kOC3v5nPv7yl19-ug11P-aPT8Gan8iaj6H9-wT2XUPtqKZ9Cuxpc8-8wA-KhFZhT4Ld5YSAjsPmzKZDUXyetZ2JDQTt1e5JqUGomb-SQzhSXyu7WM5u-vn44k38c3P1hK5_EnO82gNm84uXqhmC8evdyxczXPwd1HIE4tw_jCfbtR9smNgQPhUCeMCCy4nncP3RmzqBXOxSdcMgKmeYDkMTXEZYpimkgIfCx3-HBDkjTeFBLsoy61p9cTgrdRldTdjJAw-KMBTlBH5TvggDPN0cTmpQ9vuRlKZz4m5x3agyjru43JqyXIt8l4Ha8PtQ9yjEA6a5iPiLRwoeJm9c1xEGStnTaa1GifBa9PxTZJa8wHyHcrXSEUMnk0kLtYv5f05n0GiJG6VRfmJaWq00flYeZiA2Hi3MFgHEXOjdKkmZM5dEB-pe3h74PpxK2yfdzbjMI5p1qU4VitJqoOT9axjOiyMSaQ7Pe6xqnyYHeDAmzpQdtV-JgozG5ela2i_2U4IwisZE7pzS2x6h1w3fOzfW6PZ6xISXLpphPIQvoFfXy3fTwOM8_BdCK8yL5KbRnmctcmucgzXX2MJmXMfhR4_57hqrY8Hlunq6UJJi59hrrp5P4dEOy3Q4Ianh2T8dWvZvIO6k8w8wny2i936Drb-h_RRxSV-Yx8lgCh3v9gt7CaCeLrj9Q5KBsLxHbnnzV9_yOLpeybLsmP0EfP4Glp0f88CQrn_W_Wz-SpadX128VM1Q3rXYvXwxA12bD49oOWJZ40XpuJTYbzKEgplXPNrZo7mc2decY-x_aOx1u7So4lBhN7zFb1ycfU6zKAFL00xjwE8eaOTWAlPcHoVNytqBPeXj3I3RhJNhFDZU5zxuRO7TAAxPFzPwux1p2RjLjGX7U71LEGE2NNy6uJuZxnFkxoaqUUaiIaInZgvDh2QwbYjXK6OVp2S9f43i-SMu4ORWrTGO-qQ8IbRmWujt7NcOAg6DcOoX9BxgeCQnS4PpPXC7-SaaDQ0b_P_U-NnBXl3jnw_05Z5TA1q8Py9Yn1uPo-CZXx0Qj9Y-gaMj47eRN_r6-9f_CwAA__8){ .md-button }

??? abstract "The source — `06-in-the-wild/10-agent-run-replay.dgm`"

    ```dgm
    %% A finished multi-agent run, replayed. A main agent spawns three subagents —
    %% research, design doc, test runner — one of those spawns a subagent of its
    %% own, the test run dies on a fixture database nobody migrated, and the run
    %% recovers and says so. Scrub to any second and every agent shows the context,
    %% tokens and cost it had consumed by then, and how many subagents were alive
    %% at that instant. Drawn for anthropics/claude-code#24537
    %% (https://github.com/anthropics/claude-code/issues/24537), the Agent
    %% Hierarchy Dashboard proposal, which asks for historical replay of a
    %% multi-agent run with transport controls, a timeline scrubber, per-agent
    %% resource numbers, and an interactive HTML export.
    %% ---
    %% Honest scope: this is the replay half, not the dashboard. Cinegram reads no
    %% telemetry and knows nothing about a session that is still running. What it
    %% compiles is a record of a run that already ended — into one self-contained
    %% HTML page with a scrubber, a step list, keyboard transport and narration.
    %% Every number below was written into the `.dgm` by hand from a transcript;
    %% none of it is sampled live, and the page will say the same thing tomorrow.
    %% ---
    %% The per-agent metrics panel is `gauge`. Different gauge labels coexist on
    %% one node, so a single agent carries `context`, `tokens` and `cost` at once,
    %% and each write holds until the next write to the same label — including the
    %% mid-step updates here, where a value lands at the moment the work landed
    %% rather than when the step began. Because those windows are exact, dragging
    %% the scrubber backwards does not merely rewind the animation, it restores the
    %% readings as they stood at that second of the run. `set` badges carry status
    %% (planning / running / failed / passed), `dim` keeps the not-yet-spawned
    %% agents visibly out of play, and the broken query is a `status: fail` flow,
    %% so the ✕ lands on the database that actually broke rather than on the agent
    %% that was only doing as it was told.
    flowchart TB
      session[(Recorded session)]
      main[Main agent]

      subgraph spawned[Subagents spawned by the main agent]
        research[research-auth]
        design[design-doc]
        tests[test-runner]
      end

      scan[codebase-scan]
      fixtures[(Test fixture DB)]
      summary[Run summary]

      session --> main
      main --> research
      main --> design
      main --> tests
      research --> scan
      tests --> fixtures
      main --> summary

    interact {
      click tests -> step trace { label: "Why did this agent fail?" }
    }

    scenario "replay a finished run" { speed: 1.0 }

      step open "00:00 — the run is already over; this is the record" {
        desc: "Nothing here is live. A 41-minute session ended at 14:02 and what plays is its transcript, which is why this has a scrubber rather than a refresh rate. At this instant the main agent has read the prompt and has spawned nothing."
        flow session -> main { label: "replay: 41m, 4 subagents, 1 failure", dur: 700ms }
        dim spawned, scan, fixtures, summary
        highlight main { style: active }
        set main { badge: "planning", state: busy }
        set session { badge: "ended 14:02", state: ok }
        gauge session { label: "elapsed", value: "00:00" }
        gauge session { label: "run cost", value: "$0.04" }
        gauge main { label: "context", value: "8%" }
        gauge main { label: "active subagents", value: "0" }
      }

      step spawn_research "02:10 — the first subagent is spawned" {
        desc: "The main agent delegates the reading it does not want in its own context window. Two nodes of the tree are now live and their meters move independently from here on."
        flow main -> research { label: "Task: how does RFC 8628 device flow apply here", dur: 700ms }
        dim design, tests, scan, fixtures, summary
        set main { badge: "waiting", state: busy }
        set research { badge: "running", state: busy }
        gauge session { label: "elapsed", value: "02:10" }
        gauge session { label: "run cost", value: "$0.11" }
        gauge main { label: "context", value: "14%" }
        gauge main { label: "active subagents", value: "1" }
        gauge research { label: "context", value: "6%" }
        gauge research { label: "tokens", value: "5k" }
        gauge research { label: "cost", value: "$0.03" }
      }

      step depth2 "05:40 — a subagent spawns a subagent" {
        desc: "research-auth wants the repository read as well as the spec, so it spawns codebase-scan beneath itself. This is the depth a tree view is for: three agents are alive, and the one you actually prompted is none of the busy ones."
        flow research -> scan { label: "Task: grep the auth middleware", dur: 700ms }
        dim design, tests, fixtures, summary
        highlight research { style: busy }
        set scan { badge: "running", state: busy }
        gauge session { label: "elapsed", value: "05:40" }
        gauge session { label: "run cost", value: "$0.26" }
        gauge main { label: "active subagents", value: "2" }
        gauge research { label: "context", value: "22%" }
        gauge research { label: "tokens", value: "26k" }
        gauge research { label: "cost", value: "$0.19" }
        gauge scan { label: "context", value: "9%" }
        gauge scan { label: "cost", value: "$0.03" }
      }

      step research_done "09:12 — the branch returns, and its meters stop where they stopped" {
        desc: "codebase-scan answers its parent and research-auth folds that into one summary for the main agent. Both agents are finished, but nothing clears their gauges: at any later point in the replay you can still read what that branch cost."
        dur: 2s
        flow scan -> research { label: "7 files, 2 middlewares", dur: 600ms, style: response }
        flow research -> main { label: "spec summary + 3 constraints", dur: 700ms, delay: 600ms, style: response }
        dim design, tests, fixtures, summary
        set scan { badge: "done", state: ok }
        set research { badge: "done", state: ok }
        gauge session { label: "elapsed", value: "09:12" }
        gauge session { label: "run cost", value: "$0.58" }
        gauge main { label: "context", value: "21%" }
        gauge main { label: "active subagents", value: "0", delay: 1300ms }
        gauge research { label: "context", value: "29%" }
        gauge research { label: "tokens", value: "41k" }
        gauge research { label: "cost", value: "$0.34" }
        gauge scan { label: "context", value: "14%" }
        gauge scan { label: "cost", value: "$0.08" }
      }

      step design "12:30 — the expensive one, and the meters move mid-step" {
        desc: "design-doc writes the whole document in one context and finishes at 18:44, and its meters are the argument for per-agent numbers: 118k tokens and three quarters of a window sit inside a subagent the top-level session never sees. Scrub into the middle of this step and you catch it partway there, because the second set of gauge writes is timed to the moment the doc landed, not to the start of the beat."
        dur: 2800ms
        flow main -> design { label: "Task: write docs/oauth-device-flow.md", dur: 700ms }
        flow design -> main { label: "design doc, 1400 words", dur: 700ms, delay: 1400ms, style: response }
        dim tests, fixtures, summary
        set design { badge: "writing", state: busy }
        set design { badge: "done", state: ok, delay: 2100ms }
        gauge session { label: "elapsed", value: "12:30" }
        gauge session { label: "elapsed", value: "18:44", delay: 2100ms }
        gauge session { label: "run cost", value: "$1.71", delay: 2100ms }
        gauge main { label: "context", value: "34%", delay: 2100ms }
        gauge main { label: "active subagents", value: "1" }
        gauge main { label: "active subagents", value: "0", delay: 2100ms }
        gauge design { label: "context", value: "11%" }
        gauge design { label: "cost", value: "$0.09" }
        gauge design { label: "context", value: "74%", delay: 2100ms }
        gauge design { label: "tokens", value: "118k", delay: 2100ms }
        gauge design { label: "cost", value: "$1.06", delay: 2100ms }
      }

      step tests_fail "21:05 — the test runner dies on a fixture nobody migrated" {
        desc: "test-runner is spawned, reaches the fixture database and gets an error on its first query: the migration that creates the device-code table was never applied there. The ✕ lands on the database rather than on the agent, because the agent did exactly what it was asked to do."
        dur: 2200ms
        flow main -> tests { label: "Task: run the suite against the new routes", dur: 600ms }
        flow tests -> fixtures { label: "SELECT from oauth_device_codes", dur: 700ms, delay: 700ms, status: fail }
        dim summary
        set tests { badge: "running", state: busy }
        set tests { badge: "failed", state: error, delay: 1400ms }
        gauge tests { label: "context", value: "4%" }
        gauge tests { label: "tokens", value: "3k" }
        gauge tests { label: "cost", value: "$0.02" }
        gauge session { label: "elapsed", value: "21:05" }
        gauge session { label: "elapsed", value: "24:18", delay: 1400ms }
        gauge session { label: "run cost", value: "$1.83", delay: 1400ms }
        gauge main { label: "active subagents", value: "1" }
        gauge main { label: "active subagents", value: "0", delay: 1400ms }
        gauge tests { label: "context", value: "12%", delay: 1400ms }
        gauge tests { label: "tokens", value: "16k", delay: 1400ms }
        gauge tests { label: "cost", value: "$0.11", delay: 1400ms }
      }

      step trace "24:18 — the frame that replay exists for" {
        desc: "Stop on the failure and the picture holds still: which agent, at which second, on which call, and what it had already spent when it died. Without a record of the run this is the part you rebuild from scrollback, if you still have the scrollback."
        dur: 2s
        focus tests
        note tests "migrated fixtures? never checked\nSELECT ... -> 42P01 undefined_table\nexit 1 after 14 errors" { side: right }
      }

      step retry "29:40 — the main agent migrates the fixtures and respawns it" {
        desc: "Recovery is a second attempt by the same agent rather than a new one: the main agent respawns test-runner with the missing migration put in front of the suite, and it passes at 36:50. The cost gauge keeps counting across both attempts, because both attempts are what the run actually paid for."
        dur: 3s
        flow main -> tests { label: "respawn: migrate first, then run", dur: 600ms }
        flow tests -> fixtures { label: "migrate + 214 tests", dur: 700ms, delay: 700ms }
        flow tests -> main { label: "214 passed, 0 failed", dur: 600ms, delay: 1600ms, style: response }
        dim summary
        set tests { badge: "running", state: busy }
        set tests { badge: "passed", state: ok, delay: 2200ms }
        gauge session { label: "elapsed", value: "29:40" }
        gauge session { label: "elapsed", value: "36:50", delay: 2200ms }
        gauge session { label: "run cost", value: "$2.11", delay: 2200ms }
        gauge main { label: "active subagents", value: "1" }
        gauge main { label: "active subagents", value: "0", delay: 2200ms }
        gauge tests { label: "context", value: "23%", delay: 2200ms }
        gauge tests { label: "tokens", value: "34k", delay: 2200ms }
        gauge tests { label: "cost", value: "$0.29", delay: 2200ms }
      }

      step totals "41:12 — the whole tier, and what it came to" {
        desc: "One highlight naming the subgraph lights every spawned agent at once, with each one still wearing the readings it ended on. That is the artefact the proposal is asking for under HTML export: a page that can be scrubbed, stepped and read afterwards — not a monitor, because the run it describes has already stopped."
        highlight spawned
        flow main -> summary { label: "4 subagents, 1 failure, recovered", dur: 800ms, style: response }
        set main { badge: "done", state: ok }
        gauge session { label: "elapsed", value: "41:12" }
        gauge session { label: "run cost", value: "$2.19" }
        gauge main { label: "context", value: "47%" }
        gauge summary { label: "total cost", value: "$2.19" }
        gauge summary { label: "failures", value: "1, recovered" }
      }
    ```

← [order placed: payment approved](../06-in-the-wild/09-animate-the-flow.md)  
