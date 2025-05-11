#!/bin/bash

curl -X GET \
  "http://localhost:8080/api/users/all" \
  -H "accept: application/json"

#curl -X 'POST' \
#  'http://localhost:8080/api/users' \
#  -H 'accept: application/json' \
#  -H 'Content-Type: application/json' \
#  -d '{
#  "address": "Beijing",
#  "age": 18,
#  "email": "alice01@test.com",
#  "gender": 0,
#  "name": "Alice01"
#}'