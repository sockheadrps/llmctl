# Templates

Templates are pre-filled profile presets. They are the fastest way to create a new profile when you already know the rough shape you want.

---

## Built-In Templates

The current built-in templates are:

- `Blank` - starts with the default profile form values
- `Fast Inference` - favors speed and smaller working sets
- `High Quality` - uses larger context and more conservative sampling
- `Coding` - tuned for long-lived coding sessions
- `Low VRAM` - keeps memory usage down for smaller GPUs

These are starting points, not fixed policies. You can edit every field after creation.

---

## How To Use A Template

When you create a new profile, llmctl lets you start from a template or from blank parameter values.

1. Go to the **Models** tab.
2. Expand a model.
3. Select **+ New Profile**.
4. Choose a template, or leave the fields blank and fill them in yourself.
5. Press `Enter` to save.

The profile opens in the normal form after that, so you can still change anything before saving.

---

## When To Use A Template

- You want a quick first pass instead of building a profile from scratch.
- You are comparing a few common setups on the same model.
- You want a safe low-VRAM or long-context default before tuning further.

## When Not To Use One

- You already know the exact flags you want.
- You are importing a copied launch command.
- You need a highly custom experimental setup.

In those cases, a blank profile is usually cleaner.
