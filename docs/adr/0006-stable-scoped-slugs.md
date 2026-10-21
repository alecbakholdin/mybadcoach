# Stable, scoped slugs for addressable entities

`teams`, `seasons`, `events`, `matches`, and `players` each carry a `slug`, generated once from a name by
a single
server-side `slugify` package. A slug is stable: renaming an entity does not change its slug, so a shared
link keeps working. Players derive their slug from the primary name (`names[0]`); matches have no
free-form name, so theirs derives from `opponent_name` plus the scheduled date as a disambiguator.

Uniqueness is scoped to the URL hierarchy: team slugs are globally unique; season and player slugs are
unique within their team; event slugs are unique within their season; match slugs are unique within their
event. Collisions append a numeric suffix
(`my-team`, `my-team-2`). Slug generation is server-side only — clients consume slugs, never predict them.
