#!/bin/bash

# Check if the container is running
if docker-compose ps -q media-server | grep -q .; then
    echo "Container is running, restarting..."
    docker-compose restart media-server
else
    echo "Container is not running, starting..."
    docker-compose up -d media-server
fi