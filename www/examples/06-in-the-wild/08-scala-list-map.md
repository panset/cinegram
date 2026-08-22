<!-- Generated from 06-in-the-wild/08-scala-list-map.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# map applies the function to every element, in order

`List(1, 2, 3).map(_ * 2)`, one element at a time. scala/docs.scala-lang#1107 (https://github.com/scala/docs.scala-lang/issues/1107) has asked since 2019 for animated diagrams of the collection operations, and the latest proposal there is a proof-of-concept GIF of a single operation: map. This is that — as text, in the docs repo, with `cinegram record … -o map.gif` producing the GIF and the same file playable in a page.

<div class="cinegram" data-cinegram="06-in-the-wild/08-scala-list-map" data-height="900"></div>

[Edit in the playground](../../playground/#doc=nFbdbuO2En6VAQ8CJAeSY8l7cloterG5aFFg0QWK3FlBTYsjiQhFqiQVWwgW2IfoM_TB9kmKISXbcbJFsFeW-TMz_L5vPvKJPbIiSxhqb0dWsOVNKnXqW0x3Uonr5Q-pq7jiqZLOpx3vF6LpWMJMj_rtq2up0LFi_cT6t2_yrGAe954lTLCCXVzA5qN0_jJLIE9gdbXoeH_5B_wX8qtNAkYjoMIOtQfugYOXHS4gRL4WpnKLKQnXzX-ybPn_Ul9cwGXrfe-K6-tG-nbYLirTXb-65Vo6N6C7pp1X0HIH3D2gACd1hZAvsx9DvNpY4Fp23KMAIXljeefA1OBbhMoohZWXRoPp0XL6cglwLcK04h6dD2F6a3rjuKJxiyAdcBozdWrqtDK6wt7DL7_-TJE51dAoPMYsgECEu1a6EE068C338PXLX8AdEKYJSB2S0jHBYm8S2EnfwqaSGqlqsFgZK-Drl78hNSFiI-vNXJ4YKqmbEILqmM_geIdAbEOv-Mi3CikRh543uAh70zQNv3ctgkU3KA_EOxW5aaXATSpQE4MhnOfWh-DIq_bAL611rdltdAhFZ2oR6kFHcGN56ED6BJwJkxa5QAsOMS7WuIt5t0jn2A5SRegtJ8wJMA1c0bYxzi5CyVL3w7FijY-01gxVi6IA2XWD51uppB9n5Dn4dkYqKoNKrCh65Qeu1Ah0lAS2Iyjkj_NahbUnxQydBiVjbRMqqMWi1LUyu6olfD7-XmoAN2wby_sWpF6X7FmnlOyeVgAIaScB3t3GkX22zqbJfb7O58_VehU-UYtS00e9XpcsNFvJ7u_j2CGjGfy6ZJHNAkLqPIF3Cdz8W-oxO-Qb8_W7-XO1vnmeep9BmpbDcrlCqMNAfj6wOhuoj__H7HwgPx9YUR5XoeZWGihZx3vgfa8kngnLGyDCx1mIoYuMFWhLBk_geiQVZIslfJ4Q8thPIo5hPX9A0sQhJGmbBz1RiAkqdFUBJfvNROmQ27S871GjgBEnJR5CBFZikyNszX5u7U4KofD9JP9jpwmDDrTxgHvp_KJkMSs1H4xZAmOeBEziYNMq2bQ-4gq0D6GGkgmsyRTWt_eXdQEf4KcJzdurqID17X0ERQosYIvK7AgVeAZNLa0jaOg8LXIBjQmYWzM0bZx9CQvhuOPqIR44HKm2pgv_KMgCMkKj586hIM5OOTzaLdduhzZ0yBYr001khzKCrc4WcYbQCToO_5yLA6B-DFqdhQhPoPgWFdWclSwBMdgC_rdcdi4icdhVw1Gtp7vy57sScH5UWBCbvdEO5zDTD_nIpPdT5vbEaXYOvcPKaBGx15Ml7v2s7JeofyKZB1wtOrSPpPRg0TFO9MWD857MmMH3g1-QWCa_tBh6xiUgrOkdGAti6JWs6AIMt5QM7mo0plKHqz01wzkP3-Ygf52D_K0c5Ke73n0PB_kLDkg1-TkHvpWWKPhweAG8pvdnzU43SrAmAbjnlVcj0HMAerRHV_KtRQwPIBdeJPH_NO3eByrIe0KjUshPl_rqAO8riK5eR3T1VkRXp7tuvgfRF360X819eIqooGdgye52JrSuS4LRGY2vw2qsbKTmCp5dl4TIoKuW6wbFiR6dl2p-khl9uKWTaWJw9N6JQrfoB0t2fbSR9xRFSBEq6oyQ9Qhcj8HiydF5yLELvyfvCFDGPDhQ8gFPGmBGIbTH4E-8WWooWSzo7A3wDTNmn-8__xMAAP__){ .md-button }

??? abstract "The source — `06-in-the-wild/08-scala-list-map.dgm`"

    ```dgm
    %% `List(1, 2, 3).map(_ * 2)`, one element at a time. scala/docs.scala-lang#1107
    %% (https://github.com/scala/docs.scala-lang/issues/1107) has asked since 2019
    %% for animated diagrams of the collection operations, and the latest
    %% proposal there is a proof-of-concept GIF of a single operation: map. This
    %% is that — as text, in the docs repo, with `cinegram record … -o map.gif`
    %% producing the GIF and the same file playable in a page.
    %% ---
    %% The result list is `hide`-den at the start and each element is `show`n
    %% as the function produces it, so the reader sees the new list being built
    %% rather than already built. The input list is never touched: immutability
    %% is a thing the animation can actually show, by leaving the left column lit
    %% at the end.
    flowchart LR
      subgraph in["List(1, 2, 3)"]
        direction TB
        x1[1]
        x2[2]
        x3[3]
      end

      f[["_ * 2"]]

      subgraph out["result: List(2, 4, 6)"]
        direction TB
        y1[2]
        y2[4]
        y3[6]
      end

      x1 --> f
      x2 --> f
      x3 --> f
      f --> y1
      f --> y2
      f --> y3

    scenario "map applies the function to every element, in order" { speed: 1.0 }

      step start "map takes a function and a list" {
        desc: "Nothing has happened yet. The function _ * 2 is the box in the middle; the result list does not exist."
        hide y1, y2, y3
        highlight f
        note f "def map[B](f: A => B): List[B]" { side: below }
      }

      step first "The head goes through first" {
        desc: "map walks the list from the head. 1 is passed to the function, and the answer, 2, becomes the head of a new list."
        hide y2, y3
        seq {
          flow x1 -> f { label: "1", dur: 500ms }
          flow f -> y1 { label: "2", dur: 500ms, style: response }
        }
        show y1
        highlight x1, y1
      }

      step second "Then the next element" {
        desc: "Order is preserved: the second input produces the second output. map never reorders, drops or duplicates — it is one-in, one-out."
        hide y3
        seq {
          flow x2 -> f { label: "2", dur: 500ms }
          flow f -> y2 { label: "4", dur: 500ms, style: response }
        }
        show y2
        highlight x2, y2
      }

      step third "And the last" {
        desc: "The function is applied exactly once per element, three times for three elements; map on a List is O(n)."
        seq {
          flow x3 -> f { label: "3", dur: 500ms }
          flow f -> y3 { label: "6", dur: 500ms, style: response }
        }
        show y3
        highlight x3, y3
      }

      step done "Two lists, not one" {
        desc: "The original List(1, 2, 3) is unchanged — it is still there on the left, still usable. map returned a new list; it did not modify anything. That is what immutability looks like."
        highlight in, out
        note in "still List(1, 2, 3)" { side: below }
      }
    ```

← [architecture canvas](../06-in-the-wild/07-architecture-canvas.md)  
