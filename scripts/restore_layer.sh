#!/bin/bash
export PGPASSWORD="golang"
$PWD/scripts/wait.sh -h localhost -p 5436 -t 600 -- pg_restore --no-owner -h localhost -U golang -p 5436 -d gis $PWD/testdata/lbldyt