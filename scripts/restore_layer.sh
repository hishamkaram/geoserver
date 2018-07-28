#!/bin/bash
PGPASSWORD=golang
./wait.sh -h localhost -p 5436 -t 600 -- pg_restore -p 5436 -d cartoview_datastore ../testdata/lbldyt