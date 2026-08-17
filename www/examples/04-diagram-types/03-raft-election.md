<!-- Generated from 04-diagram-types/03-raft-election.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# leader failure and re-election

Raft leader election in a five-node cluster.

<div class="cinegram" data-cinegram="04-diagram-types/03-raft-election" data-height="990"></div>

[Edit in the playground](../../playground/#doc=rFZPb9vGEv8qAz4YuUiKZMoJnm55D36nhx7SoD1YATwih-TWy1l2Z2mGNQL0Q_QT9pMUsyvRsmU7TtGLvVoOh7O_P7Nzl91mm9UsIw5-zDbZcj0vDdYe23kYO5K3y3zusQpzslQE43hR1m02y1xH_B3hlbEk2ebqLuu-462QbbJAX0I2y8psk52dwUesAljCkjwcgsEwIFTmlubsSoLC9hLIL7Z8dgbz-Tz-_9QQdM5wAFdBaIwAfcG2swRGIDQYIDTTq28EJGCIzyzVZmcJMADyGHNJ4fsddE5MrBV-blyK1LJmMMRs5FsYCNBTrI9LaNwALfIIty6QAMZcBXJpSv1WgwKFs3ooKuN77LQqw7XA6HoNBcMVeai8awGhQx9MYSnWH7NpDrSesByhdkzQoQT48_c_QJwecIx5r4XCdSzpusa-put0Wq3cFI3GpdJc2xlLHm6IOgFlHHoOxoK4lmJh4KmzWJCACYstV9YNRYM-wKf_bBlA-l3tsWsOuF5F-v6bfnzWCABeXfHqsD6_4vPDOr_i_LBeX_H6sL644ou4Ji63rAtewXy-7ZfLnIDPT3byk531yc7FPlN-tLc62Tk_2Vmf7MRMUhCjNw622V6sFRrbe4qoe5qEvs3gDqQjKjewWizha6pDAnX6R3ncZry6l1dSEqEPO8IgUbWVs9YN5EWzJZRKkmID2-zDA8c0zpZKlUzihd0IAe2N4XoBl7fkxylbcoV-SgAZPnQdcXnJwRsSGIzyH1NNNgymJdcHFdMo6sh9osU2S0UJBUX9DnZY1rSZwNlms6TAzeShwlnnNeJfq3eYr3GbKTSaJCo2pbG4I6tBajZNcou2pw2sDrGqx8jzRODxWw9OpK-Xvd_AxXLZyrMJcrh7Tdj6dWEXT4QdK6B0Ayf-JbhOwJN0jkvD9SnTP7jkSGR2PaslER5qbxGb4CQWcGxHcDshf6u6BNwJcTG1ASMwNGMU2NA4S9B5F1zhrD7Z9cYGcNp4D7R7DE1SDacH7IKpTIGxSR400HNSwb54007rJ7GeyJokD9ZJmNh6p8Al9fSyiSc9hfFQ4Tbj_M0jwXqojCfRv5r2MaqXj8QtsYF65NK1RqiEzlNhhOyYOiwqKNq5S4JeerR2BE9YNCTwG3mXvrPQwxkuPLXEexOrhmf7q6FyXq1FtppFv6PcpCCnEAsEB1h7oofOyo-dNd0rR-aa9o79Vf77_fvluxN_5S_46_yl2HiCo-BttoK3cHH_gVPO82cM-pF-7UnCTy6d4jlD5S_77lgJCV5tiy3-4rwJIyDL063zEovmvhnWHpUpDNA6SSRrMuhU8pE65X9w9yCL3vtKmmCb6NVneqfvXGhgMKyG9EQ6kej0or7C48LKJCidVdj1dTPR_Trc82_hfv6M1-JZqZwwX1_sTTZa2uybkNBDEtb_QK5jpmg_B6lnYUeFa2m6AtUeEc_zU9a0xUUTQOtuSfRV7IUiEV67mG6XC-DVm0M-aUwHA2p7DcZTGWNbp9YEE2DQ_2IscUhu5H2n3dO7n2RBgrEWCoumlWTUwchLFv37l983iH-a8663cpiGjpHWEU67NME2-7-rp9_a9TxJr8D3XMbWTsA0wKHwx9j_T8fShjwdz9LxtsDiRptWdEOaavbzdZwWvBJteBZdFYm1GPajztRN9S5cwI9x9I759GnlfGg2MW-EVeJuRElAcEyzuE74vo8jfDRU4nYW5-sY0KCOAsq8A2TTamkWJUzkvb5nvXqoyF83LeTfnBayr5-__hUAAP__){ .md-button }

??? abstract "The source — `04-diagram-types/03-raft-election.dgm`"

    ```dgm
    %% Raft leader election in a five-node cluster.
    %% ---
    %% The point of this example is that the cluster's state is legible at any
    %% scrub position. Who is leader, what term we are in and how many votes a
    %% candidate has collected are not things you can infer from a particle that
    %% has already gone past — so they are `set` and `gauge` state, which the
    %% compiler keeps open until something replaces it.
    flowchart TB
      subgraph cluster[Raft Cluster]
        n1[n1]
        n2[n2]
        n3[n3]
        n4[n4]
        n5[n5]
      end

      n1 --> n2
      n1 --> n3
      n1 --> n4
      n1 --> n5

      n3 --> n1
      n3 --> n2
      n3 --> n4
      n3 --> n5

    scenario "leader failure and re-election" { speed: 1.0 }

      step steady "n1 is leader and heartbeats the followers" {
        desc: "A Raft leader holds its position by talking. Every follower that hears an AppendEntries within its election timeout stays a follower."
        set n1 { badge: "leader", state: leader, color: "#16a34a" }
        gauge n1 { label: "term", value: 1 }
        flow n1 -> n2 { label: "AppendEntries", dur: 500ms }
        flow n1 -> n3 { dur: 500ms }
        flow n1 -> n4 { dur: 500ms }
        flow n1 -> n5 { dur: 500ms }
      }

      step down "n1 stops responding" {
        desc: "Nothing announces a leader failure. The followers only observe an absence, which is why the whole protocol is built on a timeout rather than on a notification."
        unset n1
        dim n1
        flow n1 -> n3 { label: "heartbeat lost", dur: 600ms, status: fail }
      }

      step timeout "n3's election timer fires first" {
        desc: "Election timeouts are randomised precisely so that one node usually reaches zero first. n3 increments the term, votes for itself, and asks the others to agree."
        set n3 { badge: "candidate", state: candidate, color: "#d97706" }
        gauge n3 { label: "term", value: 2 }
        gauge n3 { label: "votes", value: "1 / 5" }
        dim n1
        flow n3 -> n2 { label: "RequestVote", dur: 500ms }
        flow n3 -> n4 { dur: 500ms }
      }

      step votes "A majority answers" {
        desc: "Each follower grants at most one vote per term, so two candidates in the same term cannot both win. Three of five is a majority and that is enough."
        gauge n3 { label: "votes", value: "3 / 5" }
        dim n1
        flow n2 -> n3 { label: "granted", dur: 450ms, style: response }
        flow n4 -> n3 { label: "granted", dur: 450ms, style: response }
      }

      step elected "n3 becomes leader for term 2" {
        desc: "The badge moves because the role moved. n1's leadership was retired the moment it went silent, and nothing in the diagram still claims otherwise."
        set n3 { badge: "leader", state: leader, color: "#16a34a" }
        gauge n3 { label: "votes", value: "" }
        dim n1
        pulse n3
      }

      step replicate "Log replication resumes under the new leader" {
        desc: "From here the cluster is back to the steady state it started in, one term later and one node down. Scrub back and forth: the badges and gauges say what is true at that moment, not what happened to animate last."
        dim n1
        flow n3 -> n2 { label: "AppendEntries", dur: 500ms }
        flow n3 -> n4 { dur: 500ms }
        flow n3 -> n5 { dur: 500ms }
      }
    ```

← [tcp connection](../04-diagram-types/02-tcp-connection.md)  
