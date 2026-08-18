<!-- Generated from 05-ai-systems/02-multi-agent-fanout.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# fan out, barrier, synthesise

An orchestrator splits one research brief across three worker agents, waits for the slowest of them, and writes the answer itself. Fan-out is the easy half; the barrier — sitting still while two workers are already done — is the half that decides how long the whole thing takes.

<div class="cinegram" data-cinegram="05-ai-systems/02-multi-agent-fanout" data-height="990"></div>

[Edit in the playground](../../playground/#doc=rFfdbty6EX6VgYogQLC79dqnabu98gGa2wBJgHPhDZBZcVbiMUUqHMrKIgjQh-gT9kmKGVKybK_dE6BX9lLkkJzvZ4bfq7tqt11V5FM8Vbvq4i9rtGs-caKO_3xxue4Gl-waG_JpfUQfhrQxTVetqtCT_6kFR-uIq93N96r_qXWp2lWJvqVqVZlqV716BdceQqxb4hQxhQjcO5sYgieIxISxbuEQLR0B6xiYIbWRCMYQbymCbsErGNEm3vtXr-AYIqSWgF0YiROEo_zsVoDewBhtItbv6HmkCDYxueMG3qFfhyGBzV8J-aThWnTHf-jQAWO0FOE___o3sE3J-gY4WedgbK0jSGMop2LASIAuEpoTGLmKLLL5gBJLokJqMYGh2hpiaMMILvhGP49tkICtbJHwlnijK9frtf791FoG-oZd70gO3FIkvbfslGwny-LgaAdv3qBzgHWywTNYDwicqAdOGFM-TWgotRRzfuQjQxy8hsJjkhT7IBM2b97Ap5bg-CBRkY4hEvhQoik2GmUFtkySoS9HF8Yvi5OwNSSnsb5xecUGfos2JfKaGA034knzIYmkCEzEMErWFGClm4Qc0DnJM7HmOe8Y6etAnBgc4Z31jQbEBMHXBQ7PidAIQSQraGONTBq6DobA0x1F6CkeQ-x4Ax8I61bT_IXp6xeNF7w7wdhSyZfeDhryg_XkTlDjwIVunr6lxyjec0pBKdRj7Ejh02vDAevbEaPhDXxaMizTlzC6TFSBT2hnbNeRyZTVsLdUMPXCiwPpofRzam000CILfhCpd5bMao4mU3pbpyFSTlIXOLnT2ho3Z7-cuQ1eteZps_cCdd1iTPDp172HrN6bD5OYf5Wfn-WD6P7m_UL8n_dexnk4NBH7drrszW9Z7Ncqdl0KMG5vPoYh1sR5fBq-vHlnnfXNo-Grmw-Wb5eD5E3eLlIfYrr5ePKpJbZMBq41u-U42X3W6_1wcXFFeurp9Pej4_bM2OWZsaunY_kAshnX5DHaAPvqiB7CkFYTR1bA8wH3FXwH7onMDrabC_hR8iZw59Puq_ee8v-rIgiVw1J9NkmcnB9DXO9gXwkpl3b8muFoIyf4PRwEbEN16PrAViLtIA1RWYWT3MD6FATAdUJJd_a4oATzRAZURGoorycS8wau_am4XZ5PnGcX75mnQo1eQh3UhzyZnCJh64HEPovJORTnimHwZrOv8hWFlhOWCyjhOzg8kJPbIzNxKQ_118Hma-6rFZgh7uCvFxcdS7IlHFOa1h_QNCTre4eqMlnBCRPt4DDwKS9ZgpRrIuyra-ceFDTO1gyTx4kXiFGhPwuWLJR76bXFgiT6aiFyVA9WzzSWe0x1C5Z3xVUXDklzJdBs-slpcmWF4Oc6AB-lQqeMlhrwoiznAjKGwRkwEccn1lp8I5tr7Wx34IcIZWnMqlrCc7TeQB9th_EEnMU_g_N2Cc6ZSJfLSOqschyHnOAYhgjH7Bp_ON7VMp6znAEzhA6iOM35QMIavdTMmeyKz5GmrLhcrpDD_4_5V4_mc_hDtJzK0b6SUnO03nKbS8xcEB70arnrOsNLyw9KWom70ZpXgNOImqnSxOXqpXXUm1LGVrq8QJPngcaWtqvkYTWfjRz24t7JdlT6vuwBssaHEcgnG6UwqzBkq0zy1wx9DAdH3UMujttnrWK7fcJAtQfJ8Elar0jcB8_0gELj1bMB3z6izcvRSv_wyKoVcqhb9A3x7AKiX7HvFnknTRmn0Pdk5kCTa2m-utBJkh_5w7GYbG7pJJW580ErPrB53hHLjJe4-kANgv9icrhdQR1ciPLtT9u3ePUL7qvnqf5Ty43tStku_1_l_1vbtM42bdFdQeC8YiT3TeOyZhavjmKekaRCnpHI9dxH10FSrKnHb7YbOmGuQHaI6AXdVTFLrUp3FLGhDfzzjuIJDoNpJAelb8YGpVYsZ6r7x-ldkUt5lsdMfmhlN8yvDxPqQQjwSAeXz9L2l1mdPHQdRmmfZg7__UUOP3W2_wN8D8CZG6YznY3aBwPOBbg0I3MdfArau1APDF24k54-PLXDZW8tuc2Km35N70Jtn4tl2Qg9xmTRzduLuNpcj6XNiVQHX1tHJrdW-sS7b5KCEMiniMbWaX7MqgFPzwnJxH05r7U06wZG2_UJZ73b1NueF_Ocz-cV7UMqHNlXV4_vtt_77eK48lZKQWgR3B3lgyzhM-TsnSrrvZ_f69qryIPfpjZXXMvYRCK1LdnenO9qJ5pO800JMYapVMyF6SC61HegC3JXMJjovsyUkzCeGDhARO2aUotenku3uR8erfdS8e6Frqi1-tgXwS3PrWD9PnCSdkkcJE4PYy15M0iDn2B5ri3JT4mlQMOcuxVsocY7wjQL9G8vCPTH3lc_Pv_4bwAAAP__){ .md-button }

??? abstract "The source — `05-ai-systems/02-multi-agent-fanout.dgm`"

    ```dgm
    %% An orchestrator splits one research brief across three worker agents, waits
    %% for the slowest of them, and writes the answer itself. Fan-out is the easy
    %% half; the barrier — sitting still while two workers are already done — is
    %% the half that decides how long the whole thing takes.
    %% ---
    %% This example is here for one timing rule: **all actions in a step start
    %% together, and steps run one after another.** The fan-out is therefore not
    %% three steps, it is three `flow` actions inside a single step. Written that
    %% way the reader sees what the system actually does — three requests leaving
    %% at once — instead of a staircase the code never performs. Reach for `seq`
    %% only when one action genuinely causes the next.
    %% ---
    %% The barrier step is the same rule read backwards. Two workers answer early
    %% and are dimmed; the step keeps running because the third has not replied,
    %% and the picture of a mostly-idle system is the honest one.
    flowchart TB
      brief[Research Brief]
      orch[Orchestrator]

      subgraph workers[Worker Agents]
        w1[Sources Agent]
        w2[Filings Agent]
        w3[Risks Agent]
      end

      report[Synthesised Answer]

      brief --> orch
      orch --> w1
      orch --> w2
      orch --> w3
      orch --> report

    scenario "fan out, barrier, synthesise" { speed: 1.0 }

      step brief "One brief, three questions inside it" {
        desc: "The orchestrator's first job is decomposition: turning a request into sub-tasks that do not need each other's answers. Anything that does need another's answer cannot be fanned out, and belongs in a later round."
        flow brief -> orch { label: "assess the acquisition", dur: 700ms }
        set orch { badge: "planning", state: busy }
      }

      step fanout "All three workers start at the same instant" {
        desc: "Three flows in one step, because that is what dispatch is: the requests leave together and no worker waits on another. Splitting them across three steps would draw a staircase the system never climbs."
        flow orch -> w1 { label: "find primary sources", dur: 600ms }
        flow orch -> w2 { label: "read the last four filings", dur: 600ms }
        flow orch -> w3 { label: "list the deal risks", dur: 600ms }
        set w1 { badge: "searching", state: busy }
        set w2 { badge: "reading", state: busy }
        set w3 { badge: "reasoning", state: busy }
      }

      step barrier "Two finish early and the orchestrator waits" {
        desc: "This step is the barrier. The sources and risks agents are done and dimmed, the filings agent is still reading, and the elapsed time of the round is now entirely that one worker's problem."
        flow w1 -> orch { label: "11 sources", dur: 700ms, style: response }
        flow w3 -> orch { label: "6 risks", dur: 700ms, style: response }
        %% The orchestrator's badge changes because its job has: it stopped
        %% planning the moment the requests left, and it is now only waiting.
        set orch { badge: "waiting", state: busy }
        set w1 { badge: "done", state: ok, color: "#16a34a" }
        set w3 { badge: "done", state: ok, color: "#16a34a" }
        dim w1
        dim w3
        highlight w2 { style: busy }
      }

      step straggler "The slowest worker returns" {
        desc: "A fan-out costs the maximum of its branches, never the average. Every budget written against the average is wrong the first time one worker hits a long document."
        flow w2 -> orch { label: "4 filings summarised", dur: 900ms, style: response }
        set w2 { badge: "done", state: ok, color: "#16a34a" }
        dim w1
        dim w3
      }

      step synthesise "The orchestrator reads all three answers together" {
        desc: "Focus moves to the orchestrator because the work has: the workers are idle and their partial answers now have to be reconciled into one that does not contradict itself. This is the step no worker could have done."
        focus orch
        set orch { badge: "synthesising", state: busy }
        note orch "3 partial answers\n1 contradiction to resolve"
      }

      step deliver "One answer leaves, with the disagreement noted" {
        desc: "The filings disagreed with two of the sources about the closing date, and the answer says so rather than picking a winner. A fan-out that hides its disagreements is just a slower single agent."
        unset orch
        flow orch -> report { label: "one answer, 1 caveat", dur: 800ms, style: response }
      }
    ```

← [one question, four turns](../05-ai-systems/01-agent-tool-loop.md)  
→ [rag pipeline](../05-ai-systems/03-rag-pipeline.md)
