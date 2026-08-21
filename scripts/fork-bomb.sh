#!/usr/bin/env bash

# WARN: Be careful when you run this!
# defines a function called ':' which
# recursively calls itself and pipes into
# a bg process, will blow up your system
# :() { : | : & }
# :

# recurse() {
#     idx=$1
#     if ((idx == 0)); then
#         echo "last recurse"
#         return 0
#     fi
#     idx=$((idx - 1))
#     echo $idx
#     recurse $idx | recurse $idx &
# }

# recurse 40

# recurse() {
#     local idx=$1
#     local max_depth=10 # Safe limit
#
#     if ((idx <= 0)); then
#         echo "last recurse"
#         return 0
#     fi
#
#     if ((idx > max_depth)); then
#         echo "Recursion depth too deep: $idx"
#         return 1
#     fi
#
#     echo "recurse $idx"
#     idx=$((idx - 1))
#     recurse $idx
# }
#
# recurse 21
