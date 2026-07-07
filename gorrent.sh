#!/bin/bash
# Tiny shell script to drop into the Gorrent CLI inside the Docker container
docker exec -it gorrent /gorrent "$@"
