#!/bin/sh
# Launch the installed game under a virtual X display and check that it is still running after startup, that its
# window carries the application ID as WM_CLASS, and that it created its save. Saves a screenshot for review.
# Usage: launch-test.sh "<command>" <screenshot.png> <expected save path>
set -eu
command=$1 shot=$2 save=$3
xvfb-run -a -s '-screen 0 1280x800x24' sh -euc '
  $1 > /tmp/wwhsrt.log 2>&1 &
  pid=$!
  for _ in $(seq 60); do
    xdotool search --class io.github.adsouza.wwhsrt > /dev/null 2>&1 && break
    kill -0 $pid 2> /dev/null || { cat /tmp/wwhsrt.log; echo "game exited during startup"; exit 1; }
    sleep 1
  done
  xdotool search --class io.github.adsouza.wwhsrt > /dev/null || { cat /tmp/wwhsrt.log; echo "no window with the app ID class"; exit 1; }
  sleep 8
  kill -0 $pid || { cat /tmp/wwhsrt.log; echo "game exited after opening its window"; exit 1; }
  import -window root "$2"
  kill $pid
  wait $pid || true
' launch-test "$command" "$shot"
test -s "$save" || { echo "no save at $save"; exit 1; }
echo "launch test passed: $command"
