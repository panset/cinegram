# raft-election

## leader failure and re-election

`main` · flowchart · opens here

### Scenario: leader failure and re-election

`s0` · 3.4s long

#### 1. n1 is leader and heartbeats the followers

0.0s–0.5s

A Raft leader holds its position by talking. Every follower that hears an AppendEntries within its election timeout stays a follower.

- **n1 → n2** carries "AppendEntries" (0.0s–0.5s)
- **n1 → n3** carries a message (0.0s–0.5s)
- **n1 → n4** carries a message (0.0s–0.5s)
- **n1 → n5** carries a message (0.0s–0.5s)

#### 2. n1 stops responding

0.5s–1.1s

Nothing announces a leader failure. The followers only observe an absence, which is why the whole protocol is built on a timeout rather than on a notification.

- **n1** is dimmed (0.5s–1.1s)
- **n1 → n3** carries "heartbeat lost", which fails (0.5s–1.1s)

#### 3. n3's election timer fires first

1.1s–1.6s

Election timeouts are randomised precisely so that one node usually reaches zero first. n3 increments the term, votes for itself, and asks the others to agree.

- **n1** is dimmed (1.1s–1.6s)
- **n3 → n2** carries "RequestVote" (1.1s–1.6s)
- **n3 → n4** carries a message (1.1s–1.6s)

#### 4. A majority answers

1.6s–2.0s

Each follower grants at most one vote per term, so two candidates in the same term cannot both win. Three of five is a majority and that is enough.

- **n1** is dimmed (1.6s–2.0s)
- **n2 → n3** carries "granted" (1.6s–2.0s)
- **n4 → n3** carries "granted" (1.6s–2.0s)

#### 5. n3 becomes leader for term 2

2.0s–2.9s

The badge moves because the role moved. n1's leadership was retired the moment it went silent, and nothing in the diagram still claims otherwise.

- **n1** is dimmed (2.0s–2.9s)
- **n3** pulses (2.0s–2.9s)

#### 6. Log replication resumes under the new leader

2.9s–3.4s

From here the cluster is back to the steady state it started in, one term later and one node down. Scrub back and forth: the badges and gauges say what is true at that moment, not what happened to animate last.

- **n1** is dimmed (2.9s–3.4s)
- **n3 → n2** carries "AppendEntries" (2.9s–3.4s)
- **n3 → n4** carries a message (2.9s–3.4s)
- **n3 → n5** carries a message (2.9s–3.4s)

#### Standing state

State that outlives the step that set it, with the window it holds for.

- **n1** is badged "leader" and marked *leader* (0.0s–0.5s)
- **n1** reads term = "1" (0.0s–0.5s)
- **n3** is badged "candidate" and marked *candidate* (1.1s–2.0s)
- **n3** reads term = "2" (1.1s–3.4s)
- **n3** reads votes = "1 / 5" (1.1s–1.6s)
- **n3** reads votes = "3 / 5" (1.6s–2.0s)
- **n3** is badged "leader" and marked *leader* (2.0s–3.4s)
