#!/bin/bash

base=$(pwd)

PLUGIN_DIR=${PLUGIN_DIR:-"/tmp/plugins/"}

build_plugin() {
  plugin=$1

  cd "$base/plugins/$plugin"
  echo "building $plugin"
  go build -o "$PLUGIN_DIR/$plugin.so" -buildmode=plugin
}

build_plugin "scheduler"
build_plugin "executor"
