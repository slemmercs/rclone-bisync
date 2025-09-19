#!/bin/bash
if [[ -n "$DBUS_SESSION_BUS_ADDRESS" && -n "$DISPLAY" ]]; then
    mkdir -p "$HOME/.config"
    echo "DBUS_SESSION_BUS_ADDRESS=$DBUS_SESSION_BUS_ADDRESS" > "$HOME/.config/dbus-vars"
    echo "DISPLAY=$DISPLAY" >> "$HOME/.config/dbus-vars"
fi
