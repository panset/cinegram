# Storyboard frames

Mockups for the `storyboard` block in `../oidc-login.dgm`. Cinegram supplies the
synchronisation, not the content — a frame is whatever picture you point it at —
so nothing here is generated and nothing here is required.

The convention these follow, for whatever comes next:

- **SVG, hand-written.** It diffs, it scales to whatever width the panel gets,
  and it stays legible next to a diagram in a way a screenshot does not.
- **One geometry for the whole set.** Every frame is a 480×320 page with a 34px
  address strip, so the panel does not jump as the scenes change. The address is
  usually the most load-bearing detail in the frame: half the point of an OIDC
  storyboard is showing *whose* domain the password is typed on.
- **Flat colour, no gradients, no animation.** The frame is a still beside
  something that is already moving; anything that competes with the diagram is
  working against it.
- **A `role="img"` and an `aria-label`.** The panel renders these as `<img>`
  with an empty `alt`, since the caption beside them is the accessible text —
  but the label costs a line and helps anyone who opens the file directly.

The palette matches `runtime.css`'s light theme, so the frames sit comfortably
in the panel: `#ffffff` page, `#f1f3f5` chrome, `#dfe3e8` borders, `#1c2024`
text, `#6b7480` muted, `#2f6feb` the app's accent, `#7c3aed` the provider's,
`#16a34a` success.
