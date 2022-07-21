#!/bin/bash
set -e

#Checking postgresql server  
until PGPASSWORD=password psql -h postgresql -p 5432 -U pguser -d postgres -c '\q'; do
  >&2 echo "Postgres is unavailable - sleeping"
  sleep 1
done
>&2 echo "Postgres is up!" 
#Checking redis
until redis-cli -h redis -c "quit"; do 
  >&2 echo "Redis is unavailable - sleeping"
  sleep 1
done 
>&2 echo "Redis is up!"
#Checking mongo 
until mongosh mongodb://root:rootpass@mongo:27017 --eval "quit"; do 
  >&2 echo "Mongo is unavailable - sleeping"
  sleep 1
done 
>&2 echo "Mongo is up!"

#set up the database data initization
echo "All servers are ready"
redis-cli -h redis config set requirepass password
cat psql_commands.txt | PGPASSWORD=password psql -h postgresql -p 5432 -U pguser -d postgres
mongosh mongodb://root:rootpass@mongo:27017 -f mongodb_commands.js
echo "running testinit.sh is completed"