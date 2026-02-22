#!/bin/bash
echo -ne "\033]0;slope breakpad.envelope\007"
export PS1="$ "
clear
echo "$ slope breakpad.envelope"
exec /tmp/slope /tmp/breakpad.envelope
