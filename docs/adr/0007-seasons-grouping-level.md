# Seasons as a required grouping level

A `Season` sits between a `Team` and its `Event`s: `Team → Season → Event → Match`. Every event and
match belongs to a season (`season_id` is required), and `Event` remains required on every match.

We kept `Event` required (rather than making it optional and letting matches hang directly off a season)
because the product targets competitive ultimate, which is organized as tournament play. Non-tournament
play is still representable by modeling it as an `Event` — a league night or scrimmage is a one-off
grouping of matches. Teams always play within a season (the app requires creating one before any event),
so a null `season_id` never arises.
