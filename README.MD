# Kutta Simple Payload Sharing Server

**Kutta** is a lightweight, responsive web-based file sharing server written in **Go**.
It was created to use during our trainings to move, paylaods and code snippets across amcchines in the lab environment.
It is an alternative tool for python http.server, motivated from tools like `goshs` and `updog`, with extra features.

## Features

* **Directory Browsing** - Navigate and download files/folders
* **File Upload** - Drag & drop, file picker, or `curl` uploads
* **Deletion** - Delete uploaded files (never touches existing system files)
* **Clipboard** - Add, clear, and export snippets as JSON
* **Security** - Read-only / Upload-only / Auth modes
* **Embedded UI** - Dark theme, responsive design
* **Service Ready** - Install as a `systemd` service

---


## Installation

### One-liner (Go Install)

```bash
go install github.com/breachsimrange/kutta@latest
```

This puts the `kutta` binary in `~/go/bin/`.
Make sure `~/go/bin` is in your `$PATH`.

---

### Manual Build + Service Install

```bash
git clone https://github.com/breachsimrange/kutta.git
cd kutta
make build install service
```

This will:

* Build `kutta`
* Install it to `/usr/local/bin/kutta`
* Install and enable the **systemd service**

---

## Usage

### Start Globally (default port `13377`)

```bash
kutta
```

* Runs on `http://localhost:13377`
* **No files visible** initially (uploads only)

---

### Start on Custom Port with Directory

```bash
kutta -p 8980
```

* Runs on `http://localhost:8980`
* Full directory browsing and downloads
* Similar to python Simple HTTP Servr

---

### Read-only Mode

```bash
kutta --read-only
```

### Upload-only Mode

```bash
kutta --upload-only
```

### With Authentication (Still working on it)

```bash
kutta --auth user:password
```

### Drop Privileges

```bash
sudo kutta --user www-data
```

---

## Web UI Features

* **Upload files** - Drag and drop or click to browse
* **Browse subfolders** - When using `-p` with a directory
* **Clipboard** - Add, clear, export snippets
* **Bulk delete** - Uploaded files only
* **Responsive design** - Works on desktop and mobile

---

## Developers, come here

```bash
make build   # build kutta
make install # install binary to /usr/local/bin
make service # setup systemd service
```

---

## License
Released under the [GNU General Public License v3 (GPL-3.0)](./LICENSE).