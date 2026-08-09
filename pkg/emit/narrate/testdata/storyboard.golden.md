# storyboard

## sign in

`main` · flowchart · opens here

### Scenario: sign in

`s0` · 1.6s long

#### 1. The reader opens the app

0.0s–0.5s

Nothing has happened yet except a page load.

- **user → app** carries a message (0.0s–0.5s)
- the storyboard shows *The app's sign-in page* (0.0s–0.5s)

#### 2. The app hands off to the provider

0.5s–1.1s

- **app → idp** carries a message (0.5s–1.1s)
- the storyboard shows *The provider's credential form* (0.5s–1.1s)

#### 3. And land back signed in

1.1s–1.6s

- **idp → user** carries a message (1.1s–1.6s)
- the storyboard shows *home* (1.1s–1.6s)
