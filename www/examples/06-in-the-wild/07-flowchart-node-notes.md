<!-- Generated from 06-in-the-wild/07-flowchart-node-notes.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# escaping the self-loop note hack

Mermaid has no way to attach a note to a flowchart node — only sequence diagrams get `note right of X`. mermaid-js/mermaid#2712 (https://github.com/mermaid-js/mermaid/issues/2712) and its much older, much bigger sibling #821 (https://github.com/mermaid-js/mermaid/issues/821, open since 2019, 150 thumbs-up) both ask for it. The workaround people reach for — drawn verbatim in #2712 — is a fake self-loop edge that carries the note text, hidden afterwards with `linkStyle`. Its own author lists the cost: the fake edges are "hard to compute the number of ... when there are many edges", they "seriously affect the layout", and they are "very not convenient to distinguish" from the real ones.

<div class="cinegram" data-cinegram="06-in-the-wild/07-flowchart-node-notes" data-height="990"></div>

[Edit in the playground](../../playground/#doc=lFjdbtw40n2VggZBvgHU7Z_BJDN9Zzuf44HhxQKbiwVWC3S1WN1izCY1ZMnt3mSAfYh9wn2SRRUltdpO4slN0pLIInnOqVNFfyoeisVZWZDnuC8WxembmfUzbmi2s86cnL6drV3Y1Q1GnvlgaOYDU5qbzbYoi9CS_945a-soFYt_fCra753KxaJgeuSiLEyxKF69gjuKW7QGGkzgA-xwDxwAmbFuAEEm6wsYo4JEhf_--z8QvNtDot878jVV_tUrMBY3EbcJNsSw1MnRbhqGsIa_L-ewzavNPqaT_ucP52_PznXu_zXMbVqcnGwsN91qXoftyfPxJzaljtKJTPsR0BuwnGDb1Q0EZyiWGkufV3azoQjJrpz1G_jhl_Oz71zkl_OzHE9ogmR9TXB-evZrCWc_nwI33XaVZl37I6wCN4DpHtYhguU5fGgIdiHeYwydN9BSaF2GKJIgK-MEQhNx5-GB4grZbsF6UED0m00CO94TJHLrmQuhBTIbAm6QocYYLSWNyQ31TNEjl9BYY8gDrpniDqNJsLPcwNJZf_833jtazuE3ThB2HrDjJkRwNnGSOBqvDokXGlWXl0UTYCSoigajEUHUYdt2sqIs3W1XFIXj-XwOu4a8vI4kUzId6Pc5SlWU8m0PVZEo2tAltwdcr6lmDeVwHzqWUUKtjszrPlDcyxn7_fkH8pY8y1aMTWz9prOpqQpYx7DVUJHQQfCU5jpnNpvp_x8amwRabshGhaAXbZmFLczoD9mAxJgp5vJ6fCoV0BF63oUnPKVh-1PQwVnfk9dY_boFQ46YjPCt4XzgRsRKLknedXVDJqtJFhF6ZechHa2VpWCEE4IV1veAWRc5Ay-Wupv8cL0E6xMTmkVGaDCAGp0LHSeBgaFt0HPYZtLmPXA0QAUNujWsyIWdQNk6tL6ERsDmMSBu0HoVcouJCSyD9Rw0Vj_mddagsw8iMssh6k5JyVbY96GDRKS54Cfq3xL6fMYUtqSYPaVZ6F1ew2xWdaenPxHcgPw4fwPvhx-3PTCHFzCbD8PvlpCaELlRj0lwTy0DPWLNoti89i5aZvLz8cyR0CQRHSRxICUcEyQ5EbqMZpZ2CmACCZk2LUYlicmmHliO-EBOj9Bv6bDfySatN_bBmg6d2wtyknuh22R1erKSidCijbDDpNAO24bgAfMew1ptNOz8MxTBYWJITC2gSwHajsWXVE1LCXFXap0wPTNel2jQAE58y_rxjCHajfXoABnQuewzTHXj7e-dMO1fMyC0yOLhwUl6BM9hTDVZa_A9ZNgEhgbrezKQ3bYUqYn5Jj2h3-uMeeU3EdsGPlxWHuDiIIzPVfGOUh1tyzb488UFNAJ_gMuq-AyXh4FXMnHy_E6e3000cy0vrr4e-WqIfC2RdfS3BSojbg4j7uT5y2qtfOVTTR6jDVAVlGpsxUgEtAMNagECVlXAJ0gtkVnA2fwU_pD50NOsLQNUxcVQZAZx6PxsEHhkERJO5gMYSvUCquI3rQFSEylKmg-ki95ViUthYPa5qo4wOlv8NYaWIsuqYQ0X8v3zxVJ9ZCqovsat9qLFpx677qLK3oig4UZKkbjPsW4vDkaoIOF2qJ9aLX0A6x-sdA-5AmoJQE1xiiXoQy5XQH5jPZXCLEfbQnigOK-KjEhjN43TJugiv8imDC-cWwmyhhbgaM1CEByxJL4U1muoiqE6iJ2X4HBFzpHpq7YJfaX7GFbQ-bpBvyHznK8LFbz8cyVADUY35bA3uX6JBNbXrjOioCckHqVQVRUScIWSoHbI46GQWBZKpcKhErRr9qMR9j44aNCHvPDErFQVkVpCHrU-0CgUy4tteJBvrTiE2ELnDHzsxNJwL6rgna1pJEvXvRj99XL8dQWfwHRxAb-enm5TpmPK7WWZ7WFKkbYpcUMzYekq-_7la-GExdQMcegicBPFr-FdmdtIJ8OCF494StJ1hkj0LEfrC64W47DWV9niIn0M1qce3YUkDkfUfWKMYZebpCuFdi_oDHgOdUyBzhvGfrsldH7U1tOtz-EvfaWR2qcVwBMZseRDCyl6oEdtFmCnfLM6gQlavacMXI24Xw-4v53iroMO5Lw7Gq67XBzn10-Ly9F7S3iwCO-kyfwWo9dP6VwPvng9dlwHxUVb32fANpQ76fxhbR-f8_hhklWJ6iD9wOBsZW5aXnTGa3XG62UpmojENkrOT9JWrnIXr9Vs5_BhqJtlX0JZfv6LYjhu9TSjuE9GiUJeIk_qsxo4PeK2FV98zPeGIL3SbuQwd5rwwv4PDpfviM8sbo1e3FUA_1aRhBVJ5yv7jTTcVqRxrNED1pxbozp0nr9EhU25aPQHHiuWylmuOSJYhi0aAmdZGrkFLPPiS6iDc9gm1YKsbkguIygnTtnucNJHDsgKd9-8SaltrvZAe5rD_z-26I2k4KEXvJk-vFfhTRpFvTTlDYVorMe-n878KhS4clQed49Demcd6z0Vvd1mcw1-dAmOWN8fJ-xkY1_JwJ8X10MG3oyp9-ZZUk8P9eVAbw6B3v-pQLdfCfT2EOj2y4EOZnBTwvsy92NThYrNQlXIV8XsVjpaiDQ4v_bHz0V3p76XO-UWIw_uPXZJkz5HrnP61wAe_kbT2MQh7mVSYvRGDdz37cnotq8TMMYNsc5edWq36k1jE5Tv6H6_w3055PCzbly7PWe3VkrW0y78kDG5Xx978GN13BwuLF_h4pfFzcDF3TdIff9ioF8X7_9MoNsXA52dLm5fiKQ2dwdVoX_TITP87UHSznpDLXlDnmEV0dcNpYnj5dbmmczueoEVf_zzj_8FAAD__w){ .md-button }

??? abstract "The source — `06-in-the-wild/07-flowchart-node-notes.dgm`"

    ```dgm
    %% Mermaid has no way to attach a note to a flowchart node — only sequence
    %% diagrams get `note right of X`. mermaid-js/mermaid#2712
    %% (https://github.com/mermaid-js/mermaid/issues/2712) and its much older,
    %% much bigger sibling #821 (https://github.com/mermaid-js/mermaid/issues/821,
    %% open since 2019, 150 thumbs-up) both ask for it. The workaround people
    %% reach for — drawn verbatim in #2712 — is a fake self-loop edge that carries
    %% the note text, hidden afterwards with `linkStyle`. Its own author lists the
    %% cost: the fake edges are "hard to compute the number of ... when there are
    %% many edges", they "seriously affect the layout", and they are "very not
    %% convenient to distinguish" from the real ones.
    %% ---
    %% This is their own diagram, node for node and real-edge for real-edge, with
    %% the two fake self-loops and the `linkStyle` line that hid them deleted —
    %% nothing else touched. The two texts those self-loops carried come back as
    %% `note A` and `note F` instead: real Mermaid callouts, not phantom edges.
    %% The diagram half below is plain, honest Mermaid again — paste it into
    %% Mermaid's own live editor and every edge you see is an edge that means
    %% something.
    %% ---
    %% Their `F --> H & G & K` and `G & K -.-> M` shorthand is kept exactly as
    %% written. Mermaid reads one such line as several edges, and so does this:
    %% the flows below travel `F -> K` and `G -> M` individually even though
    %% neither pair was ever written on a line of its own.
    %% ---
    %% The last step also puts a `note` on M, a node that never had a self-loop in
    %% the original at all: the technique isn't a patch bolted onto the two nodes
    %% that got hacked around, it works on any node.
    graph TB
      A --> |"Description2:A how to B"| B --> C
      B --> D
      D -.-> F
      C --> |"Description2:C how to F"| F
      F --> H & G & K
      H --> M
      G & K -.-> M

    scenario "escaping the self-loop note hack" { speed: 1.0 }

      step a-note "A carries its own note, not a phantom edge" {
        desc: "In the asker's original this was `A ---|\"Description1:Properties of A\"|A` — a self-loop hidden by a `linkStyle` line further down. Here it is a `note` on A instead: the same text, with no invisible edge for a reader, or a layout engine, to trip over."
        highlight A
        note A "Description1:Properties of A" { side: left }
      }

      step handoff "The two real, labelled edges do their job unchanged" {
        desc: "A to B to C is exactly the asker's Mermaid, labels included: \"Description2:A how to B\" is baked into the diagram itself. That is why the flow below carries no label of its own — repeating the same text on the moving packet would just say it twice."
        flow A -> B -> C { dur: 900ms }
        highlight B, C
      }

      step converge-f "C, and B's dotted detour through D, both land on F" {
        desc: "F is where the honest half of the graph rejoins itself: a straight arrow from C carrying its own Mermaid label, and a dotted, unlabelled detour through D. Neither one ever needed a fake edge to explain what it was doing."
        flow C -> F { dur: 700ms }
        flow B -> D -> F { label: "Description3:B how to F, via D", dur: 900ms }
        highlight F
      }

      step f-note "F carried the same trick, and gets the same fix" {
        desc: "The asker's second self-loop, `F ---|\"Description1:Properties of F\"|F`, is retired exactly the way A's was. Two nodes, two notes, zero phantom edges — that is the entire technique this example exists to show."
        note F "Description1:Properties of F" { side: right }
      }

      step fanout "F --> H & G & K becomes three edges you can actually count" {
        desc: "This line is the asker's other complaint made literal: `&` collapsed three destinations into a shorthand that is \"hard to compute the number of\" by eye. Expanded, F -> H, F -> G and F -> K are three ordinary edges — countable, individually labelled, and each animating on its own track."
        flow F -> H { label: "Description5:F how to H", dur: 600ms }
        flow F -> G { label: "Description6:F how to G", dur: 600ms }
        flow F -> K { label: "Description7:F how to K", dur: 600ms }
        highlight H, G, K
      }

      step join "H, G and K all reconverge on M" {
        desc: "M was never part of the original self-loop hack — it has no history of standing in for a fake edge's target — but it gets a `note` here anyway, to show the technique is not limited to the two nodes the asker patched around."
        flow H -> M { label: "Description8:H how to M", dur: 600ms }
        flow G -> M { label: "Description9:G how to M", dur: 600ms }
        flow K -> M { label: "Description10:K how to M", dur: 600ms }
        note M "reached from three independent branches" { side: below }
        highlight M
      }
    ```

← [conference signup flow](../06-in-the-wild/06-calm-flow-over-architecture.md)  
→ [two questions, answered in order](../06-in-the-wild/08-overlapping-activations.md)
