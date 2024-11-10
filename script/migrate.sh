#!/bin/bash

# Define the Redis prefix
PREFIX="kazan"
URI=redis://username:password@host:port

# Get all keys
keys=$(redis-cli -u $URI KEYS "gopray*")

for key in $keys; do

  # Check if the key is a set
  type=$(redis-cli -u $URI TYPE "$key")

  if [ "$type" == "set" ]; then
    # Re-add the set elements with the prefix
    members=$(redis-cli -u $URI SMEMBERS "$key")
    for member in $members; do
      redis-cli -u $URI SADD "${PREFIX}:${key}" "$member"
    done
  else
    # Re-add key-value pair with the prefix
    value=$(redis-cli GET "$key")
    updated_key="${PREFIX}:$(echo $key | sed "s/^gopray_//")"

    redis-cli -u $URI SET "${updated_key}" "$value"
  fi

  # Optional: delete the original key after migration
  redis-cli -u $URI DEL "$key"
done
