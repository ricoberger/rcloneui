# rcloneui

rcloneui is a small terminal ui for [rclone](https://rclone.org) to view, copy and delete files from all remotes configured in your `rclone.conf` file.

## Usage

You can download rcloneui for your environment from the [releases](https://github.com/ricoberger/rcloneui/releases) page. Then extract the archive and run the `rcloneui` binary. To install the binary into your path you can use the following command:

```sh
sudo install -m 755 rcloneui /usr/local/bin/rcloneui
```

## Development

To build and and run rcloneui from source you can use the following commands:

```sh
git clone git@github.com:ricoberger/rcloneui.git
cd rcloneui

make build
./bin/rcloneui
```
