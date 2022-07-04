#!/bin/bash

#
# Starting hyprspace on NIC hs0 using tun network device.
#

#
# Make tun device
#
if [ ! -f /dev/net/tun ]
then
  mknod /dev/net/tun c 10 100
  mkdir -p /dev/net
fi

hyprspace up hs0
