#!/usr/bin/env bash
# Verify the Markdown conversion of the 2026-27 USAU rulebook.
# Expected section titles come from the rulebook's own table of contents
# (independent of the conversion script).
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MD="$ROOT/docs/reference/usau-rules-2026-27.md"
TXT="$ROOT/docs/reference/usau-rules-2026-27.txt"
GLOSSARY="$ROOT/GLOSSARY.md"

fail=0
check() {
  if eval "$2"; then
    echo "ok   - $1"
  else
    echo "FAIL - $1"
    fail=1
  fi
}

check_heading() {
  local title="$1"
  if grep -qxF -- "## $title" "$MD"; then
    echo "ok   - heading ## $title"
  else
    echo "FAIL - heading ## $title"
    fail=1
  fi
}

check "converted markdown exists" "[ -f \"$MD\" ]"

if [ ! -f "$MD" ]; then
  echo
  echo "Cannot continue: $MD is missing."
  exit 1
fi

check "document title is an H1" "grep -qxF '# 2026-2027 Official Rules of Ultimate' \"$MD\""

sections=(
  "1. Introduction"
  "2. Spirit of the Game"
  "3. Definitions"
  "4. Playing Field"
  "5. Equipment"
  "6. Game Structure"
  "7. Timeouts"
  "8. Player Substitutions"
  "9. The Pull"
  "10. Restarting and Continuing Play"
  "11. In- and Out-of-bounds"
  "12. End Zone Possession"
  "13. Scoring"
  "14. Turnovers"
  "15. Stalling"
  "16. Marking Violations"
  "17. Making and Resolving Calls"
  "18. Travels"
  "19. Picks"
  "20. Fouls"
  "21. Positioning"
  "22. Observers"
  "23. Etiquette"
)

for title in "${sections[@]}"; do
  check_heading "$title"
done

appendices=(
  "Appendix A: Field Diagram"
  "Appendix B: Mixed Rules and Adaptations"
  "Appendix C: Misconduct System"
  "Appendix D: Hand Signals"
  "Appendix E: Youth Rules Adaptations"
  "Appendix F: Beach Ultimate Rules Adaptations"
  "Appendix G: Ultimate 4's Rules Adaptations"
)

for title in "${appendices[@]}"; do
  check_heading "$title"
done

check "exactly 23 numbered section headings" "[ \"\$(grep -cE '^## [0-9]{1,2}\\. ' \"$MD\")\" -eq 23 ]"
check "exactly 7 appendix headings" "[ \"\$(grep -cE '^## Appendix [A-G]: ' \"$MD\")\" -eq 7 ]"
check "preface heading present" "grep -qxF '## Preface' \"$MD\""

# Section numbers appear in ascending order.
check "sections are in rule order" "awk '/^## [0-9]+\\. /{n=\$2; sub(/\\./,\"\",n); if (n+0 <= prev) exit 1; prev=n+0}' \"$MD\""

# Sub-rule numbers must remain as body text so grep finds them by number.
for rule in "9.A." "9.C.1." "17.C.5" "17.C.6.b.1.a." "20.B.2.c." "C1.D.3.b." "3.Q."; do
  check "sub-rule $rule is grep-able" "grep -qF '$rule' \"$MD\""
done

# Artifacts from the PDF text extraction must be gone.
check "no form-feed characters" "! grep -q \$'\\f' \"$MD\""
check "no table-of-contents dot leaders" "! grep -qE '\\.{5,}' \"$MD\""
check "no standalone page-number lines" "! grep -qE '^[[:space:]]*[0-9]{1,2}[[:space:]]*\$' \"$MD\""
check "no unclosed PDF page-break indentation (4+ leading spaces)" "! grep -qE '^    ' \"$MD\""

# The output should retain essentially all body content: same order of
# magnitude as the source, not a stub or an accidental duplicate.
input_lines=$(wc -l < "$TXT" | tr -d ' ')
output_lines=$(wc -l < "$MD" | tr -d ' ')
check "output line count sane vs input ($output_lines vs $input_lines)" "[ \"$output_lines\" -ge 2000 ] && [ \"$output_lines\" -le 3000 ]"

# Spot-check body text from early, middle, and late parts of the rulebook.
check "body text preserved: start-of-play rule" "grep -qF 'Play starts at the beginning of each half and after each goal with a pull.' \"$MD\""
check "body text preserved: observer rule" "grep -qF 'Observers may be used if desired by the captains or the event organizer.' \"$MD\""
check "body text preserved: misconduct appendix" "grep -qF 'This appendix describes the system for handling misconduct within a game that is using observers.' \"$MD\""

# Glossary links to the converted document.
check "GLOSSARY.md links to the markdown rulebook" "grep -qF 'docs/reference/usau-rules-2026-27.md' \"$GLOSSARY\""

if [ "$fail" -ne 0 ]; then
  echo
  echo "Verification FAILED"
  exit 1
fi

echo
echo "All checks passed."
