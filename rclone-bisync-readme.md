# Rclone Bisync with Desktop Notifications

This repository contains a set of shell scripts designed to perform a two-way synchronization (`bisync`) between a local directory and a Google Drive remote using `rclone`. It includes a mechanism for sending desktop notifications for the sync status, which is especially useful when the script is run as a cron job.

## Files

- `gdrive-bisync.sh`: The main script that executes the `rclone bisync` command, handles a common synchronization error by forcing a `--resync`, and sends a desktop notification upon completion (success or failure). It includes a file-locking mechanism to prevent multiple instances from running at the same time.

- `dbus-var.sh`: A helper script that captures the necessary D-Bus session variables (`DBUS_SESSION_BUS_ADDRESS` and `DISPLAY`) from your graphical session. These variables are essential for allowing a non-graphical process, like a cron job, to send desktop notifications.

- `dbus-vars.desktop`: A `.desktop` file used to automatically run `dbus-var.sh` at your desktop session's startup. This ensures the D-Bus variables are always up-to-date and available for the main bisync script.

## Usage

1. **Set up D-Bus Variables**

This step is crucial for enabling desktop notifications.

- Place `dbus-var.sh` in a persistent location, such as your home directory (`~/dbus-var.sh`).

- Modify the `dbus-vars.desktop` file to point to the correct path of your `dbus-var.sh` script. For example, change `Exec=/path/to/your/dbus-vars.sh` to `Exec=/home/your_username/dbus-var.sh`.

- Place the modified `dbus-vars.desktop` file in your desktop environment's autostart directory, typically `~/.config/autostart/`.

- Make the `dbus-var.sh` script executable by running `chmod +x ~/dbus-var.sh`.

- Log out and log back in to your system. This will run the `dbus-var.sh` script, which will create the file `~/.config/dbus-vars` containing the required environment variables.

2. **Configure `gdrive-bisync.sh`**

Before running the main script, you'll need to customize the paths.

- Edit `gdrive-bisync.sh` and update the following variables to match your system's setup:

    - `LOG_FILE` The path for the sync log file.

    - `RCLONE_BIN` The path to your `rclone` binary.

    - `LOCAL_PATH`: The local directory you want to synchronize.

    - `REMOTE_PATH`: The name of your `rclone` remote.

- Make the `gdrive-bisync.sh` script executable: `chmod +x ~/gdrive-bisync.sh`.

3. **Schedule with Cron**

To automate the synchronization, you can add an entry to your crontab.

- Open your crontab for editing by running `crontab -e`.

- Add a line to run the script at your desired interval. For example, to run the sync every 10 minutes, you would add:

```shell
*/10 * * * * /home/your_username/gdrive-bisync.sh
```

- Save and exit the crontab editor.

With this setup, the `gdrive-bisync.sh` script will run automatically, and you will receive a desktop notification alerting you to the outcome of each sync.