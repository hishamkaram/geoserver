#!/bin/bash
PGPASSWORD=golang
./wait.sh -h localhost -p 5432 -t 600 -- pg_restore -d cartoview_datastore ../testdata/lbldyt