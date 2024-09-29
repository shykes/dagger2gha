#!/bin/bash

# Make sure not to load any implicit module
cd $(mktemp -d)
# Run a simple core function to "warm up" the engine
dagger core version
