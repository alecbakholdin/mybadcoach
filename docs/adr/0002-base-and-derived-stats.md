# Statistics split into base (credited) and derived (computed) fields

`reducePoint` credits only seven base stats per player (touches, completions, drops, turnovers,
blocks, goals, assists); three stats (receptions, throw attempts, plus-minus) are computed upward
during aggregation and never stored.

This deviates from the original spec, which stored ten fields and credited an on-field plus-minus to
every lined-up player on each score. We chose derived computation because the three fields are pure
functions of the base seven, so storing them invites inconsistency; and we redefined plus-minus as an
individual production metric (goals + assists + blocks − turnovers − drops) rather than an on-field
plus/minus.
