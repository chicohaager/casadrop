#!/bin/sh
# Tests the awk expression entrypoint.sh uses to pull `src <ip>` out of
# `ip -4 route get`.
#
# Why this exists as its own test: the original code used `grep -oP 'src \K\S+'`,
# and BusyBox grep (what the alpine runtime image actually ships) has no -P. It
# printed a usage dump into the log on every single start and silently fell
# through to the hostname fallback. Nothing caught it, because nothing in the
# repo ever fed real `ip route get` output through the parser. This does.
set -eu

# The expression under test, kept byte-identical to scripts/entrypoint.sh.
# Verified against the source below so the two cannot drift apart unnoticed.
AWK_EXPR='{for(i=1;i<NF;i++) if($i=="src" && $(i+1) ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$/) {print $(i+1); exit}}'

script_dir="$(cd "$(dirname "$0")" && pwd)"
entrypoint="$script_dir/../scripts/entrypoint.sh"

if ! grep -qF "$AWK_EXPR" "$entrypoint"; then
    echo "FAIL: the awk expression in this test no longer appears in $entrypoint"
    echo "      Someone changed the parser without updating its test."
    exit 1
fi

fails=0
check() {
    name="$1"; input="$2"; want="$3"
    got="$(printf '%s\n' "$input" | awk "$AWK_EXPR")"
    if [ "$got" = "$want" ]; then
        printf 'ok   %s -> %s\n' "$name" "${got:-<empty>}"
    else
        printf 'FAIL %s: got %s, want %s\n' "$name" "${got:-<empty>}" "${want:-<empty>}"
        fails=$((fails + 1))
    fi
}

# Real output shapes, collected from actual hosts.
check "plain host route" \
    "1.1.1.1 via 192.168.1.1 dev eth0 src 192.168.178.23 uid 1000" \
    "192.168.178.23"

check "docker bridge" \
    "1.1.1.1 via 172.19.0.1 dev eth0 src 172.19.0.2 uid 0" \
    "172.19.0.2"

check "no uid suffix (older iproute2)" \
    "1.1.1.1 via 10.0.0.1 dev wlan0 src 10.0.0.42" \
    "10.0.0.42"

check "table/metric fields after src" \
    "1.1.1.1 via 192.168.0.1 dev eno1 src 192.168.0.9 metric 100 uid 1000" \
    "192.168.0.9"

# No src field at all — must print nothing so the caller falls back to
# `hostname -i` rather than exporting a stray token.
check "route without src" \
    "1.1.1.1 dev tun0 uid 1000" \
    ""

check "empty input" "" ""

# A device literally named "src" must not be mistaken for the keyword's value.
check "interface named src" \
    "1.1.1.1 via 192.168.1.1 dev src src 192.168.178.7 uid 0" \
    "192.168.178.7"

if [ "$fails" -ne 0 ]; then
    echo "$fails case(s) failed"
    exit 1
fi
echo "all LOCAL_IP parser cases passed"
