@echo off
REM Tiny batch script to drop into the Gorrent CLI inside the Docker container
docker exec -it gorrent /gorrent %*
