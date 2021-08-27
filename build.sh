#!/bin/bash

base=$(pwd)

PLUGIN_DIR=${PLUGIN_DIR:-"/usr/lib64/sg-core"}

build_plugin() {
  plugin=$1

  cd "$base/$plugin"
  echo "building $plugin"
  go build -o "$PLUGIN_DIR/$plugin.so" -buildmode=plugin
}

build_plugin "scheduler"
