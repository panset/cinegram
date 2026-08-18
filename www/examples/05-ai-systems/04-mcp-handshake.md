<!-- Generated from 05-ai-systems/04-mcp-handshake.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# launch, negotiate, call

The handshake `cinegram mcp` performs. An agent host launches the server on stdin and stdout, agrees a protocol version with it, asks what it can do, and only then calls a tool — here `sheet`, which comes back as a picture of a whole scenario.

<div class="cinegram" data-cinegram="05-ai-systems/04-mcp-handshake" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=jFfdjuNMEX2VktGKC5zZmf0DhatRPkAfEsuKQdwQpFTa5biZdldvVzvZMBqJh-AJeRJU3XZ-Jpllb2aSuN2uPqfOqeOnalvN7-qKfIr7al7dfpyhncleEvXy9vbDrDdh1qFvpMNHumk2fVVXHMj_6NrWOpJq_venKvzoLamaV4m-paqummpevXkDf-0IDqtgZaynTcQeehNWECi2HHu5gXsPuCGfoGNJ4HDwpiOB1BEIxS1FYL_0b96ApMZ6QN_oJx5SDbiJRAIIIXJiww62FMWyh51NHVhdIo8Cuw4T2AQGPTRc5910H_Zurw_yYNA53SgxO_jvv_8DHUWClXREaVXDrrOmA8M9CazRPALmp1qThkjAbdkRdh07AjHkMVq-yb_OZrP8_3dbinvoSQQ3BGtyvAMrwJ5Oj2o9pM4KRAosNnHcA5o0oHN7EPKNAMexfNlRlDm0dku5bKnzZpGEh2iozidEWOVrb_WAK0gKRKQ0RC-AHmyPG8r7RUwdRV3gQWm8yfRNcAbrPTWlOoI1NhvS4vXLAfpIW6uL83YnR_K04WQxkZwDovu3jncCBmPcg2dYOVyTW93APQh9HcgbgsZi7pom4k7AJgHe-QOOWmrhUxgQ8v2w48E1EKL1CRCEDPsGDIc9sIfEAbjVblCaFaNd5jpXjD1BQFsgpm-mQ78hAaEtRXTTU2UCIq9vbCSTLPsaVr1s5ivti9x02jMYI-8A84Z6WkgRt-TkDHBHuLV-k_c03Afr9ArDZiBR1CYwfipYLD1AwJissQF9gnttx3vV0MsrC73yp8UXWDh75fKDXj7V5dLrkvvZcri9fU_l72IOP0OHWwIE1boWZjoyj7p0cbb0YQ7W22TR2X-RXn6YvdwqkgwuzQ9t87fSYTUYDLi2ziZLcnVnz8m21qBCLW-Pz2muri5d76yk63V8nDRzd1DM1aNLxzvoS3ucy_s7j1WxQfYOeAqYuufrNWT1vQ1-A78CQ85Bj962VEpenC-_n4Nhn9CksrEyNVUCy6rYZn0UW50tbVnBE0ggauZwd3MLz4VgSRRAAu48LKts02q9kjCmM-PNLifDOkQ2JKK76e0ADYmZl3tjdgLPEDimrCfP0CD17IuHmNx6R2N_MQiynwfCR4E_Pvz58-wvXxbQqMRV6peWX1RlVV77c5ehJpdh2Ld2M8TcJ7CmvYp_FFaP-TMezmrTzbIqR8rivIcRdFjAEzRDnMOn29tealBlwx08l8WeE8EDLCstkFXTXvT8y6Ufkai1lrG2xBAHXx50ysCxh0caRqh0Uss4wo5yugY-tDZKOrihx55euPLk4Om4fbFHZx91bhTsRx8sU1KGoCeQG_jMqVNfIicEPe5hrXj7BINP1pVJNbkkdOol3AdHiZpirmWcZjWsqWVtlaTcnQxsipHjkQQ2g8DDCSOLAyMPEyO_vmTkFNWDBEaIpl4uA7PAmk5mm02ws6pXBeI6ypGC2-sQLOBOkw5ajqfGQKK_6vQakc6thvJITV4q3FMB1NOOIpDNIwCNoZCOKUVTQdtSpKakEI5gHMvIrGHvy7z5rX4_yC_SdHCtzGGieN7bD5e9fYZkDZL2jrJFB_ZCU7MLpQx-HvsKybvbdx9nt59md79ZVnoXJpoDP9Zg2HHUFb-4-4TvP-CyuqQnEjZ7WFb3Z4Z-esJDZrwkY_XqEFjl_ss4BLevYU0GB5kazrMmga8DScoM_Zym-DIyJbhXXmzK27TWW-moycVOg7n00S_lbFKVlGUP2_XcF-IP7aCV5Nw4CK4dnXNypbs_nnDy7hK-xoph7eczxzim3BNLLGn3EsTfHxJjVqmzXt0KY8xjo6fYo21qaKOmm-y9eZLp2uAGeZEy9YEO_WZQA4qkfauxzSYh197AT2O9-1PPzuPm6NjjPJhJIKPsguGG5uUopqNex1CMWrWmV_We_oij0NfpgK-Demri76e-fl0Xp8vfvaqL8d8Gh015VE6finAGV7WxRTeQho0LHrMpFg7L208JjaNl4jSzCr3l7cSm6-5U7jcdy-GNBdrI_RmA6nIoWVvUjINZS9DsbXUqa05ZZZI5wqqwu8q2hZq929HNdBOvcRgEt9TU45c1p64GhDg4OohYUyz5lqOh87yb4iDpEHhHr-TvUHplLl_RyY92wIcXDJ5MnbPxrPHsMAE0to6zq7z21aOrBIyaykN3xa5GNiL5RmdPIWQMbRpsoCNsHInAoos8yu1sUiF8-fyHorvsL2NCLGT0GAQI9dVU42PiMTRR0CHUc0OuLFQrEBpfcrKa9Zec4vNv6G1ffNgm-OcgCXZRI471We2O-fE77PyfyfL-VQVd0HV_NXV9X4HPS189_-P5fwEAAP__){ .md-button }

??? abstract "The source — `05-ai-systems/04-mcp-handshake.dgm`"

    ```dgm
    %% The handshake `cinegram mcp` performs. An agent host launches the server on
    %% stdin and stdout, agrees a protocol version with it, asks what it can do,
    %% and only then calls a tool — here `sheet`, which comes back as a picture of
    %% a whole scenario.
    %% ---
    %% Every message below is one the server in this repository actually sends or
    %% answers: five tools, one resource, and a `tools/call` that returns an image
    %% rather than text. The version pinned in the badge is the protocol revision
    %% the server negotiates.
    %% ---
    %% The flows carry no `label`. A sequence diagram draws its own message text,
    %% so a label would print a second copy on top of it — and where the same pair
    %% exchanges several messages in the same direction, `msg:` picks which arrow a
    %% flow travels rather than leaving the compiler to guess.
    sequenceDiagram
      participant A as Agent
      participant C as MCP Client
      participant S as cinegram mcp

      A->>C: I have a .dgm to check
      C->>S: initialize
      S-->>C: result: protocolVersion, capabilities
      C->>S: notifications/initialized
      C->>S: tools/list
      S-->>C: 5 tools, 1 resource
      A->>C: show me the whole scenario
      C->>S: tools/call sheet {path}
      S-->>C: image/png + cell manifest
      C-->>A: contact sheet

    scenario "launch, negotiate, call" { speed: 1.0 }

      step spawn "The host starts the server as a subprocess" {
        desc: "There is no port and no daemon. The client launches `cinegram mcp` and speaks JSON-RPC down its stdin and stdout, which is why the server needs no configuration beyond the command that starts it."
        flow A -> C { dur: 600ms, msg: 1 }
        note S "stdio transport\nno port, no server to run"
      }

      step initialize "The client opens with initialize" {
        desc: "The first message names the protocol version the client would like to speak and what it supports. Nothing else may be sent until this exchange has completed — a tool call before it is a protocol error."
        focus S
        flow C -> S { dur: 700ms, msg: 1 }
      }

      step negotiate "The server answers with the version it will speak" {
        desc: "The reply pins the revision for the whole session. A client that asked for something newer either accepts what it is offered here or closes the connection; there is no renegotiation later."
        flow S -> C { dur: 700ms, msg: 1, style: response }
        set S { badge: "2025-06-18", state: ok, color: "#16a34a" }
      }

      step ready "A notification closes the handshake" {
        desc: "`notifications/initialized` has no reply, because it is not a question. It is the client saying it has finished reading the server's capabilities, and it is the moment the session becomes usable."
        flow C -> S { dur: 500ms, msg: 2 }
      }

      step discover "The client asks what the server can do" {
        desc: "Five tools — lint, narrate, mermaid, frame and sheet — plus one resource, the language reference itself. Discovery is why the host needs no cinegram-specific code: the schemas arrive at runtime."
        seq {
          flow C -> S { dur: 600ms, msg: 3 }
          flow S -> C { dur: 600ms, msg: 2, style: response }
        }
        gauge S { label: "tools", value: 5 }
      }

      step call "The agent picks a tool and the client calls it" {
        desc: "The agent chose `sheet` from the schemas it was handed. The call carries `path` — or `source` for a draft that was never saved, never both, a rule the handler enforces rather than trusting the client to."
        seq {
          flow A -> C { dur: 500ms, msg: 2 }
          flow C -> S { dur: 600ms, msg: 4 }
        }
        focus S
      }

      step image "The result is a picture, not a paragraph" {
        desc: "`sheet` renders the scenario in a headless Chrome and answers with a PNG plus the manifest that maps each cell to its step. A model that can see the sheet can check the animation it just wrote in one look."
        seq {
          flow S -> C { dur: 700ms, msg: 3, style: response }
          flow C -> A { dur: 600ms, msg: 1, style: response }
        }
      }
    ```

← [rag pipeline](../05-ai-systems/03-rag-pipeline.md)  
