<!-- Generated from 06-in-the-wild/05-claude-code-tool-call.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# claude code tool call

What actually happens when Claude Code runs a single tool call, end to end: one prompt, one round trip through the model, one Bash invocation against the repo, and the answer that comes back. Drawn for the people in anthropics/claude-code#14375 (https://github.com/anthropics/claude-code/issues/14375) who use Claude Code to map out codebases and wanted real diagrams instead of ASCII art.

<div class="cinegram" data-cinegram="06-in-the-wild/05-claude-code-tool-call" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=rFnRjttGsv2VAi-MxIikmUluHEB5CBzvIjuLIBskXgSBFWRKZEnsVbOb29WUTBgG8hHZt33eD_OXLKq6SVEjzawN5MkjiSw265w6dar8ptgXy5tZQS6GvlgW18_mxs1jTfODsdXV9efz0mJX0bz0Fc2j93ZeorWLatsUs8K35D74po2xxMXy1Zui_eB7Y7EsIr2OxayoimXx5An8VGMELGOH1vZQY9uSYzjU5OCFBoIXviIInWNAYOO2lkDCgoSdAbkKopd_liv35Al4R9AG37Rxpn8H38kVwbQQ6-C7bQ2xJmh8RTZd8TVyDcbtfYnReAe4ReM4ajS5NFDrZ4ASpSZAxwcKEOXYpW-IYY3lbgF_CnhwsPFBr2rJt5bAOI2CTh7dmpKvJsn5v5v__-yLz_WCj-sYW15eXW1NrLv1ovTN1eWbrgxzR3yl9z6FQ-2hY8qp0liaruihwRZ8J4esaI1MrK9wQBepgkBooTK4DdgwyOsSVuA38PzHF7e3gCEuNNh8Ptd_X9YEsTZuCwcfYg1VwIN8qikQGE7pGPMKjvaSI9-VNTH0vgvQYFkbRynsbQRqTBRE7wTLXzumO1hbX-70lBx9y19qwBqDI2Z499vv-vnFt7dgnMZMCFFojEOrFxhhjgBTU7ljaOU3ZuMdzxKDJII8MOG5IarSd76LbZcgFzgBj0cLxJ2NdwvNgfW-FUIQRobORWMnbx2otYYYDibWGqq1aBwI36cpRudjTWHy5ue5Ln3nIgUGDAR3W-y2dAccMRIE1LtjjQ4chqCknQF74DJ067XAEj2g6zVa4xtyETgaayGStYoHHGpT1hC74ORIcXg1-UkeaVx6X6bSu0oDcUkOg_GSZIS7PQaDLt4twUR9c-wllYa1prpWzpAqYcAgoSKJH0rLQWX2FLbEen65nGsMVIEIk7yIABpMjOSAXmMZbQ_elcKijfWHssYQ4dsfVg7k5K9-9t0v8ndpzaupeLz8-61-j6159XwoK3j-_S1cJej0Vznpm-_vHfet_LJGrl-pTAhkevGGX338A7UevLwE757-snL5GDCfr7rr689IzpGPc_wO2_Pv5MnDCY7fylOHpx-_3fD5d_qclRsRWhWCwUQkjyK4KuANcEtULeFmcQ1v07E5UgvIO1gVPwsFeAcI_-yIkyKuRUgGLZQQcg9ARVwuYVWsVqvip0Qo4shQei1_QR8DU_hKLhAsnY9ZQHFtCTbBN8f6-YjhQGZbx1TsJiambbCM-QTaeSTjqiiS9gX8eU-hT9KkIrTx1voDA702cpToYUsx_ySRjMvELL3bU2Atn8WqSG8krEoYHlMLb8Dimqy86eHRtyxmUHVhCV9cXzcMb1NILd77caTw5PI92o6WcJMunoLB0thWxcssefKRz849diXFmsuaGuRzfF4qcgpnEmuSxmHvxWIPGwzQ2k7Sbg3HQRokOgPu0VjFTeCRcpjBD4TVDL4Jwp58lCBPkSsIy1q1EETp__rj377LJ0zSkjSzRCekEJqKZCUgTYQDJrbU6CqqpPPrrdJgT8HSQjrW1jTJDTHjlhg-uZefSzAxxXz_GqstKUi1cTvjtnKDau8S1h3351gNQp5Tnd4s8Tx1A8BpNcp7tcEzXULK8ACRhs5GouPkj1pTxi4QHIJ3W9tPM1l5SikLnRtT-WVSZ6HbxWbrsJGE5-MJhNqVw7aTrsGj7XESR7tydhEmcnq4thBRfSmG1BHZNyQClNqriYCQmugpcpLtB8psOOZSafZx2MLcgYhIqjSVEy3Cq6enaApQvaWlPLD1jukxePNDVsVwkfMxkWhVOA9sKgLabKiMnIzOu99-X63cseF7Z_tsZITe6e2mxJg0v0SNwc0cU5mtilZOZy9Q4mva-EDH4qDXVHZRWuZghxqM6rOyOpSBovzRNBI_u9kkmWg18a6CilyvD1QYDcNWvIVhsL5EO8vqm-oZ2bsTmU5uKcnzoCtTW4JO_U7oyuRNEq-GgBY5QuuNi4AxGxGxROonkfMMQBX0FB-ude2VE8q8P1GeTcu-NtvaStMZAmb-YBnNns5LXYprVTyXPFI1uhatnjSi5HQ_3CyTvUvgDFiK31SAOqYwG_yteG3RH6hMoDL60M-SmuiPqRkOtpixEfNE1iZ75ztbQY17gti3lNLvol-kees4uGjphso4DD1wrHwXZ1AKAlWq_dC5EmVkWPcnfnyKd_5J6TEiJhWXczqWHKa8TQTV7wYkFOHkgCYGaApxztYI5OdTIPX2ZItGpzS9-RFqnDJjBhVZ7C-F3_CDZ7v5dCjD0-ON0W5uHtGnE4apVGai5Erb-gEuHUwmc8lDzV5jSIFKLUkHza5HeXJqIc5GndwbWjTheKO0ro8YTHUchZOHyC2bxaLExH-RFJS-oyZmKHvv22VqF_HgocQQZFiSSbGH6HeUJhGdSGSGFxsy1NaoOad6cAL3pRaSXmgJF9B59mi7-EBrMYJxkUjPPtQOfvpHmJK8qFgV3x-n0NFEChiXqZMHeH9QLZZrRWjUDqrc5WlvmHOTkVFKOH_uL0SeEl1QTDyTK2kghHbPiXNPPXE43QL-gipbjkj4K5xopBFuA7Wp9-X9RRY7Nk1r--R-hITH6eL-CDqSqHM5we_pSuh1XGbNSCLyq3xYbH2aBcTrvKcdOeOXiPbkURm8T3JP126hViqge4BlX_wvfcmrBeMiiUeXvG_Q2C5tcHQ0g3XQQkwrkmEC0-6mEMl1UrBpq5XEXAQidG4Bd-r-lpN038kNxpW2Y7On3C1FKYZBNe8NxlXbGplO1nTqSkYPmtYXCJXZbCiI3JCr5D2ONMsEThsL3Xkojb3wtOpKCZYrYwZrKlFcO4obMmjltP_oOA4bGg1yugQ6GbInvJIA0tfgDeTFyPLxKXwGZ9maicZJZ14qLifFnOLDqvjOq2sb5ExzOpinwU087jlqsasnuTUMgTYdU5XmNqOdfd2nxCSbCD6M-RKuVlRaIyZNC0z3rWkeePfvf6mncJNdm3BraiGX-ue4UEnDnq4Mjdv73WA9soWa_BwIq0cNxoDDqI0Ugg_vbzFUtVKA0yrG2PERF82tafIyJg8OGlgnBwVH5yr12-rrV27jwzjOvPvtP0_Px4UglZzhOlJSchDUgYkBsGanwwCkJeJjLmCwauNkyMaSi7aHKvh27OoLuI15v3DfCDD2SZ5pApRPaTcxigNO_n1H1F5YTxzI2vnGhyZTCx10LpUfVdNukSTckhrWsQtx7ModNKaaS4M87f0nMN4T6guF-Yd0_RNPcZKH9-n8Qpi1bk0G2nyAF5jShF6nxfJD24ZhWSfHPWfGN2ZPbtS82STdyhPdn5sIJTpd7TL2xy_z_x4kxU-bgLpPhhB5lzbL0WfamyiiIdKLHGmyZp-s0pNEj7ZE3ekBe10a2IM8Wg8l3Elu4-hlzizHQ209fRx2rO_b5m_VeOj-MFCbu34isb6dVPJXf1C3H9Ibg6Hsrw91P6xc2gfp9fhT7_Pt7coVb395-98AAAD__w){ .md-button }

??? abstract "The source — `06-in-the-wild/05-claude-code-tool-call.dgm`"

    ```dgm
    %% What actually happens when Claude Code runs a single tool call, end to end:
    %% one prompt, one round trip through the model, one Bash invocation against
    %% the repo, and the answer that comes back. Drawn for the people in
    %% anthropics/claude-code#14375
    %% (https://github.com/anthropics/claude-code/issues/14375) who use Claude
    %% Code to map out codebases and wanted real diagrams instead of ASCII art.
    %% ---
    %% The thing worth drawing here is that the model never touches your machine.
    %% It emits a `tool_use` block and stops; the harness — the CLI in your
    %% terminal — is what checks permissions, runs the tool, and feeds the output
    %% back as a `tool_result`. The loop repeats until the model replies with
    %% plain text instead of another `tool_use`.
    %% ---
    %% The counters are `gauge` state rather than narration, so scrubbing to any
    %% moment still tells you which turn of the loop you are in. The second
    %% scenario is a `variant`: it replays this one up to the permission check and
    %% then diverges, so the shared opening is written exactly once.
    flowchart LR
      you[You]
      cli[Claude Code TUI]
      api[Anthropic API / model]
      perm{Permission check}
      bash[Bash tool]
      fs[(Repo on disk)]

      you --> cli
      cli --> api
      cli --> perm
      perm --> bash
      bash --> fs
      bash --> cli

    scenario "one tool call, round trip" { speed: 1.0 }

      step ask "You ask a question about the repo" {
        desc: "\"Which tests cover the parser?\" is not answerable from the model's weights — it is a fact about files on your disk. Everything that follows exists to get that fact into the conversation."
        flow you -> cli { label: "which tests cover the parser?", dur: 700ms }
        gauge cli { label: "turn", value: 1 }
      }

      step send "The CLI sends the conversation and the tool schemas" {
        desc: "The request is the whole conversation so far plus a list of the tools available — Bash, Read, Grep and the rest — each with its JSON schema. The model cannot call anything it was not handed a schema for."
        flow cli -> api { label: "messages + tool schemas", dur: 700ms }
        set api { badge: "thinking", state: busy }
      }

      step tool_use "The model answers with a tool call, not prose" {
        desc: "This is the step people usually picture wrongly. The model does not run anything; it returns a `tool_use` block naming a tool and its arguments, and then it stops and waits. The turn is over until someone feeds it a result."
        flow api -> cli { label: "tool_use: Bash(rg -n \"parser\" tests/)", dur: 700ms, style: response }
        set api { badge: "tool_use" }
        note api "no side effects here —\nthe model only emits JSON"
      }

      step permission "The harness stops and checks the rule" {
        desc: "Before anything executes, the CLI matches the concrete command against your allow and deny rules. This gate is local, it is the reason the model's output is a request rather than an instruction, and it is the last point at which nothing has happened yet."
        flow cli -> perm { label: "Bash(rg -n \"parser\" tests/)", dur: 600ms }
        highlight perm { style: active }
      }

      step run "Allowed, so the tool runs against the repo" {
        desc: "The command executes as your user, in your working directory, with your files — the same shell you would have typed it into. What comes back is ordinary stdout, capped and truncated by the harness rather than by the model."
        set perm { badge: "allowed", state: ok }
        flow perm -> bash { label: "execute", dur: 500ms }
        flow bash -> fs { label: "rg -n \"parser\" tests/", dur: 600ms, delay: 500ms }
        flow fs -> bash { label: "12 matches", dur: 500ms, delay: 1100ms, style: response }
      }

      step result "The output goes back as a tool_result" {
        desc: "The result is appended to the same conversation as a `tool_result` block paired to the call's id, and the whole thing is sent again. That resend is the loop: turn two carries every token of turn one plus the tool's output."
        flow bash -> cli { label: "tool_result: 12 matches", dur: 600ms, style: response }
        flow cli -> api { label: "messages + tool_result", dur: 600ms, delay: 600ms }
        gauge cli { label: "turn", value: 2 }
        set api { badge: "thinking", state: busy }
      }

      step answer "Plain text ends the loop" {
        desc: "The model now has the file list, so it replies with prose and no `tool_use` block — and that absence is the only thing that stops the loop. Had it needed one more grep, the diagram would simply run again from the permission check."
        unset api
        flow api -> cli { label: "text: tests/parser_test.go covers it", dur: 700ms, style: response }
        flow cli -> you { label: "answer + the commands it ran", dur: 600ms, delay: 700ms, style: response }
      }

    %% The interesting failure is not a broken tool, it is a tool that is never
    %% allowed to run. `until: permission` is inclusive, so this scenario replays
    %% the base through the gate and then tells a different ending — and the model
    %% still has to produce an answer, because a denial is just another
    %% `tool_result`.
    scenario "permission denied" { variant: "one tool call, round trip", until: permission, outcome: fail }

      step denied "No rule matches, so nothing executes" {
        desc: "The command hits the gate and is refused — either by a deny rule or because you declined the prompt. The ✕ is on the tool, not on the model: the Bash tool was never invoked and the repo was never read."
        set perm { badge: "denied", state: error }
        flow perm -> bash { label: "blocked", dur: 700ms, status: fail }
        dim fs
        note perm "no matching allow rule\nfor Bash(rg …)"
      }

      step relay "The denial is reported back like any other result" {
        desc: "The harness does not silently drop the call. It sends a `tool_result` saying the tool was not permitted, which keeps the conversation well-formed — an unanswered `tool_use` would leave the model stuck mid-turn."
        flow perm -> cli { label: "permission denied", dur: 600ms, style: response }
        flow cli -> api { label: "tool_result: not permitted", dur: 600ms, delay: 600ms }
        dim bash, fs
        gauge cli { label: "turn", value: 2 }
      }

      step explain "The model answers without the tool" {
        desc: "Given a denial, the model does what it can: it says what it wanted to run and why, and asks you to allow it or to paste the output. The loop still ends the same way it always does — with plain text and no `tool_use`."
        unset api
        unset perm
        flow api -> cli { label: "text: I need to grep tests/ — allow Bash?", dur: 700ms, style: response }
        flow cli -> you { label: "what it tried, and why it stopped", dur: 600ms, delay: 700ms, style: response }
        dim bash, fs
      }
    ```

← [auth refresh edge cases](../06-in-the-wild/04-auth-refresh-edge-cases.md)  
→ [pytest tests/ — one session, hook by hook](../06-in-the-wild/06-pytest-session-hooks.md)
