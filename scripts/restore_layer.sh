#!/bin/bash
PGPASSWORD=golang
$PWD/scripts/wait.sh -h localhost -p 5436 -t 600 -- pg_restore -p 5436 -d gis $PWD/testdata/lbldyt