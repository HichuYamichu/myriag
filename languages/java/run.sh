#!/bin/sh
set -e

printf %s "$1" > Main.java
javac Main.java && java Main || true
