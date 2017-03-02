#!/bin/bash
CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CHALLENGE_FILE="$CURRENT_DIR/sample_challenge.gta.enc"
GTA_BIN="$CURRENT_DIR/go-term-adventure"

source $CURRENT_DIR/challenger.sh
