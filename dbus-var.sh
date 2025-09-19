#!/bin/bash
# This script will capture the necessary environment variables for desktop notifications
# from the user's graphical session and save them to a file.

# Check if the DBUS_SESSION_BUS_ADDRESS and DISPLAY variables are set
if [[ -n "$DBUS_SESSION_BUS_ADDRESS" && -n "$DISPLAY" ]]; then
    # Create the directory if it doesn't exist
    mkdir -p "$HOME/.config"
    # Save the variables to a file
    echo "export DBUS_SESSION_BUS_ADDRESS=\"$DBUS_SESSION_BUS_ADDRESS\"" > "$HOME/.config/dbus-vars"
    echo "export DISPLAY=\"$DISPLAY\"" >> "$HOME/.config/dbus-vars"
    echo "Environment variables saved to $HOME/.config/dbus-vars"
fi
