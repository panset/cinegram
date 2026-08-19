<!-- Generated from 05-ai-systems/01-agent-tool-loop.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# one question, four turns

One question through an agent's tool loop: think, call a tool, get a call wrong, retry, answer. The loop is not a metaphor here — the same agent node is entered four times, and the token and iteration counters say how far in you are at any moment.

<div class="cinegram" data-cinegram="05-ai-systems/01-agent-tool-loop" data-height="990"></div>

[Edit in the playground](../../playground/#doc=rFjNjtzGEX6VAgHBEjI72ZVs2RkfBJ1jJEHswIeloC2yi2RnyG6qqjkUIQjIQyS3vEfueRQ_SVDdTe6sNLIlIJfdme6u6vr56qvqeVecisPNriAXeCkOxfU3V2ivZJFAg_z--uYKW3LhKnjfX_Xej3vTDsWu8CO5zz_d2J6kONy-K8bPFwrFoQj0NhS7whSH4tEj-LMjeDORBOsdhI791HaADqL8VwKqAVTDAUJn3XEHNfY9YNzYQUsBMC6V7tEjmNm7dgdMgZcdoJOZeA8_dRRVgBVwXgUGCjh2nqEjJvjlH_-E0BEIDpQuBucNRY1WgFwgJgONnxiCHUhUtYkiwR_JxW82EGP0ovaTSggILtD5GRpksC6qW_wEyAQYAN0Cgx_IhX3curq6iv9fZuc3k_WeGoUAQQIGW4Ox2DIOujl7lgAYkpeVf0sCjk7EUVfdoWvpe5g7DPmLqJRaNagB0Z1oVYcngorIAfspeTdEx1Q2KlM7epSQciJozR5-9CkMs4_paQUQmNAQA9Zhwr5fYEYXZA1yCuoWKzcNlZ49D6eAjOq_Smis7lqcWrqLzhMwho4YQocpog456dqSsujJJZqBVU8wa5JPUYZAap4qvdIKjMhHMg-jr1Fs0PZk4M6zIZbXprpLoLPq3Z2aMckhnrqDpvfzLqNKQeId7eGlfozqkNnPSbD2vQKo7ryt6fsP9QzIx5RqMi0BymqFOmXYj5L0wS___peiR08aLRuXnZ87W3cREZrrAY-U1MVagJ5aq7HAlKDAy750anvdIQf44a-lA5iE-PHtX4jFu1dPdCUC8fZlhOMP3o-vSqfLMlUt49hFJMjtT_r3lW4ACCHX3e1M1ev0Ma-b6vbxFs8ncZGcSepSnd6-jP_yFWoLXF2V0_X1s1yUm0H36-mKCxumurCY7lH9UpNDth7Kwp8R0C7X-MROygLeKRDJHOBmfw3vs-uBRkA5QlkoVDbqQmZ7oiiVHSapD1AWZVkWP3cLGGuAqZmcUXzbY66lmej4Qs8kkGz6nAexru0pVVutlBjNP4AN4IiMHs_l07AfEiIwYBWpwplUieJd2vZTEGsIbDgjRHprJQhUVOMkFFUM3lC8TzF9dH7WUgswuWB7vbpDgcBW66ZIniqMcr7O0wXvoMeKeg3CfCkAL8piB2biA3x7fT2IRjghKGwKKjQtqYLI_da1KhJ54ADVJMsqExni42s3nlGxE_YTHeDmN2QSBZ0JlMXN_uZYFknuHAZjjy7jIEXNUG0N5RIMXmN8hGmExrKEj6HxJx85E3rCU67WkX1NImC1GVqJ96R8pRuU1CRTpZJw3dGAsoPR1kdR8kksOLMNWSNyO2mTSQysm8qFS-5cvknpTUhRY6xAZXtlnooaz7HrnAFR08-T25Lf2bbrbdvd50zC0n-QHufDGumySKbeE0RZOuRWDvDuzQG0EhJIcpFEYIouv1-v_JLE_eFi4tLFOXUxOWuV9T3IVNdaXh_nS09n0XicKTKFJk1y4ONHjWHdEY6KAaZY0ynyb0dyYk8aS6iReTlkjpapD1B7bcYV1kewLqTOWnunAxPM1hk_Ry3a3WJumaDxvOZzYpdy2fi-97NsKcpdjWnsFzXNUI-L5ncBeot16JfU2sm1oUuIoNXsHcjW4TdtHfaKV98o4NZxge0Y0ZlbzNnifcNOeyNaBt9s6mKPlFgFCsDc3OIwZp0EjAPSRjSZ0x80gAdUswHr8ZsnG8M8P2eYqCdLfpKynuesyEMduxS-7VtGO5OM3gl9Obl8vX92EaODlcYybbCrvTPbGDJg33geyFwGaeIKlGOcWjkNH9OQ8WE8pTk4sr-yi5V1zvRsrENe4vQxcVIVMaGA1x6hLJLGnYojhaRBXJObmeasjVh30unZbNwTcuf5SknS9wR_91W824NMfNLSsOFhZ3mYcFOdx_PNRLw8ToTxOviA_ZOHTSV1i3XMesBHpoKy0DY71V2OzwHOVZWl056lw_FA6NY9HHS-f_EbbHSp9Tz9cnR8s7--iI401aVsE7NnaP09d2yhXvHCpEX3KbgkBZFm8gtJZlQOIbPTlm8FGjKQGWkF19aFvpLELYokDGkETZSSUYsh0DAGcDisQ2nsGGvQV1AoBbAjo2Wvo8pGA7V3J2JJT4ZzPqkWqEhJg0kCcvhwKvls7KSk_hpfmOrTXPH1zVOIHPbf_8Dvnn336P9JGZeA9OzLgfR8__VFIKWhEsriZxs6qHzooME6xAmkO69YkKCPkIsASrOkDnZzp4ywoYQGGyQjy0nQ9uCb_HiP4NzDOgRRL5TuUMgllkkmVpPRZ_6skwdh3ZGJz_bd2T1iB21uHRogF39CCD67dvYyigTnem17VtHnjI2Imj2HTh2ybRyBGs8bjCa3zaOfxFWO4VkCFAQ7MPokiCCNSO7sOKr6H__4tw0g3_0qJNb3qHXYQ-SkWF1sQ1C9WuzYhPyyvYum3kGtVSRnTLvpitP01tHzzxQr7yaJGPDYJWI6VyX7L4Xbt_unR13AcICn97X0vnTF-1fv_xcAAP__){ .md-button }

??? abstract "The source — `05-ai-systems/01-agent-tool-loop.dgm`"

    ```dgm
    %% One question through an agent's tool loop: think, call a tool, get a call
    %% wrong, retry, answer. The loop is not a metaphor here — the same agent node
    %% is entered four times, and the token and iteration counters say how far in
    %% you are at any moment.
    %% ---
    %% An agent loop is the case a static diagram is worst at. The boxes never
    %% change; what changes is how many times you have been round them and what
    %% the last tool said. So the two things a reader actually wants — the
    %% iteration number and the tokens spent — are `gauge` state rather than
    %% narration, and they stay readable wherever the scrubber is parked.
    %% ---
    %% The failed `orders_db` call is a `status: fail` flow, not a red one. A red
    %% arrow is a colour choice; `status: fail` marks the edge as failed and drops
    %% a ✕ at the destination, which is what makes the retry legible as a retry.
    flowchart LR
      user([Person])
      agent[Agent Loop]

      subgraph tools[Tools]
        search[web_search]
        db[(orders_db)]
      end

      answer[Answer]

      user --> agent
      agent --> search
      agent --> db
      agent --> answer

    scenario "one question, four turns" { speed: 1.0 }

      step ask "The question arrives" {
        desc: "\"Why did refunds spike last week?\" is a question no single tool can answer: it needs a number from the database and a reason from outside it. The loop exists because the model cannot know that until it has tried."
        flow user -> agent { label: "why did refunds spike?", dur: 700ms }
        set agent { badge: "thinking", state: busy }
        gauge agent { label: "iteration", value: 1 }
        gauge agent { label: "tokens", value: "1.1k" }
      }

      step plan "The model decides what to look up first" {
        desc: "Nothing leaves the process in this step. The model reads the tool schemas, picks one, and writes the arguments — and every token of that reasoning is billed before a single tool has run."
        highlight agent { style: busy }
        note agent "picks web_search\nargs: {q: \"refund spike causes\"}"
        gauge agent { label: "tokens", value: "1.9k" }
      }

      step search "The first tool call succeeds" {
        desc: "The search tool returns prose, and prose is cheap to request and expensive to carry: the result comes back into the context window and stays there for every turn that follows."
        %% The reply is delayed by exactly the length of the request, so the two
        %% halves of one round trip read as a round trip rather than as a pair of
        %% arrows leaving at the same instant.
        flow agent -> search { label: "web_search(q)", dur: 600ms }
        flow search -> agent { label: "6 results", dur: 600ms, delay: 600ms, style: response }
        gauge agent { label: "tokens", value: "4.3k" }
      }

      step misfire "The second call is malformed" {
        desc: "The model asked for a column that does not exist. This is the ordinary failure mode of tool use — not a broken tool, an argument the model invented — and the loop's whole job is to survive it."
        flow agent -> db { label: "query(refund_total)", dur: 700ms, status: fail }
        note db "no such column: refund_total\ndid you mean refund_amount?"
        gauge agent { label: "iteration", value: 2 }
        gauge agent { label: "tokens", value: "5.0k" }
      }

      step retry "The error goes back in and the call is repaired" {
        desc: "The error text is not swallowed, it is fed to the model as the tool's reply. That is why the second attempt names the right column: the loop learned inside the same conversation rather than by being restarted."
        flow agent -> db { label: "query(refund_amount)", dur: 600ms }
        flow db -> agent { label: "412 rows · +38%", dur: 600ms, delay: 600ms, style: response }
        gauge agent { label: "iteration", value: 3 }
        gauge agent { label: "tokens", value: "6.4k" }
      }

      step answer "With both facts in hand the loop stops" {
        desc: "The loop ends when the model emits text instead of a tool call. Nothing else stops it — no step budget was reached here, the model simply had enough to answer, which is the only exit condition worth designing for."
        unset agent
        flow agent -> answer { label: "+38%, driven by the shipping SKU", dur: 800ms, style: response }
        %% The final total is written back after the `unset` clears the loop's
        %% badge, so the counter survives the step that ends the loop.
        gauge agent { label: "tokens", value: "7.2k", at: 200ms }
      }
    ```

← [leader failure and re-election](../04-diagram-types/03-raft-election.md)  
→ [fan out, barrier, synthesise](../05-ai-systems/02-multi-agent-fanout.md)
