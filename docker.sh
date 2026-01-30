#!/bin/bash
docker build -f docker/Dockerfile -t greenwave .

# Push to GitHub Container Registry:
# docker tag greenwave ghcr.io/lddl/greenwave:latest
# docker push ghcr.io/lddl/greenwave:latest

# Push to Docker Hub:
# docker tag greenwave dimahkiin/greenwave:latest
# docker push dimahkiin/greenwave:latest
