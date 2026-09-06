<!-- Generated from 06-in-the-wild/08-overlapping-activations.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# two questions, answered in order

Alice fires two questions at John before he answers either, so both of his activation bars are open at once. The animation plays the four messages in the order they happened — first question, second question, first answer, second answer — and a gauge on John counts the open questions live, so the pairing the static diagram cannot show is simply watched happening. See mermaid-js/mermaid #1765: https://github.com/mermaid-js/mermaid/issues/1765

<div class="cinegram" data-cinegram="06-in-the-wild/08-overlapping-activations" data-height="810"></div>

[Edit in the playground](../../playground/#doc=rFbbjuREDP0VK2i0u6Kvg9hF2SckQAIJhLQr8UCQppK4k2IqrlCudG8YjcRH8IV8CbIr6cvMCJYRT91JXC7b5xzbd9k-y7eLDCmGMcuzzeulpWVscXmwrl5vvlj6PQZn-t5SszRVtHsTrSde1U2XLTLfIz3j2M465Cz_-S7rn3E6ZnkW8UPMFlmd5dnVFXzpbIWwswEZ4sHDbwOyngAT4TvfEpS48wGhRTDEBwwMaGOLYQHsofSxBb-D1nJBV1dwuhJKExhMQJBMxZunClfwXh3ZLhn1zowMsUXY-SFAh8ymQQZL6k4--FBjkH8jtKbvkbCGv_74U2LmeIx3AYyVp_rsRTJIQS_U3WSSXqkTI4_QmKFB8JQSrvxAMQWloZ9K4uweNe3YojrsjQ2WGrXlaKKtoLamCaaDyhD5CNz6A1gGtl3vRjiYWLVYT5lYalbq5x0idBg6Y-vlr7ye_sIn2zevP8-hjbHnfL1ubGyHclX5bv3YeG2ZB-S1HFGfy-VSf9-3lqH2yCDx7OwH-D6deMFzzAGpRk1EaiI0OKNRApKjdU79VSaEEcgfk7e082HC05PWojK0N7zQ8iq4aOIQEAJqMaHE1lKt7jRHeOlMic5hfcYgfqWFk4thoA7jCn5qTYSqNaQcSRh1WNuhyxP9INoOnSWE1kjC0KDvMIYRoocSwXSlbQY_CMNW8LWpWuCIPZDpkAE_mCq6ETwleCc6KuRFdmht1c7kKdF5aljcpvczS4pMAktWWEM5Ya7-lMoLxaEcpa7ArelxdYHXTcdNfiNOnDf1skSjVW4x4GqS67IYNpvPcKKrCcFO6j3q56VWXQSQ6kL1Q3280pfqYXaXnF_4E-WffMmRx0J69TbFDNsbWE9_r2-ECij17S1NCnf-IPWS_5aicC55MyGISIgjmlq6icMYZ1lVvuutwwDNgMypf1zqNaCp-ShJVa84GegIwknBwu_tQowJrqff0lS3Etf8fqNW7FOmVRjKUoPxYGjUMA0lYZ8YIyrv5PNZ_wySJuycbdq4KoiF-1ThV6lBFAQAl3hOMHyqaeXwzUWD-3f7d5cApwMJ4MsDy-RmviGV6SPM352jXlBBXCGZYEUcF7NjceK_pcT6IoM74B6xzmG72sB9oQGq-AzfbqHIUm6Gbye6XKafwV0KsUaucigyDXXqFqhCbA3VDsHGFfzgYyugoWMUKfVItTyPGKcGbnlWCxjnSa0GOvUH3CNJ9I-aZW8r6WWrIkvxKK2f0OUd1EPI4c1m0_ECkkAkbT3jq4HVLD0nQk_HtBVKhpfTp8gWsDduQPm0LbLk60Edry_raB6qfp7mpwKfNauna_yCH0x1KcTI01jX-TldIlXkaEJkLWnDtk5o6Aia27XvkOJTE9N5xmQTW9F0ri3oco2Y5gubDsHZnbb6NGYksGOzT4jOI9iMU48WTONg3NzEq3Rl-ujpWZhez5j-Zwyvn8RQI9se-T2tXB8jiG9JoNBxaTvhtZI8oOwdlR9cLdvWwLaUZ8l8WuRSZUsT3so1QZVAHvaWpVSdCbeyf4066qPoTGp7JJRsCmlXksMr-NGZ2XTWPgTs_H7CVtv4wYfbPMV32o3iccWcp8TxEsO3WKcCXGL0xPz6J939TzrTIK-fwujREqrUlCX5bK9J1X8MoIw2ZzgeAZkJOq9RKfyAcQikHe93DD6NxHkZO8jag4IJUuXr1IHl7KSyF3xSiYw5G6ESMSFFG9CNsAu-m1JJ80ocJIXEFnla1KUODNI15pX2Obg8XzubGReAgRjjsZfeF5Td_3L_dwAAAP__){ .md-button }

??? abstract "The source — `06-in-the-wild/08-overlapping-activations.dgm`"

    ```dgm
    %% Alice fires two questions at John before he answers either, so both of his
    %% activation bars are open at once. The animation plays the four messages in
    %% the order they happened — first question, second question, first answer,
    %% second answer — and a gauge on John counts the open questions live, so the
    %% pairing the static diagram cannot show is simply watched happening.
    %% See mermaid-js/mermaid #1765: https://github.com/mermaid-js/mermaid/issues/1765
    %% ---
    %% This does not fix Mermaid's static rendering — two overlapping bars still
    %% carry no pairing information on the canvas, and the feature request behind
    %% #1765 (labelled activations) is still unmet. What changes is the medium:
    %% a timeline has no geometry to be ambiguous in. Each step names exactly one
    %% message, so "which answer belongs to which question" is answered by watch
    %% order, not by bar shape.
    %% ---
    %% `msg:` is load-bearing here. Alice -> John carries two messages (the first
    %% and second question) and John -> Alice carries two more (the first and
    %% second answer); `msg: 1` / `msg: 2` on each pins the flow to the intended
    %% arrow instead of letting the compiler guess. The gauge on John reads the
    %% count of unanswered questions — 1, then 2, then back to 1, then 0 — so
    %% scrubbing to any instant shows exactly how many questions are in flight.
    sequenceDiagram
        Alice ->> + John: First question
        Alice ->> + John: Second question
        John -->> - Alice: First answer
        John -->> - Alice: Second answer

    scenario "two questions, answered in order" { speed: 1.0 }

      step ask1 "Alice asks the first question" {
        desc: "John activates to handle it. Nothing else is pending yet, so this message alone is unambiguous even in Mermaid's static picture."
        flow Alice -> John { dur: 700ms, msg: 1 }
        focus John
        gauge John { label: "open questions", value: "1" }
      }

      step ask2 "Alice asks a second question before the first is answered" {
        desc: "John's activation bar stays open and a second one starts alongside it. This is the moment the static diagram loses the thread: two bars are open on the same lifeline, and bar geometry alone cannot say which eventual answer closes which one."
        flow Alice -> John { dur: 700ms, msg: 2 }
        gauge John { label: "open questions", value: "2" }
      }

      step answer1 "John answers the first question" {
        desc: "In a still image this reply could plausibly close either open bar; there is no visual marker tying it to one question over the other. Playing it in order removes the guesswork: this is simply the answer to the question asked first."
        flow John -> Alice { dur: 700ms, msg: 1 }
        gauge John { label: "open questions", value: "1" }
      }

      step answer2 "John answers the second question, and both activations close" {
        desc: "The last open bar closes and the gauge returns to zero. The pairing was never encoded in the diagram's geometry — it came entirely from the sequence in which these four steps were watched."
        flow John -> Alice { dur: 700ms, msg: 2 }
        gauge John { label: "open questions", value: "0" }
        unset John
      }
    ```

← [escaping the self-loop note hack](../06-in-the-wild/07-flowchart-node-notes.md)  
→ [order placed: payment approved](../06-in-the-wild/09-animate-the-flow.md)
