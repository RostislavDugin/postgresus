# Writing ADRs

ADRs live here, numbered `NNNN-slug.md`. Start from [`0000-adr-template.md`](0000-adr-template.md)
and drop every section you don't need.

## What an ADR is for

Record **why** a decision was made, so nobody re-litigates it in six months.

Write one only if all three hold:

1. It's expensive to reverse.
2. A future reader would look at the code and ask "why on earth like this?"
3. There was a real alternative and we picked against it for a reason.

Obvious decisions don't get an ADR.

## Write the substance, not the details

The code is the source of truth for *how*. The ADR is the source of truth for *why*. Never restate
the code in prose.

Keep out:

- Env-var names, config keys, flags, defaults, limits
- Function, package and type names, call chains, file layout
- Anything you'd have to update after a rename

Keep in:

- The problem and the constraint that forced the decision
- What we decided, in plain words
- Alternatives we rejected, and why they lost
- Consequences we're knowingly accepting

Rule of thumb: if a sentence would go stale after a refactor, it belongs in the code, not here.

## Style

- One page. If it's longer, you're explaining the implementation.
- Plain language. A new engineer should get it without opening the repo.
- Present tense, stated as fact: "We export logs over OTLP", not "It was decided that...".
- English only.
- Name the decision in the title, not the technology: "Export logs over OTLP" beats "OpenTelemetry".

## Keeping them honest

An ADR is a snapshot of a decision, not living documentation. When a decision is replaced, don't
edit history: write a new ADR and mark the old one `Superseded by ADR-NNNN`.
