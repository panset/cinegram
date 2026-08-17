# authorization code flow

## authorization code flow

`oauth-login` · flowchart · opens here

### Scenario: authorization code flow

`s0` · 6.1s long

#### 1. The user asks to sign in

0.0s–0.6s

The application has no credentials of its own to check, so it does not try. It hands the browser off to the identity provider and steps out of the conversation entirely.

- **browser → app** carries "GET /login" (0.0s–0.6s)
- **app** is highlighted (active) (0.0s–0.6s)

#### 2. The app redirects to the authorization server

0.6s–1.3s

The redirect carries a client id, a redirect URI and a scope — but no secret, because anything in a redirect is visible to the user and to anything watching. What it asks for is permission, not proof.

- **app → auth** carries "302 → /authorize" (0.6s–1.3s)
- **auth** is highlighted (active) (0.6s–1.3s)

#### 3. The user authenticates and consents

1.3s–2.1s

This is the only moment a password exists, and it exists between the user and the provider alone. The application never sees it, which is why a breach of the application cannot leak it.

- a note on **auth**: "user signs in\nand approves the scopes" (1.3s–2.1s)
- **auth** pulses (1.3s–2.1s)

#### 4. The browser comes back carrying a code

2.1s–2.9s

The provider redirects back with a short-lived authorization code. A code is deliberately useless on its own: intercepting it buys nothing without the client secret that only the application holds.

- **auth → app** carries "302 ?code=Ab3x…" (2.1s–2.9s)
- **app** is highlighted (active) (2.1s–2.9s)

#### 5. The app trades the code for tokens

2.9s–3.7s

Now the application speaks to the provider directly, back channel, server to server. It sends the code together with its client secret, and that pairing is what proves the exchange is genuine.

- **app → auth** carries "POST /token + secret" (2.9s–3.6s)
- **auth → tokens** carries "mint & record" (3.2s–3.7s)

#### 6. Tokens come back over the back channel

3.7s–4.4s

The access token returns on a channel the browser was never part of, so it is never exposed to the address bar, to history, or to a referrer header. The one-time code is burned in the process.

- **auth → app** carries "access + refresh token" (3.7s–4.4s)
- **app** is highlighted (3.7s–4.4s)

#### 7. The app calls the API on the user's behalf

4.4s–5.1s

The API trusts the token, not the caller: it validates the signature and the scopes and never contacts the application. That is what lets the same token work across services that have never heard of each other.

- **app → api** carries "Authorization: Bearer …" (4.4s–5.1s)
- **api** is highlighted (active) (4.4s–5.1s)

#### 8. The user is signed in

5.1s–6.1s

The application ends up able to act for the user without ever having held the user's password. Every step above exists to make that sentence true.

- **api → app** carries "200 OK" (5.1s–5.6s)
- **app → browser** carries "200 OK" (5.6s–6.1s)
