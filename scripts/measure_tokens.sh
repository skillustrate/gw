#!/usr/bin/env bash
# Token-efficiency measurement script (Section 20)
set -euo pipefail

echo "================================================================"
echo "          Token Efficiency & Footprint Measurement              "
echo "================================================================"

RAW_GIT_STATUS=$(git status 2>&1 || true)
RAW_GIT_BRANCH=$(git branch -vv 2>&1 || true)
RAW_TOTAL="${RAW_GIT_STATUS}"$'\n'"${RAW_GIT_BRANCH}"
RAW_CHARS=$(echo -n "${RAW_TOTAL}" | wc -c)
RAW_TOKENS=$(( RAW_CHARS / 4 ))

GW_JSON=$(./bin/gw inspect --json 2>&1 || true)
GW_CHARS=$(echo -n "${GW_JSON}" | wc -c)
GW_TOKENS=$(( GW_CHARS / 4 ))

echo "Raw multi-command shell output:"
echo "  - Characters: ${RAW_CHARS}"
echo "  - Est Tokens: ~${RAW_TOKENS}"
echo ""
echo "Deterministic 'gw inspect --json' envelope:"
echo "  - Characters: ${GW_CHARS}"
echo "  - Est Tokens: ~${GW_TOKENS}"
echo "================================================================"