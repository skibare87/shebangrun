#!/bin/bash
set -e

IMAGE="docker.dingo.bar/shebang:latest"

echo "Building $IMAGE..."
docker build -t "$IMAGE" .

echo "Pushing $IMAGE..."
docker push "$IMAGE"

echo "Done!"
